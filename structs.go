package gql

import (
	"context"
	"net/http"

	"github.com/lukaszraczylo/go-simple-graphql/utils/concurrency"
	"github.com/lukaszraczylo/go-simple-graphql/utils/logger"

	"github.com/allegro/bigcache"
)

type cacheStore struct {
	client  *bigcache.BigCache
	ttl     int
	enabled bool
}

type BaseClient struct {
	Logger             Logger
	LoggerWriter       Writer
	client             *http.Client
	concurrencyManager *concurrency.Pool
	endpoint           string
	responseType       string
	cache              cacheStore
	MaxGoRoutines      int
	LoggingLevel       logger.LogLevel
	LoggerColorful     bool
	validate           bool
}

type Query struct {
	context       context.Context
	variables     map[string]any `json:"variables"`
	query         string         `json:"query"`
	compiledQuery []byte         `json:"compiledQuery"`
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
