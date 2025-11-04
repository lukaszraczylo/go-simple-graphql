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
	if strings.HasPrefix(b.endpoint, "http://") {
		// Use HTTP/1.1 for http:// endpoints
		// Note: h2c (HTTP/2 Cleartext) is rarely supported by servers
		httpTransport := &http.Transport{
			MaxIdleConns:          100,
			MaxConnsPerHost:       50,
			MaxIdleConnsPerHost:   10,
			IdleConnTimeout:       30 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			DisableKeepAlives:     false,
			DisableCompression:    true, // Disable automatic compression to prevent trailing garbage errors
			WriteBufferSize:       4096,
			ReadBufferSize:        4096,
			// Additional settings for connection health
			ExpectContinueTimeout: 1 * time.Second,  // Timeout for 100-Continue
			TLSHandshakeTimeout:   10 * time.Second, // Timeout for TLS handshake
		}

		http_client = &http.Client{
			Timeout:   30 * time.Second,
			Transport: httpTransport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}

		b.Logger.Debug(&libpack_logger.LogMessage{
			Message: "Using HTTP/1.1 for http:// endpoint",
			Pairs:   nil,
		})
	} else if strings.HasPrefix(b.endpoint, "https://") {
		// Use HTTP/2 for https:// endpoints
		tlsClientConfig := &tls.Config{
			InsecureSkipVerify: true, // TODO: Make this configurable via environment variable
		}

		http2Transport := &http2.Transport{
			AllowHTTP:          true,
			TLSClientConfig:    tlsClientConfig,
			ReadIdleTimeout:    30 * time.Second, // Close idle connections after 30s
			PingTimeout:        10 * time.Second, // Detect dead connections with PING
			WriteByteTimeout:   10 * time.Second, // Timeout for write operations
			DisableCompression: true,             // Disable automatic compression to prevent trailing garbage errors
			// StrictMaxConcurrentStreams forces the HTTP/2 connection to wait for SETTINGS frame
			// This helps with connection health detection
			StrictMaxConcurrentStreams: true,
		}

		http_client = &http.Client{
			Timeout:   30 * time.Second,
			Transport: http2Transport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}

		b.Logger.Debug(&libpack_logger.LogMessage{
			Message: "Using HTTP/2 for https:// endpoint",
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
