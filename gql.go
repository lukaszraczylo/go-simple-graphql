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
	"net"
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
	Endpoint   string
	HttpClient *http.Client
	Log        *logging.LogConfig
	Cache      bool // Enable caching for read queries
	CacheStore *bigcache.BigCache
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
	return &GraphQL{
		Endpoint: pickGraphqlEndpoint(),
		HttpClient: &http.Client{
			Transport: &http2.Transport{
				DisableCompression: false,
				AllowHTTP:          true,
				PingTimeout:        5 * time.Second,
				DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
					return net.Dial(network, addr)
				},
			},
		},
		Log:        logging.NewLogger(),
		Cache:      setCacheEnabled(),
		CacheStore: setupCache(),
	}
}

func setupCache() *bigcache.BigCache {
	cache, err := bigcache.NewBigCache(bigcache.DefaultConfig(time.Duration(setCacheTTL()) * time.Second))
	if err != nil {
		panic("Error creating cache: " + err.Error())
	}
	return cache
}
