package gql

import (
	"crypto/tls"
	"net/http"
	"strings"
	"time"

	libpack_logger "github.com/lukaszraczylo/go-simple-graphql/logging"
	"golang.org/x/net/http2"
)

func (b *BaseClient) createHttpClient() (http_client *http.Client) {
	// Create TLS config for HTTPS endpoints
	var tlsClientConfig *tls.Config
	if strings.HasPrefix(b.endpoint, "https://") {
		tlsClientConfig = &tls.Config{
			InsecureSkipVerify: true, // TODO: Make this configurable via environment variable
		}
	}

	// Use HTTP/2 transport for both http:// (h2c) and https:// endpoints
	// AllowHTTP=true enables HTTP/2 Cleartext (h2c) for unencrypted connections
	http2Transport := &http2.Transport{
		AllowHTTP:          true,             // Enable h2c for http:// endpoints
		TLSClientConfig:    tlsClientConfig,  // nil for http://, configured for https://
		ReadIdleTimeout:    30 * time.Second, // Increased for better connection reuse
		PingTimeout:        10 * time.Second, // Reduced for faster dead connection detection
		WriteByteTimeout:   10 * time.Second, // Write timeout for better error handling
		DisableCompression: true,             // Disable automatic compression to prevent trailing garbage errors
	}

	http_client = &http.Client{
		Timeout:   30 * time.Second, // Overall request timeout
		Transport: http2Transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects automatically
		},
	}

	// Log which protocol is being used
	if strings.HasPrefix(b.endpoint, "http://") {
		b.Logger.Debug(&libpack_logger.LogMessage{
			Message: "Using HTTP/2 Cleartext (h2c) over http",
			Pairs:   nil,
		})
	} else if strings.HasPrefix(b.endpoint, "https://") {
		b.Logger.Debug(&libpack_logger.LogMessage{
			Message: "Using HTTP/2 over TLS (https)",
			Pairs:   nil,
		})
	} else {
		b.Logger.Critical(&libpack_logger.LogMessage{
			Message: "Invalid endpoint - must start with http:// or https://",
		})
		return nil
	}

	return http_client
}
