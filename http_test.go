package gql

import (
	"net/http"
	"os"
	"testing"
	"time"

	"golang.org/x/net/http2"
)

func (suite *Tests) TestBaseClient_createHttpClient() {
	suite.T().Run("should create HTTP/1.1 client for http endpoints", func(t *testing.T) {
		client := NewConnection()
		client.SetEndpoint("http://example.com/graphql")

		httpClient, err := client.createHttpClient()
		assert.NoError(err)
		assert.NotNil(httpClient)
		assert.Equal(30*time.Second, httpClient.Timeout)
		assert.NotNil(httpClient.Transport)

		// Check that it's using regular HTTP transport
		transport, ok := httpClient.Transport.(*http.Transport)
		assert.True(ok)
		assert.Equal(100, transport.MaxIdleConns)
		assert.Equal(50, transport.MaxConnsPerHost)
		assert.Equal(10, transport.MaxIdleConnsPerHost)
		assert.Equal(30*time.Second, transport.IdleConnTimeout)
		assert.Equal(10*time.Second, transport.ResponseHeaderTimeout)
		assert.False(transport.DisableKeepAlives)
		assert.True(transport.DisableCompression) // Request compression disabled to prevent "trailing garbage" errors
		assert.Equal(4096, transport.WriteBufferSize)
		assert.Equal(4096, transport.ReadBufferSize)
	})

	suite.T().Run("should create HTTP/2 client for https endpoints", func(t *testing.T) {
		// Set environment variable for test to skip TLS verification
		oldValue := os.Getenv("GRAPHQL_INSECURE_SKIP_VERIFY")
		os.Setenv("GRAPHQL_INSECURE_SKIP_VERIFY", "true")
		t.Cleanup(func() {
			if oldValue != "" {
				os.Setenv("GRAPHQL_INSECURE_SKIP_VERIFY", oldValue)
			} else {
				os.Unsetenv("GRAPHQL_INSECURE_SKIP_VERIFY")
			}
		})

		client := NewConnection()
		client.SetEndpoint("https://example.com/graphql")

		httpClient, err := client.createHttpClient()
		assert.NoError(err)
		assert.NotNil(httpClient)
		assert.Equal(30*time.Second, httpClient.Timeout)
		assert.NotNil(httpClient.Transport)

		// Check that it's using HTTP/2 transport
		transport, ok := httpClient.Transport.(*http2.Transport)
		assert.True(ok)
		assert.True(transport.AllowHTTP)
		assert.NotNil(transport.TLSClientConfig)
		assert.True(transport.TLSClientConfig.InsecureSkipVerify)
		assert.Equal(30*time.Second, transport.ReadIdleTimeout)
		assert.Equal(10*time.Second, transport.PingTimeout)
		assert.Equal(10*time.Second, transport.WriteByteTimeout)
	})

	suite.T().Run("should handle invalid endpoints", func(t *testing.T) {
		client := NewConnection()
		client.SetEndpoint("ftp://example.com/graphql")

		httpClient, err := client.createHttpClient()
		assert.Error(err)
		assert.Contains(err.Error(), "invalid endpoint")
		assert.Nil(httpClient)
	})

	suite.T().Run("should set redirect policy", func(t *testing.T) {
		client := NewConnection()
		client.SetEndpoint("http://example.com/graphql")

		httpClient, err := client.createHttpClient()
		assert.NoError(err)
		assert.NotNil(httpClient)
		assert.NotNil(httpClient.CheckRedirect)

		// Test redirect policy
		req, _ := http.NewRequest("GET", "http://example.com", nil)
		err2 := httpClient.CheckRedirect(req, []*http.Request{})
		assert.Equal(http.ErrUseLastResponse, err2)
	})

	suite.T().Run("should handle empty endpoint", func(t *testing.T) {
		client := NewConnection()
		client.SetEndpoint("")

		httpClient, err := client.createHttpClient()
		assert.Error(err)
		assert.Contains(err.Error(), "invalid endpoint")
		assert.Nil(httpClient)
	})

	suite.T().Run("should handle endpoint without protocol", func(t *testing.T) {
		client := NewConnection()
		client.SetEndpoint("example.com/graphql")

		httpClient, err := client.createHttpClient()
		assert.Error(err)
		assert.Contains(err.Error(), "invalid endpoint")
		assert.Nil(httpClient)
	})

	suite.T().Run("should enable TLS verification by default for https endpoints", func(t *testing.T) {
		// Ensure the environment variable is not set
		oldValue := os.Getenv("GRAPHQL_INSECURE_SKIP_VERIFY")
		os.Unsetenv("GRAPHQL_INSECURE_SKIP_VERIFY")
		t.Cleanup(func() {
			if oldValue != "" {
				os.Setenv("GRAPHQL_INSECURE_SKIP_VERIFY", oldValue)
			}
		})

		client := NewConnection()
		client.SetEndpoint("https://example.com/graphql")

		httpClient, err := client.createHttpClient()
		assert.NoError(err)
		assert.NotNil(httpClient)

		// Check that TLS verification is enabled by default
		transport, ok := httpClient.Transport.(*http2.Transport)
		assert.True(ok)
		assert.NotNil(transport.TLSClientConfig)
		assert.False(transport.TLSClientConfig.InsecureSkipVerify) // Should be false by default
	})
}

