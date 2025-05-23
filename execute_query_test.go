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
		assert.Contains(err.Error(), "can't create HTTP request")
	})
}

func (suite *Tests) TestQueryExecutor_executeQuery_gzipTrailingGarbage() {
	suite.T().Run("should handle gzip trailing garbage error with fallback", func(t *testing.T) {
		// Create a test server that returns malformed gzip data (with trailing garbage)
		malformedGzipServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Encoding", "gzip")

			// Create valid gzip data first
			var buf bytes.Buffer
			gzipWriter := gzip.NewWriter(&buf)
			gzipWriter.Write([]byte(`{"data":{"viewer":{"login":"testuser"}}}`))
			gzipWriter.Close()

			// Add trailing garbage to simulate the error
			validGzipData := buf.Bytes()
			malformedData := append(validGzipData, []byte("trailing garbage data")...)

			w.WriteHeader(http.StatusOK)
			w.Write(malformedData)
		}))
		defer malformedGzipServer.Close()

		client := CreateTestClient()

		qe := &QueryExecutor{
			BaseClient: client,
			Query:      []byte(`{"query":"query { viewer { login } }"}`),
			Headers:    map[string]interface{}{"Content-Type": "application/json"},
			CacheKey:   "no-cache",
			Retries:    false,
		}
		qe.endpoint = malformedGzipServer.URL
		qe.client = malformedGzipServer.Client()
		qe.cache = client.cache

		// This should fail with gzip decompression but fallback to treating as uncompressed
		result, err := qe.executeQuery()
		// The fallback should fail because the raw data (including gzip headers) is not valid JSON
		assert.Error(err)
		assert.Nil(result)
		assert.Contains(err.Error(), "error unmarshalling HTTP response")
	})

	suite.T().Run("should handle gzip with valid JSON fallback", func(t *testing.T) {
		// Create a test server that claims gzip but returns plain JSON
		fakeGzipServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Encoding", "gzip") // Claims gzip but sends plain JSON

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data":{"viewer":{"login":"fallbackuser"}}}`))
		}))
		defer fakeGzipServer.Close()

		client := CreateTestClient()

		qe := &QueryExecutor{
			BaseClient: client,
			Query:      []byte(`{"query":"query { viewer { login } }"}`),
			Headers:    map[string]interface{}{"Content-Type": "application/json"},
			CacheKey:   "no-cache",
			Retries:    false,
		}
		qe.endpoint = fakeGzipServer.URL
		qe.client = fakeGzipServer.Client()
		qe.cache = client.cache

		// Should fail gzip decompression but succeed with fallback to plain JSON
		result, err := qe.executeQuery()
		assert.NoError(err)
		assert.NotNil(result)
		assert.Contains(string(result), "fallbackuser")
	})

	suite.T().Run("should handle corrupted gzip header", func(t *testing.T) {
		// Create a test server that returns corrupted gzip data
		corruptedGzipServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Encoding", "gzip")

			// Send corrupted gzip data (invalid magic number)
			corruptedData := []byte{0x1f, 0x8c, 0x08, 0x00} // Invalid gzip header
			corruptedData = append(corruptedData, []byte(`{"data":{"viewer":{"login":"corruptuser"}}}`)...)

			w.WriteHeader(http.StatusOK)
			w.Write(corruptedData)
		}))
		defer corruptedGzipServer.Close()

		client := CreateTestClient()

		qe := &QueryExecutor{
			BaseClient: client,
			Query:      []byte(`{"query":"query { viewer { login } }"}`),
			Headers:    map[string]interface{}{"Content-Type": "application/json"},
			CacheKey:   "no-cache",
			Retries:    false,
		}
		qe.endpoint = corruptedGzipServer.URL
		qe.client = corruptedGzipServer.Client()
		qe.cache = client.cache

		// Should fail gzip reader creation and fallback to raw data parsing
		result, err := qe.executeQuery()
		// This will likely fail because the raw data includes corrupted gzip headers
		assert.Error(err)
		assert.Nil(result)
	})

	suite.T().Run("should handle empty gzip response", func(t *testing.T) {
		// Create a test server that returns empty gzip data
		emptyGzipServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Encoding", "gzip")

			w.WriteHeader(http.StatusOK)
			// Send empty response
		}))
		defer emptyGzipServer.Close()

		client := CreateTestClient()

		qe := &QueryExecutor{
			BaseClient: client,
			Query:      []byte(`{"query":"query { viewer { login } }"}`),
			Headers:    map[string]interface{}{"Content-Type": "application/json"},
			CacheKey:   "no-cache",
			Retries:    false,
		}
		qe.endpoint = emptyGzipServer.URL
		qe.client = emptyGzipServer.Client()
		qe.cache = client.cache

		result, err := qe.executeQuery()
		assert.Error(err)
		assert.Nil(result)
		assert.Contains(err.Error(), "empty response data")
	})

	suite.T().Run("should handle partial gzip data", func(t *testing.T) {
		// Create a test server that returns partial/truncated gzip data
		partialGzipServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Encoding", "gzip")

			// Create valid gzip data but truncate it
			var buf bytes.Buffer
			gzipWriter := gzip.NewWriter(&buf)
			gzipWriter.Write([]byte(`{"data":{"viewer":{"login":"partialuser"}}}`))
			gzipWriter.Close()

			// Send only partial gzip data (truncated)
			fullData := buf.Bytes()
			partialData := fullData[:len(fullData)/2] // Send only half

			w.WriteHeader(http.StatusOK)
			w.Write(partialData)
		}))
		defer partialGzipServer.Close()

		client := CreateTestClient()

		qe := &QueryExecutor{
			BaseClient: client,
			Query:      []byte(`{"query":"query { viewer { login } }"}`),
			Headers:    map[string]interface{}{"Content-Type": "application/json"},
			CacheKey:   "no-cache",
			Retries:    false,
		}
		qe.endpoint = partialGzipServer.URL
		qe.client = partialGzipServer.Client()
		qe.cache = client.cache

		result, err := qe.executeQuery()
		// Should fail gzip decompression and fallback, but raw data won't be valid JSON
		assert.Error(err)
		assert.Nil(result)
	})

	suite.T().Run("should successfully decompress valid gzip", func(t *testing.T) {
		// Ensure our fix doesn't break valid gzip handling
		validGzipServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Encoding", "gzip")

			var buf bytes.Buffer
			gzipWriter := gzip.NewWriter(&buf)
			gzipWriter.Write([]byte(`{"data":{"viewer":{"login":"validgzipuser"}}}`))
			gzipWriter.Close()

			w.WriteHeader(http.StatusOK)
			w.Write(buf.Bytes())
		}))
		defer validGzipServer.Close()

		client := CreateTestClient()

		qe := &QueryExecutor{
			BaseClient: client,
			Query:      []byte(`{"query":"query { viewer { login } }"}`),
			Headers:    map[string]interface{}{"Content-Type": "application/json"},
			CacheKey:   "test-valid-gzip",
			Retries:    false,
		}
		qe.endpoint = validGzipServer.URL
		qe.client = validGzipServer.Client()
		qe.cache = client.cache
		qe.CacheTTL = 5 * time.Second

		result, err := qe.executeQuery()
		assert.NoError(err)
		assert.NotNil(result)
		assert.Contains(string(result), "validgzipuser")
	})
}

