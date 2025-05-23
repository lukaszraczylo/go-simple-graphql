package gql

import (
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
	}
	b.client = b.createHttpClient()
	b.Logger.Debug(&logging.LogMessage{
		Message: "Created new GraphQL client connection",
		Pairs: map[string]interface{}{
			"values": b,
		},
	})
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
