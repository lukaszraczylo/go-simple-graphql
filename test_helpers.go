package gql

import (
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
		testLogger.SetMinLogLevel(logging.LEVEL_ERROR)
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
