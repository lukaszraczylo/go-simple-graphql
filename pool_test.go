package gql

import (
	"os"
	"testing"
	"time"
)

func (suite *Tests) TestPoolWarmup() {
	suite.T().Run("should create client with pool warmup disabled by default", func(t *testing.T) {
		// Save original env vars
		originalPoolEnabled := os.Getenv("GRAPHQL_POOL_WARMUP_ENABLED")
		originalEndpoint := os.Getenv("GRAPHQL_ENDPOINT")
		defer func() {
			os.Setenv("GRAPHQL_POOL_WARMUP_ENABLED", originalPoolEnabled)
			os.Setenv("GRAPHQL_ENDPOINT", originalEndpoint)
		}()

		os.Setenv("GRAPHQL_POOL_WARMUP_ENABLED", "false")
		os.Setenv("GRAPHQL_ENDPOINT", mockServer.URL)

		client := NewConnection()
		defer client.StopPoolMonitor()

		assert.False(client.pool_warmup_enabled)
		assert.Equal(5, client.pool_size) // Default size
	})

	suite.T().Run("should warmup pool when enabled", func(t *testing.T) {
		originalPoolEnabled := os.Getenv("GRAPHQL_POOL_WARMUP_ENABLED")
		originalPoolSize := os.Getenv("GRAPHQL_POOL_SIZE")
		originalEndpoint := os.Getenv("GRAPHQL_ENDPOINT")
		defer func() {
			os.Setenv("GRAPHQL_POOL_WARMUP_ENABLED", originalPoolEnabled)
			os.Setenv("GRAPHQL_POOL_SIZE", originalPoolSize)
			os.Setenv("GRAPHQL_ENDPOINT", originalEndpoint)
		}()

		os.Setenv("GRAPHQL_POOL_WARMUP_ENABLED", "true")
		os.Setenv("GRAPHQL_POOL_SIZE", "3")
		os.Setenv("GRAPHQL_ENDPOINT", mockServer.URL)

		startTime := time.Now()
		client := NewConnection()
		defer client.StopPoolMonitor()

		duration := time.Since(startTime)

		assert.True(client.pool_warmup_enabled)
		assert.Equal(3, client.pool_size)
		// Warmup should complete relatively quickly for 3 connections
		assert.Less(duration.Seconds(), 5.0, "Pool warmup took too long")
	})

	suite.T().Run("should use custom warmup query", func(t *testing.T) {
		originalPoolEnabled := os.Getenv("GRAPHQL_POOL_WARMUP_ENABLED")
		originalQuery := os.Getenv("GRAPHQL_POOL_WARMUP_QUERY")
		originalEndpoint := os.Getenv("GRAPHQL_ENDPOINT")
		defer func() {
			os.Setenv("GRAPHQL_POOL_WARMUP_ENABLED", originalPoolEnabled)
			os.Setenv("GRAPHQL_POOL_WARMUP_QUERY", originalQuery)
			os.Setenv("GRAPHQL_ENDPOINT", originalEndpoint)
		}()

		os.Setenv("GRAPHQL_POOL_WARMUP_ENABLED", "false") // Don't actually warmup, just test configuration
		os.Setenv("GRAPHQL_POOL_WARMUP_QUERY", "query{__schema{queryType{name}}}")
		os.Setenv("GRAPHQL_ENDPOINT", mockServer.URL)

		client := NewConnection()
		defer client.StopPoolMonitor()

		assert.Equal("query{__schema{queryType{name}}}", client.pool_warmup_query)
	})
}

func (suite *Tests) TestPoolHealthMonitor() {
	suite.T().Run("should stop pool monitor gracefully", func(t *testing.T) {
		client := CreateTestClient()
		client.pool_warmup_enabled = false

		// Should not panic
		client.StopPoolMonitor()

		assert.True(true, "Handled stop gracefully when monitor not running")
	})

	suite.T().Run("should handle multiple stop calls", func(t *testing.T) {
		originalPoolEnabled := os.Getenv("GRAPHQL_POOL_WARMUP_ENABLED")
		originalPoolSize := os.Getenv("GRAPHQL_POOL_SIZE")
		originalEndpoint := os.Getenv("GRAPHQL_ENDPOINT")
		defer func() {
			os.Setenv("GRAPHQL_POOL_WARMUP_ENABLED", originalPoolEnabled)
			os.Setenv("GRAPHQL_POOL_SIZE", originalPoolSize)
			os.Setenv("GRAPHQL_ENDPOINT", originalEndpoint)
		}()

		os.Setenv("GRAPHQL_POOL_WARMUP_ENABLED", "true")
		os.Setenv("GRAPHQL_POOL_SIZE", "2")
		os.Setenv("GRAPHQL_ENDPOINT", mockServer.URL)

		client := NewConnection()

		// Call stop multiple times - should not panic
		client.StopPoolMonitor()
		client.StopPoolMonitor()
		client.StopPoolMonitor()

		assert.True(true, "Handled multiple stop calls gracefully")
	})
}

func (suite *Tests) TestPoolHealthCheck() {
	suite.T().Run("should perform health check without panic", func(t *testing.T) {
		client := CreateTestClient()
		client.pool_warmup_enabled = true
		client.pool_size = 2
		client.pool_warmup_query = "query{__typename}"

		// Perform a health check - should not panic
		client.performPoolHealthCheck()

		assert.True(true, "Health check completed")
	})

	suite.T().Run("should refresh pool without panic", func(t *testing.T) {
		client := CreateTestClient()
		client.pool_size = 2
		client.pool_warmup_query = "query{__typename}"

		// Perform pool refresh - should not panic
		client.refreshConnectionPool()

		assert.True(true, "Pool refresh completed")
	})
}
