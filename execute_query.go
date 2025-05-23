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
	"unicode/utf8"

	"github.com/avast/retry-go/v4"
	"github.com/goccy/go-json"
	libpack_logger "github.com/lukaszraczylo/go-simple-graphql/logging"
)

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the larger of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

var (
	// Shared HTTP transport with optimized settings
	defaultTransport = &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  true, // Disable automatic request compression to prevent "trailing garbage" errors
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
		httpRequest.Header.Set("Content-Type", "application/json; charset=utf-8")
	}

	// Explicitly remove Accept-Encoding header to prevent automatic request compression
	// Response decompression is still handled intelligently based on Content-Encoding header
	// and gzip magic bytes detection in the response processing logic below
	httpRequest.Header.Set("Accept-Encoding", "identity")

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

			// Comprehensive request body analysis for trailing garbage debugging
			requestBody := buf.Bytes()
			qe.Logger.Debug(&libpack_logger.LogMessage{
				Message: "Request body analysis",
				Pairs: map[string]interface{}{
					"body_size":          len(requestBody),
					"contains_jsonQuery": bytes.Contains(requestBody, []byte(`"jsonQuery"`)),
					"contains_base64":    bytes.Contains(requestBody, []byte("eyJ")),
					"first_100_chars":    string(requestBody[:min(100, len(requestBody))]),
					"last_50_chars":      string(requestBody[max(0, len(requestBody)-50):]),
					"is_valid_utf8":      utf8.Valid(requestBody),
					"has_null_bytes":     bytes.Contains(requestBody, []byte{0}),
					"has_control_chars":  hasControlChars(requestBody),
				},
			})

			// Validate request body structure
			if err := validateRequestBody(requestBody, qe.Logger); err != nil {
				qe.Logger.Error(&libpack_logger.LogMessage{
					Message: "Request body validation failed",
					Pairs:   map[string]interface{}{"error": err.Error()},
				})
				return fmt.Errorf("request body validation failed: %w", err)
			}

			// Set the body of the request (plain JSON, no compression)
			httpRequest.Body = io.NopCloser(bytes.NewReader(requestBody))
			httpRequest.ContentLength = int64(len(requestBody))

			// Debug log to confirm request is sent as plain JSON
			qe.Logger.Debug(&libpack_logger.LogMessage{
				Message: "Sending GraphQL request as plain JSON",
				Pairs: map[string]interface{}{
					"content_length":       httpRequest.ContentLength,
					"content_type":         httpRequest.Header.Get("Content-Type"),
					"compression_disabled": true,
				},
			})

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
			var detectionMethod string

			// Intelligent content detection: Check for gzip magic bytes regardless of headers
			// This handles buggy servers that send incorrect Content-Encoding headers
			isGzipData := len(rawData) >= 2 && rawData[0] == 0x1f && rawData[1] == 0x8b

			logPairs := map[string]interface{}{
				"content_encoding_header": encoding,
				"has_gzip_magic_bytes":    isGzipData,
				"data_size":               len(rawData),
			}

			// Only add first_two_bytes if we have data
			if len(rawData) >= 2 {
				logPairs["first_two_bytes"] = fmt.Sprintf("0x%02x 0x%02x", rawData[0], rawData[1])
			} else if len(rawData) == 1 {
				logPairs["first_byte"] = fmt.Sprintf("0x%02x", rawData[0])
			}

			qe.Logger.Debug(&libpack_logger.LogMessage{
				Message: "Content format detection",
				Pairs:   logPairs,
			})

			if isGzipData {
				// Data has gzip magic bytes - attempt gzip decompression
				detectionMethod = "gzip_magic_bytes"
				gzipReader, err := gzip.NewReader(bytes.NewReader(rawData))
				if err != nil {
					qe.Logger.Error(&libpack_logger.LogMessage{
						Message: "Gzip magic bytes detected but failed to create gzip reader",
						Pairs: map[string]interface{}{
							"gzip_reader_error":       err.Error(),
							"content_encoding_header": encoding,
							"detection_method":        detectionMethod,
							"raw_size":                len(rawData),
							"first_two_bytes":         fmt.Sprintf("0x%02x 0x%02x", rawData[0], rawData[1]),
						},
					})
					return fmt.Errorf("gzip reader creation failed: data has gzip magic bytes but gzip.NewReader error: %w", err)
				} else {
					// Successfully created gzip reader, now try to decompress
					defer gzipReader.Close()

					_, copyErr := io.Copy(respBuf, gzipReader)
					if copyErr != nil {
						// Gzip decompression failed - this is an error condition, not a fallback scenario
						qe.Logger.Error(&libpack_logger.LogMessage{
							Message: "Gzip magic bytes detected but decompression failed",
							Pairs: map[string]interface{}{
								"gzip_decompress_error":   copyErr.Error(),
								"raw_size":                len(rawData),
								"content_encoding_header": encoding,
								"detection_method":        detectionMethod,
								"first_two_bytes":         fmt.Sprintf("0x%02x 0x%02x", rawData[0], rawData[1]),
							},
						})
						return fmt.Errorf("gzip decompression failed: data has gzip magic bytes but decompression error: %w", copyErr)
					} else {
						// Successful decompression
						finalData = respBuf.Bytes()
						qe.Logger.Debug(&libpack_logger.LogMessage{
							Message: "Successfully decompressed gzip response using magic byte detection",
							Pairs: map[string]interface{}{
								"compressed_size":         len(rawData),
								"decompressed_size":       len(finalData),
								"content_encoding_header": encoding,
								"detection_method":        detectionMethod,
							},
						})
					}
				}
			} else if encoding == "gzip" {
				// Header claims gzip but no magic bytes - check if client already handled decompression
				detectionMethod = "header_claims_gzip_no_magic_bytes"

				// Check if this looks like valid JSON (likely already decompressed by client)
				trimmed := bytes.TrimSpace(rawData)
				if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
					qe.Logger.Debug(&libpack_logger.LogMessage{
						Message: "Content-Encoding header indicates gzip but data appears to be JSON - likely auto-decompressed by HTTP client",
						Pairs: map[string]interface{}{
							"content_encoding_header": encoding,
							"detection_method":        detectionMethod,
							"data_starts_with_json":   true,
						},
					})
					finalData = rawData
				} else {
					qe.Logger.Warning(&libpack_logger.LogMessage{
						Message: "Content-Encoding header indicates gzip but no magic bytes and data doesn't look like JSON",
						Pairs: map[string]interface{}{
							"content_encoding_header": encoding,
							"detection_method":        detectionMethod,
							"first_few_bytes":         string(rawData[:min(50, len(rawData))]),
							"data_size":               len(rawData),
						},
					})
					return fmt.Errorf("content encoding mismatch: header claims gzip but data has no gzip magic bytes and doesn't appear to be valid JSON")
				}
			} else {
				// No compression indicated by header and no gzip magic bytes
				detectionMethod = "plain_json"
				qe.Logger.Debug(&libpack_logger.LogMessage{
					Message: "Processing as plain JSON response",
					Pairs: map[string]interface{}{
						"content_encoding_header": encoding,
						"detection_method":        detectionMethod,
					},
				})
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
					"data_size":        len(finalData),
					"data_sample":      string(debugData),
					"detection_method": detectionMethod,
				},
			})

			// Unmarshal the final processed data
			err = json.Unmarshal(finalData, &queryResult)
			if err != nil {
				qe.Logger.Error(&libpack_logger.LogMessage{
					Message: "JSON unmarshaling failed",
					Pairs: map[string]interface{}{
						"error":            err.Error(),
						"data_size":        len(finalData),
						"data_sample":      string(debugData),
						"encoding":         encoding,
						"detection_method": detectionMethod,
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

// hasControlChars checks if the byte slice contains control characters that might cause parsing issues
func hasControlChars(data []byte) bool {
	for _, b := range data {
		// Check for control characters except for common whitespace (tab, newline, carriage return)
		if b < 32 && b != 9 && b != 10 && b != 13 {
			return true
		}
	}
	return false
}

// validateRequestBody performs comprehensive validation of the request body
func validateRequestBody(data []byte, logger *libpack_logger.Logger) error {
	// Check if data is valid UTF-8
	if !utf8.Valid(data) {
		return fmt.Errorf("request body contains invalid UTF-8 sequences")
	}

	// Check for null bytes
	if bytes.Contains(data, []byte{0}) {
		return fmt.Errorf("request body contains null bytes")
	}

	// Check if it's valid JSON
	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return fmt.Errorf("request body is not valid JSON: %w", err)
	}

	// Check for jsonQuery field (this should never be present)
	if _, exists := jsonData["jsonQuery"]; exists {
		logger.Error(&libpack_logger.LogMessage{
			Message: "CRITICAL: jsonQuery field found in request body - this causes trailing garbage",
			Pairs: map[string]interface{}{
				"body_size":         len(data),
				"jsonQuery_present": true,
			},
		})
		return fmt.Errorf("jsonQuery field found in request body - this would cause trailing garbage")
	}

	// Check for unexpected base64 data that might indicate double encoding
	if bytes.Contains(data, []byte("eyJ")) {
		logger.Warning(&libpack_logger.LogMessage{
			Message: "Potential base64 encoded JSON detected in request body",
			Pairs: map[string]interface{}{
				"body_size":       len(data),
				"contains_base64": true,
			},
		})
	}

	// Log successful validation
	logger.Debug(&libpack_logger.LogMessage{
		Message: "Request body validation passed",
		Pairs: map[string]interface{}{
			"body_size":          len(data),
			"valid_utf8":         true,
			"valid_json":         true,
			"no_jsonQuery_field": true,
		},
	})

	return nil
}
