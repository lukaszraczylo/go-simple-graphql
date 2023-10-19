package gql

import (
	"crypto/tls"
	"fmt"
	"time"

	libpack_cache "github.com/lukaszraczylo/graphql-monitoring-proxy/cache"
	libpack_logging "github.com/lukaszraczylo/graphql-monitoring-proxy/logging"

	"golang.org/x/net/http2"

	"net/http"
	"strings"

	"github.com/gookit/goutil/envutil"
)

func NewConnection() *BaseClient {

	b := &BaseClient{
		client:        &http.Client{},
		MaxGoRoutines: -1,
	}

	b.endpoint = envutil.Getenv("GRAPHQL_ENDPOINT", "https://api.github.com/graphql")
	b.responseType = envutil.Getenv("GRAPHQL_OUTPUT", "string")
	b.validate = envutil.GetBool("GRAPHQL_VALIDATE", false)
	b.cache.enabled = envutil.GetBool("GRAPHQL_CACHE_ENABLED", true)
	b.cache.ttl = envutil.GetInt("GRAPHQL_CACHE_TTL", 60)

	b.retries.enabled = envutil.GetBool("GRAPHQL_RETRIES_ENABLED", true)
	b.retries.max = envutil.GetInt("GRAPHQL_RETRIES_MAX", 1)
	b.retries.delay = envutil.GetInt("GRAPHQL_RETRIES_DELAY", 300)

	b.enableCache()

	b_tmp_log_level := envutil.Getenv("LOG_LEVEL", "info")

	b.Logger = libpack_logging.NewLogger()

	var httpClient *http.Client

	httpTransport := &http.Transport{
		MaxIdleConns:          10,
		IdleConnTimeout:       15 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		DisableKeepAlives:     false,
	}

	if strings.HasPrefix(b.endpoint, "http://") {
		httpClient = &http.Client{
			Transport: httpTransport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	} else {
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
		httpClient = &http.Client{
			Transport: http2Transport,
		}
	}
	b.client = httpClient

	b.Logger.Debug("GraphQL client initialized", map[string]interface{}{"endpoint": b.endpoint, "responseType": b.responseType, "validate": b.validate, "cache": b.cache.enabled, "cacheTTL": b.cache.ttl, "maxGoRoutines": b.MaxGoRoutines, "loggingLevel": b_tmp_log_level})
	b.Logger.Info("GraphQL client initialized")
	return b
}

func (b *BaseClient) SetEndpoint(endpoint string) {
	b.endpoint = endpoint
}

func (b *BaseClient) SetOutput(output string) {
	b.responseType = output
}

func (b *BaseClient) enableCache() {
	var err error
	b.cache.client = libpack_cache.New(time.Duration(b.cache.ttl) * time.Second * 2)
	if err != nil {
		fmt.Println(">> Error while creating cache client;", "error", err.Error())
		panic(err)
	}
}

func (b *BaseClient) disableCache() {
	b.Logger.Debug("Disabling cache")
	b.cache.enabled = false
}
