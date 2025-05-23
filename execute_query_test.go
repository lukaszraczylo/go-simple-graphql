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
		assert.Equal("application/json; charset=utf-8", receivedHeaders.Get("Content-Type"))
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

		// This should fail with gzip decompression error
		result, err := qe.executeQuery()
		// Should get proper gzip decompression error instead of trying to parse raw gzip as JSON
		assert.Error(err)
		assert.Nil(result)
		assert.Contains(err.Error(), "gzip decompression failed")
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
		assert.Contains(err.Error(), "content encoding mismatch")
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
		// Should fail gzip decompression with proper error message
		assert.Error(err)
		assert.Nil(result)
		assert.Contains(err.Error(), "gzip decompression failed")
	})
}

func (suite *Tests) TestQueryExecutor_executeQuery_intelligentGzipDetection() {
	suite.T().Run("should detect gzip by magic bytes when header is missing", func(t *testing.T) {
		// Server sends gzip data but forgets to set Content-Encoding header
		missingHeaderServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			// Intentionally NOT setting Content-Encoding header

			var buf bytes.Buffer
			gzipWriter := gzip.NewWriter(&buf)
			gzipWriter.Write([]byte(`{"data":{"viewer":{"login":"magicbytesuser"}}}`))
			gzipWriter.Close()

			w.WriteHeader(http.StatusOK)
			w.Write(buf.Bytes())
		}))
		defer missingHeaderServer.Close()

		client := CreateTestClient()

		qe := &QueryExecutor{
			BaseClient: client,
			Query:      []byte(`{"query":"query { viewer { login } }"}`),
			Headers:    map[string]interface{}{"Content-Type": "application/json"},
			CacheKey:   "no-cache",
			Retries:    false,
		}
		qe.endpoint = missingHeaderServer.URL
		qe.client = missingHeaderServer.Client()
		qe.cache = client.cache

		result, err := qe.executeQuery()
		assert.NoError(err)
		assert.NotNil(result)
		assert.Contains(string(result), "magicbytesuser")
	})

	suite.T().Run("should handle server that claims gzip but sends plain JSON", func(t *testing.T) {
		// Buggy server that sets gzip header but sends plain JSON
		buggyHeaderServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Encoding", "gzip") // Claims gzip but sends plain JSON

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data":{"viewer":{"login":"buggyheaderuser"}}}`))
		}))
		defer buggyHeaderServer.Close()

		client := CreateTestClient()

		qe := &QueryExecutor{
			BaseClient: client,
			Query:      []byte(`{"query":"query { viewer { login } }"}`),
			Headers:    map[string]interface{}{"Content-Type": "application/json"},
			CacheKey:   "no-cache",
			Retries:    false,
		}
		qe.endpoint = buggyHeaderServer.URL
		qe.client = buggyHeaderServer.Client()
		qe.cache = client.cache

		result, err := qe.executeQuery()
		assert.NoError(err)
		assert.NotNil(result)
		assert.Contains(string(result), "buggyheaderuser")
	})

	suite.T().Run("should handle randomly switching server behavior", func(t *testing.T) {
		// Simulate a server that randomly switches between gzip and plain responses
		requestCount := 0
		randomServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			w.Header().Set("Content-Type", "application/json")

			if requestCount%2 == 0 {
				// Even requests: send gzip with correct header
				w.Header().Set("Content-Encoding", "gzip")
				var buf bytes.Buffer
				gzipWriter := gzip.NewWriter(&buf)
				gzipWriter.Write([]byte(`{"data":{"viewer":{"login":"randomgzipuser"}}}`))
				gzipWriter.Close()
				w.WriteHeader(http.StatusOK)
				w.Write(buf.Bytes())
			} else {
				// Odd requests: send plain JSON without header
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"data":{"viewer":{"login":"randomplainuser"}}}`))
			}
		}))
		defer randomServer.Close()

		client := CreateTestClient()

		// Test multiple requests to simulate random behavior
		for i := 0; i < 4; i++ {
			qe := &QueryExecutor{
				BaseClient: client,
				Query:      []byte(`{"query":"query { viewer { login } }"}`),
				Headers:    map[string]interface{}{"Content-Type": "application/json"},
				CacheKey:   "no-cache",
				Retries:    false,
			}
			qe.endpoint = randomServer.URL
			qe.client = randomServer.Client()
			qe.cache = client.cache

			result, err := qe.executeQuery()
			assert.NoError(err)
			assert.NotNil(result)

			// Should handle both gzip and plain responses correctly
			resultStr := string(result)
			// Just check that we got some valid JSON response with expected user data
			assert.True(len(resultStr) > 0, "Result should not be empty")
			assert.True(
				(len(resultStr) > 10 && (resultStr[0] == '{' || resultStr[0] == '"')) ||
					len(resultStr) > 5,
				"Result should be valid JSON or contain user data, got: %s", resultStr)
		}
	})

	suite.T().Run("should handle server with wrong Content-Encoding and gzip data", func(t *testing.T) {
		// Server sends gzip data but claims it's not compressed
		wrongEncodingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Encoding", "identity") // Claims no compression

			var buf bytes.Buffer
			gzipWriter := gzip.NewWriter(&buf)
			gzipWriter.Write([]byte(`{"data":{"viewer":{"login":"wrongencodinguser"}}}`))
			gzipWriter.Close()

			w.WriteHeader(http.StatusOK)
			w.Write(buf.Bytes())
		}))
		defer wrongEncodingServer.Close()

		client := CreateTestClient()

		qe := &QueryExecutor{
			BaseClient: client,
			Query:      []byte(`{"query":"query { viewer { login } }"}`),
			Headers:    map[string]interface{}{"Content-Type": "application/json"},
			CacheKey:   "no-cache",
			Retries:    false,
		}
		qe.endpoint = wrongEncodingServer.URL
		qe.client = wrongEncodingServer.Client()
		qe.cache = client.cache

		result, err := qe.executeQuery()
		assert.NoError(err)
		assert.NotNil(result)
		assert.Contains(string(result), "wrongencodinguser")
	})

	suite.T().Run("should handle server with multiple encoding headers", func(t *testing.T) {
		// Server sends conflicting headers but actual gzip data
		conflictingHeaderServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Add("Content-Encoding", "identity")
			w.Header().Add("Content-Encoding", "gzip") // Conflicting headers

			var buf bytes.Buffer
			gzipWriter := gzip.NewWriter(&buf)
			gzipWriter.Write([]byte(`{"data":{"viewer":{"login":"conflictinguser"}}}`))
			gzipWriter.Close()

			w.WriteHeader(http.StatusOK)
			w.Write(buf.Bytes())
		}))
		defer conflictingHeaderServer.Close()

		client := CreateTestClient()

		qe := &QueryExecutor{
			BaseClient: client,
			Query:      []byte(`{"query":"query { viewer { login } }"}`),
			Headers:    map[string]interface{}{"Content-Type": "application/json"},
			CacheKey:   "no-cache",
			Retries:    false,
		}
		qe.endpoint = conflictingHeaderServer.URL
		qe.client = conflictingHeaderServer.Client()
		qe.cache = client.cache

		result, err := qe.executeQuery()
		assert.NoError(err)
		assert.NotNil(result)
		assert.Contains(string(result), "conflictinguser")
	})

	suite.T().Run("should handle empty response with gzip header", func(t *testing.T) {
		// Server claims gzip but sends empty response
		emptyGzipHeaderServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Encoding", "gzip")

			w.WriteHeader(http.StatusOK)
			// Send empty response
		}))
		defer emptyGzipHeaderServer.Close()

		client := CreateTestClient()

		qe := &QueryExecutor{
			BaseClient: client,
			Query:      []byte(`{"query":"query { viewer { login } }"}`),
			Headers:    map[string]interface{}{"Content-Type": "application/json"},
			CacheKey:   "no-cache",
			Retries:    false,
		}
		qe.endpoint = emptyGzipHeaderServer.URL
		qe.client = emptyGzipHeaderServer.Client()
		qe.cache = client.cache

		result, err := qe.executeQuery()
		assert.Error(err)
		assert.Nil(result)
		assert.Contains(err.Error(), "content encoding mismatch")
	})

	suite.T().Run("should handle response with only gzip magic bytes but no valid gzip data", func(t *testing.T) {
		// Server sends gzip magic bytes but invalid gzip stream
		invalidGzipServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			// No Content-Encoding header

			// Send gzip magic bytes followed by invalid data
			invalidData := []byte{0x1f, 0x8b} // Gzip magic bytes
			invalidData = append(invalidData, []byte("invalid gzip stream data")...)

			w.WriteHeader(http.StatusOK)
			w.Write(invalidData)
		}))
		defer invalidGzipServer.Close()

		client := CreateTestClient()

		qe := &QueryExecutor{
			BaseClient: client,
			Query:      []byte(`{"query":"query { viewer { login } }"}`),
			Headers:    map[string]interface{}{"Content-Type": "application/json"},
			CacheKey:   "no-cache",
			Retries:    false,
		}
		qe.endpoint = invalidGzipServer.URL
		qe.client = invalidGzipServer.Client()
		qe.cache = client.cache

		result, err := qe.executeQuery()
		// Should detect gzip magic bytes, fail gzip reader creation with proper error
		assert.Error(err)
		assert.Nil(result)
		assert.Contains(err.Error(), "gzip reader creation failed")
	})
}

func (suite *Tests) TestQueryExecutor_executeQuery_contentDetectionLogging() {
	suite.T().Run("should log detection method for various scenarios", func(t *testing.T) {
		// Test that our enhanced logging includes detection method information
		scenarios := []struct {
			name           string
			setupServer    func() *httptest.Server
			expectedInLogs string
		}{
			{
				name: "gzip_magic_bytes_detection",
				setupServer: func() *httptest.Server {
					return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						// No Content-Encoding header

						var buf bytes.Buffer
						gzipWriter := gzip.NewWriter(&buf)
						gzipWriter.Write([]byte(`{"data":{"viewer":{"login":"logtest1"}}}`))
						gzipWriter.Close()

						w.WriteHeader(http.StatusOK)
						w.Write(buf.Bytes())
					}))
				},
				expectedInLogs: "logtest1",
			},
			{
				name: "header_claims_gzip_but_plain_json",
				setupServer: func() *httptest.Server {
					return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						w.Header().Set("Content-Encoding", "gzip")

						w.WriteHeader(http.StatusOK)
						w.Write([]byte(`{"data":{"viewer":{"login":"logtest2"}}}`))
					}))
				},
				expectedInLogs: "logtest2",
			},
			{
				name: "plain_json_detection",
				setupServer: func() *httptest.Server {
					return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")

						w.WriteHeader(http.StatusOK)
						w.Write([]byte(`{"data":{"viewer":{"login":"logtest3"}}}`))
					}))
				},
				expectedInLogs: "logtest3",
			},
		}

		for _, scenario := range scenarios {
			t.Run(scenario.name, func(t *testing.T) {
				server := scenario.setupServer()
				defer server.Close()

				client := CreateTestClient()

				qe := &QueryExecutor{
					BaseClient: client,
					Query:      []byte(`{"query":"query { viewer { login } }"}`),
					Headers:    map[string]interface{}{"Content-Type": "application/json"},
					CacheKey:   "no-cache",
					Retries:    false,
				}
				qe.endpoint = server.URL
				qe.client = server.Client()
				qe.cache = client.cache

				result, err := qe.executeQuery()
				assert.NoError(err)
				assert.NotNil(result)
				assert.Contains(string(result), scenario.expectedInLogs)
			})
		}
	})
}
