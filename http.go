package gql

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	libpack_logger "github.com/lukaszraczylo/go-simple-graphql/logging"
	"golang.org/x/net/http2"
)

func (b *BaseClient) createHttpClient() (*http.Client, error) {
	// Optimized connection pool settings for better performance
	httpTransport := &http.Transport{
		MaxIdleConns:          100,              // Reduced from 512
		MaxConnsPerHost:       50,               // Reduced from 512
		MaxIdleConnsPerHost:   10,               // Reduced from 512
		IdleConnTimeout:       30 * time.Second, // Increased for better reuse
		ResponseHeaderTimeout: 10 * time.Second, // Reduced for faster timeouts
		DisableKeepAlives:     false,
		DisableCompression:    true, // Disable automatic request compression to prevent "trailing garbage" errors
		WriteBufferSize:       4096, // Optimize buffer size
		ReadBufferSize:        4096, // Optimize buffer size
	}

	var http_client *http.Client

	// Protect read of endpoint field
	b.mu.RLock()
	endpoint := b.endpoint
	b.mu.RUnlock()

	if strings.HasPrefix(endpoint, "http://") {
		http_client = &http.Client{
			Timeout:   30 * time.Second, // Increased for better reliability
			Transport: httpTransport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		b.Logger.Debug(&libpack_logger.LogMessage{
			Message: "Using HTTP/1.1 over http",
			Pairs:   nil,
		})
	} else if strings.HasPrefix(endpoint, "https://") {
		tlsClientConfig := getHTTPClientTLSConfig()
		if strings.HasPrefix(endpoint, "https://") && tlsClientConfig.InsecureSkipVerify {
			b.Logger.Warning(&libpack_logger.LogMessage{
				Message: "TLS certificate verification is disabled. This is insecure and should not be used in production.",
			})
		}
		http2Transport := &http2.Transport{
			AllowHTTP:          true,
			TLSClientConfig:    tlsClientConfig,
			ReadIdleTimeout:    30 * time.Second, // Increased for better reuse
			PingTimeout:        10 * time.Second, // Reduced for faster detection
			WriteByteTimeout:   10 * time.Second, // Add write timeout
			DisableCompression: true,             // Disable automatic request compression to prevent "trailing garbage" errors
		}
		http_client = &http.Client{
			Timeout:   30 * time.Second, // Increased for better reliability
			Transport: http2Transport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		b.Logger.Debug(&libpack_logger.LogMessage{
			Message: "Using HTTP/2 over https",
			Pairs:   nil,
		})
	} else {
		b.Logger.Critical(&libpack_logger.LogMessage{
			Message: "Invalid endpoint - neither http or https",
		})
		// Return error instead of nil to prevent nil pointer dereference
		return nil, fmt.Errorf("invalid endpoint: %s - must start with http:// or https://", endpoint)
	}
	return http_client, nil
}

// getHTTPClientTLSConfig returns TLS configuration for HTTP client
// By default, TLS verification is enabled for security.
// Set GRAPHQL_INSECURE_SKIP_VERIFY=true to disable verification (NOT RECOMMENDED for production)
func getHTTPClientTLSConfig() *tls.Config {
	skipVerify := os.Getenv("GRAPHQL_INSECURE_SKIP_VERIFY") == "true"

	if skipVerify {
		// Log warning to stderr when running in insecure mode
		fmt.Fprintf(os.Stderr, "WARNING: TLS certificate verification is disabled. This is insecure and should not be used in production.\n")
	}

	return &tls.Config{
		InsecureSkipVerify: skipVerify,
		MinVersion:         tls.VersionTLS12, // Enforce minimum TLS 1.2 for security
	}
}
