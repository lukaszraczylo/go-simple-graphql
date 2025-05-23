package gql

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/goccy/go-json"
	libpack_logger "github.com/lukaszraczylo/go-simple-graphql/logging"
)

var (
	// Shared HTTP transport with optimized settings
	defaultTransport = &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false, // Enable compression for responses
		ForceAttemptHTTP2:   true,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true}, // Skip TLS verification for test environments
	}

	// Shared HTTP client with timeouts
	defaultClient = &http.Client{
		Transport: defaultTransport,
		Timeout:   30 * time.Second,
	}
)

func (qe *QueryExecutor) executeQuery() ([]byte, error) {
	// Reuse buffer from pool to avoid allocations
	buf := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buf)
	buf.Reset()

	buf.Write(qe.Query)

	httpRequest, err := http.NewRequest(http.MethodPost, qe.endpoint, nil)
	if err != nil {
		qe.Logger.Error(&libpack_logger.LogMessage{
			Message: "Can't create HTTP request",
			Pairs:   map[string]interface{}{"error": err.Error()},
		})
		return nil, fmt.Errorf("can't create HTTP request: %w", err)
	}

	for key, value := range qe.Headers {
		httpRequest.Header.Set(key, fmt.Sprint(value))
	}
	// Set Content-Type header if not already set
	if httpRequest.Header.Get("Content-Type") == "" {
		httpRequest.Header.Set("Content-Type", "application/json")
	}

	retriesMax := 1
	if qe.Retries {
		retriesMax = qe.retries_number
	}

	var queryResult queryResults
	err = retry.Do(
		func() error {
			// Reset buffer before each retry
			buf.Reset()
			buf.Write(qe.Query)

			// Set the body of the request
			httpRequest.Body = io.NopCloser(bytes.NewReader(buf.Bytes()))
			httpRequest.ContentLength = int64(buf.Len())

			// Use default client if custom client is not set
			client := qe.client
			if client == nil {
				client = defaultClient
			}
			httpResponse, err := client.Do(httpRequest)
			if err != nil {
				return err
			}
			defer func() {
				io.Copy(io.Discard, httpResponse.Body)
				httpResponse.Body.Close()
			}()

			if httpResponse.StatusCode < http.StatusOK || httpResponse.StatusCode >= http.StatusMultipleChoices {
				return fmt.Errorf("HTTP error - status code: %s for %s", httpResponse.Status, httpRequest.URL)
			}

			var reader io.Reader
			encoding := httpResponse.Header.Get("Content-Encoding")
			if encoding == "gzip" {
				gzipReader, err := gzip.NewReader(httpResponse.Body)
				if err != nil {
					return fmt.Errorf("error creating gzip reader: %w", err)
				}
				defer gzipReader.Close()
				reader = gzipReader
			} else {
				reader = httpResponse.Body
			}

			// Use buffer pool for reading response
			respBuf := bufferPool.Get().(*bytes.Buffer)
			respBuf.Reset()
			defer bufferPool.Put(respBuf)

			_, err = io.Copy(respBuf, reader)
			if err != nil {
				return fmt.Errorf("error reading HTTP response: %w", err)
			}

			// Unmarshal response directly from buffer
			err = json.Unmarshal(respBuf.Bytes(), &queryResult)
			if err != nil {
				return fmt.Errorf("error unmarshalling HTTP response: %w", err)
			}

			return nil
		},
		retry.OnRetry(func(n uint, err error) {
			qe.Logger.Warning(&libpack_logger.LogMessage{
				Message: "Retrying query",
				Pairs:   map[string]interface{}{"error": err.Error(), "attempt": n + 1},
			})
		}),
		retry.Attempts(uint(retriesMax)),
		retry.DelayType(retry.BackOffDelay),
		retry.Delay(time.Duration(qe.retries_delay)),
		retry.MaxDelay(10*time.Second),
		retry.LastErrorOnly(true),
	)
	if err != nil {
		qe.Logger.Debug(&libpack_logger.LogMessage{
			Message: "Error executing HTTP request",
			Pairs:   map[string]interface{}{"error": err.Error()},
		})
		return nil, err
	}

	if len(queryResult.Errors) > 0 {
		qe.Logger.Debug(&libpack_logger.LogMessage{
			Message: "Error executing query",
			Pairs:   map[string]interface{}{"error": queryResult.Errors},
		})
		return nil, fmt.Errorf("error executing query: %s", queryResult.Errors)
	}

	if queryResult.Data == nil {
		qe.Logger.Debug(&libpack_logger.LogMessage{
			Message: "Error executing query",
			Pairs:   map[string]interface{}{"error": "no data"},
		})
		return nil, errors.New("error executing query: no data")
	}

	// Marshal queryResult.Data
	jsonData, err := json.Marshal(queryResult.Data)
	if err != nil {
		qe.Logger.Debug(&libpack_logger.LogMessage{
			Message: "Error marshalling query result",
			Pairs:   map[string]interface{}{"error": err.Error(), "data": queryResult.Data},
		})
		return nil, fmt.Errorf("error marshalling query result: %w. Data: %s", err, queryResult.Data)
	}

	if qe.CacheKey != "no-cache" {
		qe.cache.Set(qe.CacheKey, jsonData, qe.CacheTTL)
	}

	return jsonData, nil
}