func (suite *Tests) TestBaseClient_createHttpClient_transportSettings() {
	suite.T().Run("should configure HTTP transport correctly", func(t *testing.T) {
		client := NewConnection()
		client.SetEndpoint("http://example.com/graphql")

		httpClient, err := client.createHttpClient()
		assert.NoError(err)
		assert.NotNil(httpClient)
		transport := httpClient.Transport.(*http.Transport)

		// Verify all transport settings
		assert.Equal(100, transport.MaxIdleConns)
		assert.Equal(50, transport.MaxConnsPerHost)
		assert.Equal(10, transport.MaxIdleConnsPerHost)
		assert.Equal(30*time.Second, transport.IdleConnTimeout)
		assert.Equal(10*time.Second, transport.ResponseHeaderTimeout)
		assert.False(transport.DisableKeepAlives)
		assert.True(transport.DisableCompression) // Request compression disabled to prevent "trailing garbage" errors
		assert.Equal(4096, transport.WriteBufferSize)
		assert.Equal(4096, transport.ReadBufferSize)
	})

	suite.T().Run("should configure HTTP/2 transport correctly", func(t *testing.T) {
		// Set environment variable for test to skip TLS verification
		oldValue := os.Getenv("GRAPHQL_INSECURE_SKIP_VERIFY")
		os.Setenv("GRAPHQL_INSECURE_SKIP_VERIFY", "true")
		t.Cleanup(func() {
			if oldValue != "" {
				os.Setenv("GRAPHQL_INSECURE_SKIP_VERIFY", oldValue)
			} else {
				os.Unsetenv("GRAPHQL_INSECURE_SKIP_VERIFY")
			}
		})

		client := NewConnection()
		client.SetEndpoint("https://example.com/graphql")

		httpClient, err := client.createHttpClient()
		assert.NoError(err)
		assert.NotNil(httpClient)
		transport := httpClient.Transport.(*http2.Transport)

		// Verify all HTTP/2 transport settings
		assert.True(transport.AllowHTTP)
		assert.NotNil(transport.TLSClientConfig)
		assert.True(transport.TLSClientConfig.InsecureSkipVerify)
		assert.Equal(30*time.Second, transport.ReadIdleTimeout)
		assert.Equal(10*time.Second, transport.PingTimeout)
		assert.Equal(10*time.Second, transport.WriteByteTimeout)
	})
}

func (suite *Tests) TestBaseClient_createHttpClient_logging() {
	suite.T().Run("should log HTTP/1.1 usage", func(t *testing.T) {
		client := NewConnection()
		client.SetEndpoint("http://example.com/graphql")

		// This test verifies that the function runs without error
		// The actual logging is tested indirectly through the logger tests
		httpClient, err := client.createHttpClient()
		assert.NoError(err)
		assert.NotNil(httpClient)
	})

	suite.T().Run("should log HTTP/2 usage", func(t *testing.T) {
		// Set environment variable for test to skip TLS verification
		oldValue := os.Getenv("GRAPHQL_INSECURE_SKIP_VERIFY")
		os.Setenv("GRAPHQL_INSECURE_SKIP_VERIFY", "true")
		t.Cleanup(func() {
			if oldValue != "" {
				os.Setenv("GRAPHQL_INSECURE_SKIP_VERIFY", oldValue)
			} else {
				os.Unsetenv("GRAPHQL_INSECURE_SKIP_VERIFY")
			}
		})

		client := NewConnection()
		client.SetEndpoint("https://example.com/graphql")

		// This test verifies that the function runs without error
		// The actual logging is tested indirectly through the logger tests
		httpClient, err := client.createHttpClient()
		assert.NoError(err)
		assert.NotNil(httpClient)
	})

	suite.T().Run("should log critical error for invalid endpoint", func(t *testing.T) {
		client := NewConnection()
		client.SetEndpoint("invalid://example.com/graphql")

		// This test verifies that the function runs without error
		// and returns nil for invalid endpoints (Critical() won't exit in tests)
		httpClient, err := client.createHttpClient()
		assert.Error(err)
		assert.Contains(err.Error(), "invalid endpoint")
		assert.Nil(httpClient)
	})
}
