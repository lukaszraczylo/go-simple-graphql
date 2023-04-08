package gql

import (
	"crypto/tls"
	"time"

	"github.com/lukaszraczylo/go-simple-graphql/utils/concurrency"
	"github.com/lukaszraczylo/go-simple-graphql/utils/logger"

	"golang.org/x/net/http2"

	"log"
	"net/http"
	"os"
	"strings"

	"github.com/allegro/bigcache"
	"github.com/gookit/goutil/envutil"
)

func NewConnection() *BaseClient {

	b := &BaseClient{
		client:             &http.Client{},
		concurrencyManager: concurrency.NewPool(-1),
		MaxGoRoutines:      -1,
		LoggingLevel:       logger.Warn,
		LoggerColorful:     true,
	}

	if b.LoggerWriter == nil {
		b.LoggerWriter = log.New(os.Stdout, "", log.LstdFlags)
	}

	b.endpoint = envutil.Getenv("GRAPHQL_ENDPOINT", "https://api.github.com/graphql")
	b.responseType = envutil.Getenv("GRAPHQL_OUTPUT", "mapstring")
	b.validate = envutil.GetBool("GRAPHQL_VALIDATE", false)
	b.cache.enabled = envutil.GetBool("GRAPHQL_CACHE_ENABLED", true)
	b.cache.ttl = envutil.GetInt("GRAPHQL_CACHE_TTL", 60)

	b.retries.enabled = envutil.GetBool("GRAPHQL_RETRIES_ENABLED", true)
	b.retries.max = envutil.GetInt("GRAPHQL_RETRIES_MAX", 1)
	b.retries.delay = envutil.GetInt("GRAPHQL_RETRIES_DELAY", 300)

	if b.cache.enabled {
		var err error
		b.cache.client, err = bigcache.NewBigCache(bigcache.DefaultConfig(time.Duration(b.cache.ttl) * time.Second))
		if err != nil {
			b.Logger.Error(b, "Error while creating cache client;", "error", err.Error())
		}
	}

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
	if strings.HasPrefix(b.endpoint, "http://") {
		httpClient = &http.Client{
			Transport: http.DefaultTransport,
		}
	} else {
		tlsClientConfig := &tls.Config{}
		if strings.HasPrefix(b.endpoint, "https://") {
			tlsClientConfig.InsecureSkipVerify = true
		}
		http2Transport := &http2.Transport{
			AllowHTTP:       true,
			TLSClientConfig: tlsClientConfig,
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
