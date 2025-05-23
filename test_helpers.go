package gql

import (
	"os"
	"sync"
	"time"

	cache "github.com/lukaszraczylo/go-simple-graphql/cache"
	logging "github.com/lukaszraczylo/go-simple-graphql/logging"
)

var (
	testCacheOnce sync.Once
	testCache     *cache.Cache
	testLogger    *logging.Logger
)

// GetTestCache returns a shared cache instance for tests to reduce resource contention
func GetTestCache() *cache.Cache {
	testCacheOnce.Do(func() {
		testCache = cache.New(5 * time.Second) // Short TTL for tests
	})
	return testCache
}

// GetTestLogger returns a shared logger instance for tests
func GetTestLogger() *logging.Logger {
	if testLogger == nil {
		testLogger = logging.New()

		// Check if LOG_LEVEL is set for tests, otherwise use LEVEL_ERROR
		logLevelStr := "error" // Default for tests
		if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
			logLevelStr = envLogLevel
		}
		logLevel := logging.GetLogLevel(logLevelStr)
		testLogger.SetMinLogLevel(logLevel)

		// Log test logger configuration for validation
		testLogger.Info(&logging.LogMessage{
			Message: "Test logger initialized",
			Pairs: map[string]interface{}{
				"LOG_LEVEL_env_var":   os.Getenv("LOG_LEVEL"),
				"effective_log_level": logLevelStr,
				"parsed_log_level":    logLevel,
				"level_name":          logging.LevelNames[logLevel],
			},
		})
	}
	return testLogger
}

// CleanupTestCache stops the shared test cache - call this in TestMain if needed
func CleanupTestCache() {
	if testCache != nil {
		testCache.Stop()
	}
}

// CreateTestClient creates a test client with shared resources
func CreateTestClient() *BaseClient {
	return &BaseClient{
		Logger:       GetTestLogger(),
		cache:        GetTestCache(),
		endpoint:     "https://example.com/graphql",
		responseType: "mapstring",
	}
}
