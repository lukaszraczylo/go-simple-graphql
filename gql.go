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
	"time"

	"github.com/allegro/bigcache/v3"
	jsoniter "github.com/json-iterator/go"
	"github.com/lukaszraczylo/go-simple-graphql/pkg/logging"
	"golang.org/x/net/http2"
)

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

type GraphQL struct {
	Endpoint      string
	HttpClient    *http.Client
	Log           *logging.LogConfig
	Cache         bool // Enable caching for read queries
	CacheStore    *bigcache.BigCache
	RetriesEnable bool
	RetriesNumber int
	RetriesDelay  int
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
	g := GraphQL{
		Endpoint: pickGraphqlEndpoint(),
		HttpClient: &http.Client{
			Transport: &http2.Transport{
				ReadIdleTimeout:    30 * time.Second,
				DisableCompression: false,
				AllowHTTP:          true,
				PingTimeout:        10 * time.Second,
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
		Log:           logging.NewLogger(),
		Cache:         setCacheEnabled(),
		CacheStore:    setupCache(),
		RetriesEnable: false,
		RetriesNumber: 1,
		RetriesDelay:  250,
	}
	var err error
	retriesEnable, retriesEnableExists := os.LookupEnv("RETRIES_ENABLE")
	if retriesEnableExists {
		g.RetriesEnable, err = strconv.ParseBool(retriesEnable)
		if err != nil {
			panic("Invalid value for RETRIES_ENABLE")
		}
		retriesNumber, retriesNumberExists := os.LookupEnv("RETRIES_NUMBER")
		if !retriesNumberExists {
			panic("RETRIES_NUMBER environment variable is not set but RETRIES_ENABLE is")
		}
		g.RetriesNumber, err = strconv.Atoi(retriesNumber)
		if err != nil {
			panic("Invalid value for RETRIES_NUMBER")
		}
		retriesDelay, retriesDelayExists := os.LookupEnv("RETRIES_DELAY")
		if !retriesDelayExists {
			panic("RETRIES_DELAY environment variable is not set but RETRIES_ENABLE is")
		}
		g.RetriesDelay, err = strconv.Atoi(retriesDelay)
	}
	return &g
}

func setupCache() *bigcache.BigCache {
	cache, err := bigcache.NewBigCache(bigcache.DefaultConfig(time.Duration(setCacheTTL()) * time.Second))
	if err != nil {
		panic("Error creating cache: " + err.Error())
	}
	return cache
}
