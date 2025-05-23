package gql

import (
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func (suite *Tests) TestQueryExecutor_executeQuery() {
	suite.T().Run("should execute query successfully", func(t *testing.T) {
		client := CreateTestClient()

		qe := &QueryExecutor{
			BaseClient: client,
			Query:      []byte(`{"query":"query { viewer { login } }"}`),
			Headers:    map[string]interface{}{"Content-Type": "application/json"},
			CacheKey:   "test-key",
			Retries:    false,
		}
		qe.endpoint = mockServer.URL
		qe.client = mockServer.Client()
		qe.cache = client.cache
		qe.CacheTTL = 5 * time.Second

		result, err := qe.executeQuery()
		assert.NoError(err)
		assert.NotNil(result)
	})

	suite.T().Run("should handle HTTP errors", func(t *testing.T) {
		// Create a test server that returns an error
		errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal Server Error"))
		}))
		defer errorServer.Close()

		client := CreateTestClient()

		qe := &QueryExecutor{
			BaseClient: client,
			Query:      []byte(`{"query":"query { viewer { login } }"}`),
			Headers:    map[string]interface{}{"Content-Type": "application/json"},
			CacheKey:   "no-cache",
			Retries:    false,
		}
		qe.endpoint = errorServer.URL
		qe.client = errorServer.Client()
		qe.cache = client.cache

		result, err := qe.executeQuery()
		assert.Error(err)
		assert.Nil(result)
		assert.Contains(err.Error(), "HTTP error")
	})

	suite.T().Run("should handle GraphQL errors", func(t *testing.T) {
		// Create a test server that returns GraphQL errors
		errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"errors":[{"message":"Field not found"}]}`))
		}))
		defer errorServer.Close()

		client := CreateTestClient()

		qe := &QueryExecutor{
			BaseClient: client,
			Query:      []byte(`{"query":"query { invalidField }"}`),
			Headers:    map[string]interface{}{"Content-Type": "application/json"},
			CacheKey:   "no-cache",
			Retries:    false,
		}
		qe.endpoint = errorServer.URL
		qe.client = errorServer.Client()
		qe.cache = client.cache

		result, err := qe.executeQuery()
		assert.Error(err)
		assert.Nil(result)
		assert.Contains(err.Error(), "error executing query")
	})

	suite.T().Run("should handle no data response", func(t *testing.T) {
		// Create a test server that returns no data
		noDataServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data":null}`))
		}))
		defer noDataServer.Close()

		client := CreateTestClient()

		qe := &QueryExecutor{
			BaseClient: client,
			Query:      []byte(`{"query":"query { viewer { login } }"}`),
			Headers:    map[string]interface{}{"Content-Type": "application/json"},
			CacheKey:   "no-cache",
			Retries:    false,
		}
		qe.endpoint = noDataServer.URL
		qe.client = noDataServer.Client()
		qe.cache = client.cache

		result, err := qe.executeQuery()
		assert.Error(err)
		assert.Nil(result)
		assert.Contains(err.Error(), "no data")
	})

	suite.T().Run("should handle gzip compressed responses", func(t *testing.T) {
		// Create a test server that returns gzip compressed data
		gzipServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Encoding", "gzip")

			var buf bytes.Buffer
			gzipWriter := gzip.NewWriter(&buf)
			gzipWriter.Write([]byte(`{"data":{"viewer":{"login":"testuser"}}}`))
			gzipWriter.Close()

			w.WriteHeader(http.StatusOK)
			w.Write(buf.Bytes())
		}))
		defer gzipServer.Close()

		client := CreateTestClient()

		qe := &QueryExecutor{
			BaseClient: client,
			Query:      []byte(`{"query":"query { viewer { login } }"}`),
			Headers:    map[string]interface{}{"Content-Type": "application/json"},
			CacheKey:   "test-gzip",
			Retries:    false,
		}
		qe.endpoint = gzipServer.URL
		qe.client = gzipServer.Client()
		qe.cache = client.cache
		qe.CacheTTL = 5 * time.Second

		result, err := qe.executeQuery()
		assert.NoError(err)
		assert.NotNil(result)
		assert.Contains(string(result), "testuser")
	})

	suite.T().Run("should handle retries", func(t *testing.T) {
		attempts := 0
		retryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts < 3 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data":{"viewer":{"login":"retryuser"}}}`))
		}))
		defer retryServer.Close()

		client := CreateTestClient()

		qe := &QueryExecutor{
			BaseClient: client,
			Query:      []byte(`{"query":"query { viewer { login } }"}`),
			Headers:    map[string]interface{}{"Content-Type": "application/json"},
			CacheKey:   "no-cache",
			Retries:    true,
		}
		qe.endpoint = retryServer.URL
		qe.client = retryServer.Client()
		qe.cache = client.cache
		qe.retries_number = 5
		qe.retries_delay = 100 * time.Millisecond

		result, err := qe.executeQuery()
		assert.NoError(err)
		assert.NotNil(result)
		assert.Contains(string(result), "retryuser")
		assert.Equal(3, attempts) // Should have retried twice before succeeding
	})

	suite.T().Run("should use default client when none provided", func(t *testing.T) {
		client := CreateTestClient()

		qe := &QueryExecutor{
			BaseClient: client,
			Query:      []byte(`{"query":"query { viewer { login } }"}`),
			Headers:    map[string]interface{}{"Content-Type": "application/json"},
			CacheKey:   "no-cache",
			Retries:    false,
		}
		qe.endpoint = mockServer.URL
		qe.client = nil // No client provided
		qe.cache = client.cache

		result, err := qe.executeQuery()
		assert.NoError(err)
		assert.NotNil(result)
	})

	suite.T().Run("should set default Content-Type header", func(t *testing.T) {
		var receivedHeaders http.Header
		headerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedHeaders = r.Header
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data":{"viewer":{"login":"headeruser"}}}`))
		}))
		defer headerServer.Close()

		client := CreateTestClient()

		qe := &QueryExecutor{
			BaseClient: client,
			Query:      []byte(`{"query":"query { viewer { login } }"}`),
			Headers:    map[string]interface{}{}, // No Content-Type header
			CacheKey:   "no-cache",
			Retries:    false,
		}
		qe.endpoint = headerServer.URL
		qe.client = headerServer.Client()
		qe.cache = client.cache

		result, err := qe.executeQuery()
		assert.NoError(err)
		assert.NotNil(result)
		assert.Equal("application/json", receivedHeaders.Get("Content-Type"))
	})

	suite.T().Run("should handle invalid JSON response", func(t *testing.T) {
		invalidJSONServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`invalid json response`))
		}))
		defer invalidJSONServer.Close()

		client := CreateTestClient()

		qe := &QueryExecutor{
			BaseClient: client,
			Query:      []byte(`{"query":"query { viewer { login } }"}`),
			Headers:    map[string]interface{}{"Content-Type": "application/json"},
			CacheKey:   "no-cache",
			Retries:    false,
		}
		qe.endpoint = invalidJSONServer.URL
		qe.client = invalidJSONServer.Client()
		qe.cache = client.cache

		result, err := qe.executeQuery()
		assert.Error(err)
		assert.Nil(result)
		assert.Contains(err.Error(), "error unmarshalling HTTP response")
	})

	suite.T().Run("should cache successful responses", func(t *testing.T) {
		client := CreateTestClient()

		qe := &QueryExecutor{
			BaseClient: client,
			Query:      []byte(`{"query":"query { viewer { login } }"}`),
			Headers:    map[string]interface{}{"Content-Type": "application/json"},
			CacheKey:   "cache-test-key",
			Retries:    false,
		}
		qe.endpoint = mockServer.URL
		qe.client = mockServer.Client()
		qe.cache = client.cache
		qe.CacheTTL = 5 * time.Second

		result, err := qe.executeQuery()
		assert.NoError(err)
		assert.NotNil(result)

		// Check that the result was cached
		cachedResult := client.cacheLookup("cache-test-key")
		assert.NotNil(cachedResult)
		assert.Equal(result, cachedResult)
	})
}

func (suite *Tests) TestQueryExecutor_executeQuery_requestCreationError() {
	suite.T().Run("should handle request creation errors", func(t *testing.T) {
		client := CreateTestClient()

		qe := &QueryExecutor{
			BaseClient: client,
			Query:      []byte(`{"query":"query { viewer { login } }"}`),
			Headers:    map[string]interface{}{"Content-Type": "application/json"},
			CacheKey:   "no-cache",
			Retries:    false,
		}
		qe.endpoint = "://invalid-url" // Invalid URL to trigger error
		qe.cache = client.cache

		result, err := qe.executeQuery()
		assert.Error(err)
		assert.Nil(result)
		assert.Contains(err.Error(), "Can't create HTTP request")
	})
}
