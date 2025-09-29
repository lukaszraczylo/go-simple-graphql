package gql

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gookit/goutil/envutil"
	cache "github.com/lukaszraczylo/go-simple-graphql/cache"
	logging "github.com/lukaszraczylo/go-simple-graphql/logging"
)

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
		endpoint:       envutil.Getenv("GRAPHQL_ENDPOINT", "https://api.github.com/graphql"),
		responseType:   envutil.Getenv("GRAPHQL_OUTPUT", "string"),
		Logger:         logger,
		cache:          cache.New(time.Duration(envutil.GetInt("GRAPHQL_CACHE_TTL", 5)) * time.Second),
		cache_global:   envutil.GetBool("GRAPHQL_CACHE_ENABLED", false),
		retries_enable: envutil.GetBool("GRAPHQL_RETRIES_ENABLE", false),
		retries_delay:  time.Duration(envutil.GetInt("GRAPHQL_RETRIES_DELAY", 250) * int(time.Millisecond)),
		retries_number: envutil.GetInt("GRAPHQL_RETRIES_NUMBER", 3),
		minify_queries: envutil.GetBool("GRAPHQL_MINIFY_QUERIES", true), // Default: enabled for production efficiency
	}
	client, err := b.createHttpClient()
	if err != nil {
		b.Logger.Critical(&logging.LogMessage{
			Message: "Failed to create HTTP client",
			Pairs: map[string]interface{}{
				"error": err.Error(),
			},
		})
		// For backward compatibility, we'll still return the client but it won't work
		return b
	}
	b.client = client
	b.Logger.Debug(&logging.LogMessage{
		Message: "Created new GraphQL client connection",
		Pairs: map[string]interface{}{
			"values": b,
		},
	})
	return b
}

func (b *BaseClient) SetEndpoint(endpoint string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.endpoint = endpoint
}

func (b *BaseClient) SetOutput(responseType string) error {
	// Validate responseType - allowed values are "byte", "string", "mapstring"
	allowedTypes := map[string]bool{
		"byte":      true,
		"string":    true,
		"mapstring": true,
	}

	if !allowedTypes[responseType] {
		return fmt.Errorf("invalid response type: %s. Allowed values are: byte, string, mapstring", responseType)
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	b.responseType = responseType

	b.Logger.Debug(&logging.LogMessage{
		Message: "Response type updated",
		Pairs: map[string]interface{}{
			"response_type": responseType,
		},
	})

	return nil
}

func (b *BaseClient) SetHTTPClient(client *http.Client) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.client = client
}

func (b *BaseClient) SetQueryMinification(enabled bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.minify_queries = enabled
	b.Logger.Debug(&logging.LogMessage{
		Message: "GraphQL query minification setting updated",
		Pairs: map[string]interface{}{
			"minify_queries": enabled,
		},
	})
}
