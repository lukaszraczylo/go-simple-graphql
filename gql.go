// Package go-simple-graphql is a wrapper for GraphQL queries execution.
// It's sole purpose is to allow developer to execute queries without specific fields / types mapping
// which can be really confusing and even impossible whilst dealing with Hasura custom generated types.
//
// Library supports advanced error reporting on unsuccessful queries.
// Library also uses HTTP2 for communication with GraphQL API if it's supported by server

// Environment variables:
// GRAPHQL_ENDPOINT - GraphQL endpoint to use. Default: http://127.0.0.1:9090/v1/graphql
// LOG_LEVEL - Log level to use. Default: INFO
//	 					 Available log levels: DEBUG, INFO, WARN, ERROR, FATAL

package gql

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	bigcache "github.com/allegro/bigcache/v3"
	jsoniter "github.com/json-iterator/go"
	"github.com/lukaszraczylo/go-simple-graphql/pkg/logging"
	retry "github.com/sethvargo/go-retry"
	"golang.org/x/net/http2"
)

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

type GraphQL struct {
	BackoffSetup  retry.Backoff
	HttpClient    *http.Client
	Log           *logging.LogConfig
	CacheStore    *bigcache.BigCache
	Endpoint      string
	RetriesNumber int
	RetriesDelay  time.Duration
	Cache         bool
	RetriesEnable bool
}

func pickGraphqlEndpoint() (graphqlEndpoint string) {
	value, present := os.LookupEnv("GRAPHQL_ENDPOINT")
	if present && value != "" {
		graphqlEndpoint = value
	} else {
		graphqlEndpoint = "http://127.0.0.1:9090/v1/graphql"
		fmt.Println("Setting default endpoint", graphqlEndpoint)
	}
	return graphqlEndpoint
}

func setCacheTTL() int {
	value, present := os.LookupEnv("GRAPHQL_CACHE_TTL")
	if present && value != "" {
		i, err := strconv.Atoi(value)
		if err != nil {
			panic("Invalid value for query cache ttl")
		}
		return int(i)
	} else {
		return 5
	}
}

func setCacheEnabled() bool {
	value, present := os.LookupEnv("GRAPHQL_CACHE")
	if present && value != "" {
		b, err := strconv.ParseBool(value)
		if err != nil {
			panic("Invalid value for query cache")
		}
		return b
	} else {
		return false
	}
}

func NewConnection() *GraphQL {

	endpoint := pickGraphqlEndpoint()
	var httpClient *http.Client
	if strings.HasPrefix(endpoint, "http://") {
		// HTTP/1.1 client
		httpClient = &http.Client{}
	} else {
		// HTTP/2 or HTTPS client
		http2Transport := &http2.Transport{
			AllowHTTP: true,
		}
		if strings.HasPrefix(endpoint, "https://") {
			http2Transport.TLSClientConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
		}
		httpClient = &http.Client{
			Transport: http2Transport,
		}
	}

	g := GraphQL{
		Endpoint:      endpoint,
		HttpClient:    httpClient,
		Log:           logging.NewLogger(),
		Cache:         setCacheEnabled(),
		CacheStore:    setupCache(),
		RetriesEnable: false,
		RetriesNumber: 1,
		RetriesDelay:  250,
		BackoffSetup:  nil,
	}
	var err error
	retriesEnable, retriesEnableExists := os.LookupEnv("RETRIES_ENABLE")
	if retriesEnableExists {
		g.RetriesEnable, err = strconv.ParseBool(retriesEnable)
		if err != nil {
			panic("Invalid value for RETRIES_ENABLE")
		}
		retriesNumber, retriesNumberExists := os.LookupEnv("RETRIES_NUMBER")
		if retriesNumberExists {
			g.RetriesNumber, err = strconv.Atoi(retriesNumber)
			if err != nil {
				panic("Invalid value for RETRIES_NUMBER")
			}
		}
		retriesDelay, retriesDelayExists := os.LookupEnv("RETRIES_DELAY")
		if retriesDelayExists {
			retDel, err := strconv.Atoi(retriesDelay)
			if err != nil {
				panic("Invalid value for RETRIES_DELAY")
			}
			g.RetriesDelay = time.Duration(retDel) * time.Millisecond
		}
	}
	g.BackoffSetup = retry.NewExponential(g.RetriesDelay)
	g.BackoffSetup = retry.WithMaxRetries(uint64(g.RetriesNumber), g.BackoffSetup)

	return &g
}

func setupCache() *bigcache.BigCache {
	cache, err := bigcache.NewBigCache(bigcache.DefaultConfig(time.Duration(setCacheTTL()) * time.Second))
	if err != nil {
		panic("Error creating cache: " + err.Error())
	}
	return cache
}