func (suite *Tests) TestQueryExecutor_executeQuery_gzipErrorScenarios() {
	suite.T().Run("should handle gzip reader creation failure", func(t *testing.T) {
		// Test server that sends invalid gzip magic number
		invalidMagicServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Encoding", "gzip")

			// Send data with wrong magic number (not gzip)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data":{"viewer":{"login":"magicuser"}}}`))
		}))
		defer invalidMagicServer.Close()

		client := CreateTestClient()

		qe := &QueryExecutor{
			BaseClient: client,
			Query:      []byte(`{"query":"query { viewer { login } }"}`),
			Headers:    map[string]interface{}{"Content-Type": "application/json"},
			CacheKey:   "no-cache",
			Retries:    false,
		}
		qe.endpoint = invalidMagicServer.URL
		qe.client = invalidMagicServer.Client()
		qe.cache = client.cache

		result, err := qe.executeQuery()
		// Should fail gzip reader creation, fallback to raw JSON parsing, and succeed
		assert.NoError(err)
		assert.NotNil(result)
		assert.Contains(string(result), "magicuser")
	})

	suite.T().Run("should handle gzip decompression with unexpected EOF", func(t *testing.T) {
		// Create server that sends truncated gzip stream
		eofGzipServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Encoding", "gzip")

			// Create a proper gzip header but truncate the data
			gzipHeader := []byte{0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff}
			// Add some compressed data but not complete
			incompleteData := []byte{0x4a, 0x49, 0x2c, 0x49, 0x54, 0xb2, 0x52, 0x50}

			w.WriteHeader(http.StatusOK)
			w.Write(append(gzipHeader, incompleteData...))
		}))
		defer eofGzipServer.Close()

		client := CreateTestClient()

		qe := &QueryExecutor{
			BaseClient: client,
			Query:      []byte(`{"query":"query { viewer { login } }"}`),
			Headers:    map[string]interface{}{"Content-Type": "application/json"},
			CacheKey:   "no-cache",
			Retries:    false,
		}
		qe.endpoint = eofGzipServer.URL
		qe.client = eofGzipServer.Client()
		qe.cache = client.cache

		result, err := qe.executeQuery()
		// Should fail gzip decompression and fallback, but raw data won't be valid JSON
		assert.Error(err)
		assert.Nil(result)
		assert.Contains(err.Error(), "error unmarshalling HTTP response")
	})
}
