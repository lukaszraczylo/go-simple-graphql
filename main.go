package gql

import (
	"net/http"
	"strings"
	"time"

	"github.com/gookit/goutil/envutil"
	cache "github.com/lukaszraczylo/go-simple-graphql/cache"
	logging "github.com/lukaszraczylo/go-simple-graphql/logging"
)

// parseRetryPatterns parses a comma-separated string of retry patterns into a slice
func parseRetryPatterns(patterns string) []string {
	if patterns == "" {
		return []string{}
	}
	parts := strings.Split(patterns, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func NewConnection() (b *BaseClient) {
	// Read LOG_LEVEL environment variable
	logLevelStr := envutil.Getenv("LOG_LEVEL", "info")
	logLevel := logging.GetLogLevel(logLevelStr)

	// Create logger and set the log level from environment
	logger := logging.New()
	logger.SetMinLogLevel(logLevel)

	// Log the LOG_LEVEL configuration for validation
	logger.Info(&logging.LogMessage{
		Message: "Logger initialized with LOG_LEVEL configuration",
		Pairs: map[string]interface{}{
			"LOG_LEVEL_env_var": logLevelStr,
			"parsed_log_level":  logLevel,
			"level_name":        logging.LevelNames[logLevel],
			"default_min_level": logging.LEVEL_INFO,
		},
	})

	b = &BaseClient{
		endpoint:             envutil.Getenv("GRAPHQL_ENDPOINT", "https://api.github.com/graphql"),
		responseType:         envutil.Getenv("GRAPHQL_OUTPUT", "string"),
		Logger:               logger,
		cache:                cache.New(time.Duration(envutil.GetInt("GRAPHQL_CACHE_TTL", 5)) * time.Second),
		cache_global:         envutil.GetBool("GRAPHQL_CACHE_ENABLED", false),
		retries_enable:       envutil.GetBool("GRAPHQL_RETRIES_ENABLE", false),
		retries_delay:        time.Duration(envutil.GetInt("GRAPHQL_RETRIES_DELAY", 250) * int(time.Millisecond)),
		retries_number:       envutil.GetInt("GRAPHQL_RETRIES_NUMBER", 3),
		retries_patterns:     parseRetryPatterns(envutil.Getenv("GRAPHQL_RETRIES_PATTERNS", "postgres,connection,timeout,transaction,could not,temporarily unavailable,deadlock")),
		minify_queries:       envutil.GetBool("GRAPHQL_MINIFY_QUERIES", true), // Default: enabled for production efficiency
		pool_warmup_enabled:  envutil.GetBool("GRAPHQL_POOL_WARMUP_ENABLED", false),
		pool_size:            envutil.GetInt("GRAPHQL_POOL_SIZE", 5),
		pool_warmup_query:    envutil.Getenv("GRAPHQL_POOL_WARMUP_QUERY", "query{__typename}"),
		pool_health_interval: time.Duration(envutil.GetInt("GRAPHQL_POOL_HEALTH_INTERVAL", 30)) * time.Second,
		pool_stop:            make(chan bool, 1),
	}
	b.client = b.createHttpClient()
	b.Logger.Debug(&logging.LogMessage{
		Message: "Created new GraphQL client connection",
		Pairs: map[string]interface{}{
			"endpoint":          b.endpoint,
			"pool_warmup":       b.pool_warmup_enabled,
			"pool_size":         b.pool_size,
			"pool_health_check": b.pool_health_interval,
		},
	})

	// Initialize connection pool if warmup is enabled
	if b.pool_warmup_enabled {
		b.warmupConnectionPool()
		b.startPoolHealthMonitor()
	}

	return b
}

func (b *BaseClient) SetEndpoint(endpoint string) {
	b.endpoint = endpoint
}

func (b *BaseClient) SetOutput(responseType string) {
	// allowed are byte, string, mapstring
	// check if responseType is allowed
	// TODO: implement
	b.responseType = responseType
}

func (b *BaseClient) SetHTTPClient(client *http.Client) {
	b.client = client
}

func (b *BaseClient) SetQueryMinification(enabled bool) {
	b.minify_queries = enabled
	b.Logger.Debug(&logging.LogMessage{
		Message: "GraphQL query minification setting updated",
		Pairs: map[string]interface{}{
			"minify_queries": enabled,
		},
	})
}
