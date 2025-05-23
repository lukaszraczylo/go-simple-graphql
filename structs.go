package gql

import (
	"net/http"
	"time"

	cache "github.com/lukaszraczylo/go-simple-graphql/cache"
	logging "github.com/lukaszraczylo/go-simple-graphql/logging"
)

type BaseClient struct {
	cache          *cache.Cache
	Logger         *logging.Logger
	client         *http.Client
	endpoint       string
	responseType   string
	retries_delay  time.Duration
	retries_number int
	MaxGoRoutines  int
	cache_global   bool
	retries_enable bool
	minify_queries bool // Enable GraphQL query minification (default: true)
}

type Query struct {
	Variables map[string]interface{} `json:"variables,omitempty"`
	Query     string                 `json:"query,omitempty"`
	JsonQuery []byte                 `json:"-"` // Exclude from JSON serialization to prevent nested encoding
}

type QueryExecutor struct {
	Result any
	Error  error
	*BaseClient
	Headers  map[string]interface{}
	CacheKey string
	Query    []byte
	CacheTTL time.Duration
	Retries  bool
}

type queryResults struct {
	Data   interface{} `json:"data"`
	Errors []struct {
		Message interface{} `json:"message"`
	} `json:"errors"`
}
