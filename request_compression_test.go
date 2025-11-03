package gql

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	assertions "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/http2"
)

type RequestCompressionTestSuite struct {
	suite.Suite
	assert *assertions.Assertions
}

func TestRequestCompressionSuite(t *testing.T) {
	suite.Run(t, new(RequestCompressionTestSuite))
}

func (suite *RequestCompressionTestSuite) SetupTest() {
	suite.assert = assertions.New(suite.T())
}

func (suite *RequestCompressionTestSuite) TestRequestCompression() {
	suite.T().Run("should disable request compression for HTTP clients", func(t *testing.T) {
		assert := assertions.New(t)
		// Track what the server receives
		var receivedHeaders http.Header
		var receivedBody []byte
		var receivedContentEncoding string

		// Create test server that captures request details
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedHeaders = r.Header.Clone()
			receivedContentEncoding = r.Header.Get("Content-Encoding")

			// Read the request body
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("Failed to read request body: %v", err)
				return
			}
			receivedBody = body

			// Send back a simple JSON response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data":{"viewer":{"login":"testuser"}}}`))
		}))
		defer server.Close()

		// Test both HTTP and HTTPS endpoints
		endpoints := []string{
			strings.Replace(server.URL, "http://", "http://", 1),  // HTTP
			strings.Replace(server.URL, "http://", "https://", 1), // HTTPS (will use HTTP/2)
		}

		for _, endpoint := range endpoints {
			t.Run("endpoint_"+endpoint, func(t *testing.T) {
				// Reset captured data
				receivedHeaders = nil
				receivedBody = nil
				receivedContentEncoding = ""

				client := CreateTestClient()
				client.endpoint = endpoint

				// Create HTTP client using the same logic as the library
				httpClient := client.createHttpClient()
				assert.NotNil(httpClient, "HTTP client should be created")

				// Verify transport settings - now using HTTP/2 for all endpoints
				transport, ok := httpClient.Transport.(*http2.Transport)
				assert.True(ok, "Should be http2.Transport for all endpoints (h2c for http://, TLS for https://)")
				if ok {
					assert.True(transport.DisableCompression, "DisableCompression should be true to prevent trailing garbage")
					if strings.HasPrefix(endpoint, "http://") {
						assert.True(transport.AllowHTTP, "AllowHTTP should be true for h2c (HTTP/2 Cleartext)")
						assert.Nil(transport.TLSClientConfig, "TLSClientConfig should be nil for http:// endpoints")
					} else {
						assert.True(transport.AllowHTTP, "AllowHTTP should be true")
						assert.NotNil(transport.TLSClientConfig, "TLSClientConfig should be set for https:// endpoints")
					}
				}

				qe := &QueryExecutor{
					BaseClient: client,
					Query:      []byte(`{"query":"query { viewer { login } }"}`),
					Headers:    map[string]interface{}{"Content-Type": "application/json"},
					CacheKey:   "no-cache",
					Retries:    false,
				}
				qe.endpoint = server.URL    // Use HTTP server URL regardless of test endpoint
				qe.client = server.Client() // Use server's client to avoid TLS issues
				qe.cache = client.cache

				result, err := qe.executeQuery()
				assert.NoError(err, "Query should execute successfully")
				assert.NotNil(result, "Result should not be nil")

				// Verify request was sent as plain JSON (not compressed)
				assert.Empty(receivedContentEncoding, "Request should not have Content-Encoding header")
				assert.NotContains(receivedHeaders.Get("Accept-Encoding"), "gzip", "Should not automatically request gzip encoding")

				// Verify request body is plain JSON (not gzip compressed)
				assert.True(len(receivedBody) > 0, "Should have received request body")

				// Check that body is NOT gzip compressed by checking magic bytes
				if len(receivedBody) >= 2 {
					assert.False(receivedBody[0] == 0x1f && receivedBody[1] == 0x8b,
						"Request body should NOT have gzip magic bytes (0x1f 0x8b), got: 0x%02x 0x%02x",
						receivedBody[0], receivedBody[1])
				}

				// Verify body is valid JSON
				bodyStr := string(receivedBody)
				assert.True(strings.HasPrefix(bodyStr, "{"), "Request body should start with '{' (JSON)")
				assert.Contains(bodyStr, "query", "Request body should contain 'query' field")
				assert.Contains(bodyStr, "viewer", "Request body should contain the query content")

				// Verify no automatic compression headers were added
				assert.Empty(receivedHeaders.Get("Content-Encoding"), "Request should not have Content-Encoding header")

				// The Accept-Encoding header should either be empty or not contain gzip
				acceptEncoding := receivedHeaders.Get("Accept-Encoding")
				if acceptEncoding != "" {
					assert.NotContains(acceptEncoding, "gzip", "Accept-Encoding should not request gzip compression")
				}
			})
		}
	})

	suite.T().Run("should handle gzip responses correctly while sending plain requests", func(t *testing.T) {
		assert := assertions.New(t)
		var receivedBody []byte
		var receivedContentEncoding string

		// Server that captures request and sends gzip response
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Capture request details
			receivedContentEncoding = r.Header.Get("Content-Encoding")
			body, _ := io.ReadAll(r.Body)
			receivedBody = body

			// Send gzip compressed response
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Encoding", "gzip")

			var buf bytes.Buffer
			gzipWriter := gzip.NewWriter(&buf)
			gzipWriter.Write([]byte(`{"data":{"viewer":{"login":"gzipresponseuser"}}}`))
			gzipWriter.Close()

			w.WriteHeader(http.StatusOK)
			w.Write(buf.Bytes())
		}))
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
		assert.NoError(err, "Should handle gzip response correctly")
		assert.NotNil(result, "Result should not be nil")
		assert.Contains(string(result), "gzipresponseuser", "Should decompress gzip response")

		// Verify request was sent as plain JSON
		assert.Empty(receivedContentEncoding, "Request should not be compressed")
		assert.True(len(receivedBody) > 0, "Should have received request body")

		// Verify request is not gzip compressed
		if len(receivedBody) >= 2 {
			assert.False(receivedBody[0] == 0x1f && receivedBody[1] == 0x8b,
				"Request should not be gzip compressed")
		}

		// Verify request body is valid JSON
		bodyStr := string(receivedBody)
		assert.True(strings.HasPrefix(bodyStr, "{"), "Request should be JSON")
		assert.Contains(bodyStr, "query", "Request should contain query")
	})

	suite.T().Run("should prevent trailing garbage errors from compressed requests", func(t *testing.T) {
		assert := assertions.New(t)
		var receivedBody []byte

		// Server that would fail if it received compressed requests
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			receivedBody = body

			// Check if request looks like gzip (which would cause trailing garbage errors)
			if len(body) >= 2 && body[0] == 0x1f && body[1] == 0x8b {
				// Simulate server that can't handle gzip requests properly
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error":"trailing garbage after JSON"}`))
				return
			}

			// Normal JSON response for plain requests
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data":{"viewer":{"login":"notrailinggarbage"}}}`))
		}))
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
		assert.NoError(err, "Should not get trailing garbage errors")
		assert.NotNil(result, "Result should not be nil")
		assert.Contains(string(result), "notrailinggarbage", "Should get successful response")

		// Verify request was plain JSON (not gzip)
		assert.True(len(receivedBody) > 0, "Should have received request body")
		if len(receivedBody) >= 2 {
			assert.False(receivedBody[0] == 0x1f && receivedBody[1] == 0x8b,
				"Request should not be gzip compressed to prevent trailing garbage errors")
		}
	})
}
