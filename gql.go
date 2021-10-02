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
	"time"

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

func NewConnection() *GraphQL {
	return &GraphQL{
		Endpoint: pickGraphqlEndpoint(),
		HttpClient: &http.Client{
			Timeout: time.Second * 10,
			Transport: &http2.Transport{
				DisableCompression: false,
				AllowHTTP:          true,
				DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
					return net.Dial(network, addr)
				},
			},
		},
		Log: logging.NewLogger(),
	}
}
