package gql

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/goccy/go-json"
	libpack_logger "github.com/lukaszraczylo/go-simple-graphql/logging"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

func (qe *QueryExecutor) executeQuery() ([]byte, error) {
	// Reuse buffer from pool to avoid allocations
	buf := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buf)
	buf.Reset()

	_, err := buf.Write(qe.Query)
	if err != nil {
		qe.Logger.Error(&libpack_logger.LogMessage{
			Message: "Can't write to buffer",
			Pairs:   map[string]interface{}{"error": err.Error()},
		})
		return nil, err
	}

	httpRequest, err := http.NewRequest(http.MethodPost, qe.endpoint, buf)
	if err != nil {
		qe.Logger.Error(&libpack_logger.LogMessage{
			Message: "Can't create HTTP request",
			Pairs:   map[string]interface{}{"error": err.Error()},
		})
		return nil, err
	}

	for key, value := range qe.Headers {
		httpRequest.Header.Set(key, fmt.Sprint(value))
	}

	retriesMax := 1
	if qe.Retries {
		qe.Logger.Debug(&libpack_logger.LogMessage{
			Message: "Retries enabled",
			Pairs:   map[string]interface{}{"retries": qe.retries_number},
		})
		retriesMax = qe.retries_number
	} else {
		qe.Logger.Debug(&libpack_logger.LogMessage{
			Message: "Retries disabled",
			Pairs:   map[string]interface{}{"retries": 1},
		})
	}

	var queryResult queryResults
	err = retry.Do(
		func() error {
			httpResponse, err := qe.client.Do(httpRequest)
			if err != nil {
				qe.Logger.Debug(&libpack_logger.LogMessage{
					Message: "Error executing HTTP request",
					Pairs:   map[string]interface{}{"error": err.Error()},
				})
				return err
			}
			defer func() {
				_, err := io.Copy(io.Discard, httpResponse.Body)
				if err != nil {
					qe.Logger.Debug(&libpack_logger.LogMessage{
						Message: "Error discarding HTTP response body",
						Pairs:   map[string]interface{}{"error": err.Error()},
					})
				}
				httpResponse.Body.Close()
			}()

			if httpResponse.StatusCode < http.StatusOK || httpResponse.StatusCode >= http.StatusNoContent {
				return fmt.Errorf("HTTP error - status code: %s for %s", httpResponse.Status, httpRequest.URL)
			}

			var reader io.ReadCloser
			encoding := httpResponse.Header.Get("Content-Encoding")
			if encoding == "gzip" {
				reader, err = gzip.NewReader(httpResponse.Body)
				if err != nil {
					qe.Logger.Debug(&libpack_logger.LogMessage{
						Message: "Error creating gzip reader",
						Pairs:   map[string]interface{}{"error": err.Error()},
					})
					return fmt.Errorf("error creating gzip reader: %w", err)
				}
				defer reader.Close()
			} else {
				reader = httpResponse.Body
			}

			body, err := io.ReadAll(reader)
			if err != nil {
				qe.Logger.Debug(&libpack_logger.LogMessage{
					Message: "Error reading HTTP response",
					Pairs:   map[string]interface{}{"error": err.Error()},
				})
				return fmt.Errorf("error reading HTTP response: %w", err)
			}

			err = json.Unmarshal(body, &queryResult)
			if err != nil {
				qe.Logger.Debug(&libpack_logger.LogMessage{
					Message: "Error unmarshalling HTTP response",
					Pairs:   map[string]interface{}{"error": err.Error()},
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
