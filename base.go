package gql

import (
	"crypto/tls"
	"fmt"
	"time"

	"github.com/lukaszraczylo/go-simple-graphql/utils/logger"
	libpack_cache "github.com/lukaszraczylo/graphql-monitoring-proxy/cache"

	"golang.org/x/net/http2"

	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gookit/goutil/envutil"
)

func NewConnection() *BaseClient {

	b := &BaseClient{
		client:         &http.Client{},
		MaxGoRoutines:  -1,
		LoggingLevel:   logger.Warn,
		LoggerColorful: true,
	}

	if b.LoggerWriter == nil {
		b.LoggerWriter = log.New(os.Stdout, "", log.LstdFlags)
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

	switch b_tmp_log_level {
	case "silent":
		b.LoggingLevel = logger.Silent
	case "error":
		b.LoggingLevel = logger.Error
	case "warn":
		b.LoggingLevel = logger.Warn
	case "info":
		b.LoggingLevel = logger.Info
	case "debug":
		b.LoggingLevel = logger.Debug
	}

	b.Logger = NewLogger(b.LoggerWriter, logger.Config{
		Colorful: b.LoggerColorful,
		LogLevel: b.LoggingLevel,
	})

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

	b.Logger.Debug(b, "graphQL client initialized;", "endpoint", b.endpoint, "responseType", b.responseType, "validate", b.validate, "cache", b.cache.enabled, "cacheTTL", b.cache.ttl, "maxGoRoutines", b.MaxGoRoutines, "loggingLevel", b_tmp_log_level, "loggerColorful", b.LoggerColorful)
	b.Logger.Info(b, "graphQL client initialized;")
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
	b.Logger.Debug(b, "Disabling cache")
	b.cache.enabled = false
}
