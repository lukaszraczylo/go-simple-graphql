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
		DisableCompression:  true, // Disable automatic compression to handle gzip manually
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

	// Set Accept-Encoding header to request gzip compression
	if httpRequest.Header.Get("Accept-Encoding") == "" {
		httpRequest.Header.Set("Accept-Encoding", "gzip")
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

			// Log Content-Encoding header for debugging
			encoding := httpResponse.Header.Get("Content-Encoding")
			qe.Logger.Debug(&libpack_logger.LogMessage{
				Message: "Processing response",
				Pairs: map[string]interface{}{
					"content_encoding": encoding,
					"content_type":     httpResponse.Header.Get("Content-Type"),
				},
			})

			// First, read the entire response body into a buffer
			rawBuf := bufferPool.Get().(*bytes.Buffer)
			rawBuf.Reset()
			defer bufferPool.Put(rawBuf)

			_, err = io.Copy(rawBuf, httpResponse.Body)
			if err != nil {
				return fmt.Errorf("error reading HTTP response body: %w", err)
			}

			rawData := rawBuf.Bytes()
			qe.Logger.Debug(&libpack_logger.LogMessage{
				Message: "Read raw response data",
				Pairs: map[string]interface{}{
					"response_size": len(rawData),
				},
			})

			// Use buffer pool for final data processing
			respBuf := bufferPool.Get().(*bytes.Buffer)
			respBuf.Reset()
			defer bufferPool.Put(respBuf)

			var finalData []byte

			// Handle gzip decompression with fallback
			if encoding == "gzip" {
				// Try to create gzip reader from raw data
				gzipReader, err := gzip.NewReader(bytes.NewReader(rawData))
				if err != nil {
					qe.Logger.Warning(&libpack_logger.LogMessage{
						Message: "Failed to create gzip reader, falling back to uncompressed parsing",
						Pairs:   map[string]interface{}{"gzip_error": err.Error()},
					})
					// Fallback: treat raw data as uncompressed
					finalData = rawData
				} else {
					// Successfully created gzip reader, now try to decompress
					defer gzipReader.Close()

					_, copyErr := io.Copy(respBuf, gzipReader)
					if copyErr != nil {
						// Gzip decompression failed, try fallback
						qe.Logger.Warning(&libpack_logger.LogMessage{
							Message: "Gzip decompression failed, falling back to uncompressed parsing",
							Pairs: map[string]interface{}{
								"copy_error": copyErr,
								"raw_size":   len(rawData),
							},
						})
						// Fallback: treat raw data as uncompressed
						finalData = rawData
					} else {
						// Successful decompression
						finalData = respBuf.Bytes()
						qe.Logger.Debug(&libpack_logger.LogMessage{
							Message: "Successfully decompressed gzip response",
							Pairs: map[string]interface{}{
								"compressed_size":   len(rawData),
								"decompressed_size": len(finalData),
							},
						})
					}
				}
			} else {
				// No compression, use raw data directly
				finalData = rawData
			}

			// Validate that we have data before attempting JSON unmarshaling
			if len(finalData) == 0 {
				return fmt.Errorf("empty response data after processing")
			}

			// Log the first few bytes for debugging (truncated for safety)
			debugData := finalData
			if len(debugData) > 100 {
				debugData = debugData[:100]
			}
			qe.Logger.Debug(&libpack_logger.LogMessage{
				Message: "Attempting JSON unmarshaling",
				Pairs: map[string]interface{}{
					"data_size":   len(finalData),
					"data_sample": string(debugData),
				},
			})

			// Unmarshal the final processed data
			err = json.Unmarshal(finalData, &queryResult)
			if err != nil {
				qe.Logger.Error(&libpack_logger.LogMessage{
					Message: "JSON unmarshaling failed",
					Pairs: map[string]interface{}{
						"error":       err.Error(),
						"data_size":   len(finalData),
						"data_sample": string(debugData),
						"encoding":    encoding,
					},
				})
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
