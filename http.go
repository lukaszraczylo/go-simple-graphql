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
	// Optimized connection pool settings for better performance
	httpTransport := &http.Transport{
		MaxIdleConns:          100,              // Reduced from 512
		MaxConnsPerHost:       50,               // Reduced from 512
		MaxIdleConnsPerHost:   10,               // Reduced from 512
		IdleConnTimeout:       30 * time.Second, // Increased for better reuse
		ResponseHeaderTimeout: 10 * time.Second, // Reduced for faster timeouts
		DisableKeepAlives:     false,
		DisableCompression:    false,
		WriteBufferSize:       4096, // Optimize buffer size
		ReadBufferSize:        4096, // Optimize buffer size
	}

	if strings.HasPrefix(b.endpoint, "http://") {
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
	} else if strings.HasPrefix(b.endpoint, "https://") {
		tlsClientConfig := &tls.Config{}
		if strings.HasPrefix(b.endpoint, "https://") {
			tlsClientConfig.InsecureSkipVerify = true
		}
		http2Transport := &http2.Transport{
			AllowHTTP:        true,
			TLSClientConfig:  tlsClientConfig,
			ReadIdleTimeout:  30 * time.Second, // Increased for better reuse
			PingTimeout:      10 * time.Second, // Reduced for faster detection
			WriteByteTimeout: 10 * time.Second, // Add write timeout
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
	}
	return http_client
}
