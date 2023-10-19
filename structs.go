package gql

import (
	"context"
	"net/http"

	libpack_cache "github.com/lukaszraczylo/graphql-monitoring-proxy/cache"
	libpack_logging "github.com/lukaszraczylo/graphql-monitoring-proxy/logging"
)

type cacheStore struct {
	client  *libpack_cache.Cache
	ttl     int
	enabled bool
}

type retriesConfig struct {
	enabled bool
	max     int
	delay   int
}

type BaseClient struct {
	Logger        *libpack_logging.LogConfig
	client        *http.Client
	endpoint      string
	responseType  string
	cache         cacheStore
	MaxGoRoutines int
	validate      bool
	retries       retriesConfig
}

type Query struct {
	context       context.Context
	variables     map[string]any `json:"variables"`
	query         string         `json:"query"`
	compiledQuery []byte         `json:"compiledQuery"`
}

type queryExecutor struct {
	result          queryExecutorResult
	context         context.Context
	client          *BaseClient
	headers         map[string]interface{}
	hash            string
	query           []byte
	should_cache    bool
	retries_enabled bool
}

type queryExecutorResult struct {
	data   interface{}
	errors error
}

type request struct {
	Variables any    `json:"variables"`
	Query     string `json:"query"`
}

type queryResults struct {
	Data   interface{} `json:"data"`
	Errors []struct {
		Message interface{} `json:"message"`
	} `json:"errors"`
}
