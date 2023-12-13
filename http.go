package gql

import (
	"crypto/tls"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/http2"
)

func (b *BaseClient) createHttpClient() (http_client *http.Client) {
	httpTransport := &http.Transport{
		MaxIdleConns:          512,
		MaxConnsPerHost:       512,
		MaxIdleConnsPerHost:   512,
		IdleConnTimeout:       15 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		DisableKeepAlives:     false,
		DisableCompression:    false,
	}

	if strings.HasPrefix(b.endpoint, "http://") {
		http_client = &http.Client{
			Timeout:   15 * time.Second,
			Transport: httpTransport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		b.Logger.Debug("Using HTTP/1.1 over http")
	} else if strings.HasPrefix(b.endpoint, "https://") {
		tlsClientConfig := &tls.Config{}
		if strings.HasPrefix(b.endpoint, "https://") {
			tlsClientConfig.InsecureSkipVerify = true
		}
		http2Transport := &http2.Transport{
			AllowHTTP:       true,
			TLSClientConfig: tlsClientConfig,
			ReadIdleTimeout: 15 * time.Second,
			PingTimeout:     15 * time.Second,
		}
		http_client = &http.Client{
			Transport: http2Transport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		b.Logger.Debug("Using HTTP/2 over https")
	} else {
		b.Logger.Critical("Invalid endpoint - neither http or https")
	}
	return http_client
}
