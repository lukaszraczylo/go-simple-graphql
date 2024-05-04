package gql

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/goccy/go-json"
)

func (qe *QueryExecutor) executeQuery() ([]byte, error) {
	httpRequest, err := http.NewRequest(http.MethodPost, qe.endpoint, bytes.NewBuffer(qe.Query))
	if err != nil {
		qe.Logger.Error("Can't create HTTP request", map[string]interface{}{"error": err})
		return nil, err
	}

	for key, value := range qe.Headers {
		httpRequest.Header.Set(key, fmt.Sprint(value))
	}

	retriesMax := 1
	if qe.Retries {
		qe.Logger.Debug("Retries enabled", map[string]interface{}{"retries": qe.retries_number})
		retriesMax = qe.retries_number
	} else {
		qe.Logger.Debug("Retries disabled", map[string]interface{}{"retries": 1})
	}

	var queryResult queryResults
	err = retry.Do(
		func() error {
			httpResponse, err := qe.client.Do(httpRequest)
			if err != nil {
				qe.Logger.Debug("Error executing HTTP request", map[string]interface{}{"error": err.Error()})
				return err
			}
			defer func() {
				_, err := io.Copy(io.Discard, httpResponse.Body)
				if err != nil {
					qe.Logger.Debug("Error discarding HTTP response body", map[string]interface{}{"error": err.Error()})
				}
				httpResponse.Body.Close()
			}()

			if httpResponse.StatusCode < http.StatusOK || httpResponse.StatusCode >= http.StatusNoContent {
				return fmt.Errorf("HTTP error - status code: %s for %s", httpResponse.Status, httpRequest.URL)
			}

			var reader io.ReadCloser
			encoding := httpResponse.Header.Get("Content-Encoding")
			switch encoding {
			case "gzip":
				reader, err = gzip.NewReader(httpResponse.Body)
				if err != nil {
					qe.Logger.Debug("Error creating gzip reader", map[string]interface{}{"error": err.Error()})
					return fmt.Errorf("error creating gzip reader: %w", err)
				}
				defer reader.Close()
			default:
				reader = httpResponse.Body
			}

			body, err := io.ReadAll(reader)
			if err != nil {
				qe.Logger.Debug("Error reading HTTP response", map[string]interface{}{"error": err.Error()})
				return fmt.Errorf("error reading HTTP response: %w", err)
			}

			err = json.Unmarshal(body, &queryResult)
			if err != nil {
				qe.Logger.Debug("Error unmarshalling HTTP response", map[string]interface{}{"error": err.Error()})
				return fmt.Errorf("error unmarshalling HTTP response: %w", err)
			}

			return nil
		},
		retry.OnRetry(func(n uint, err error) {
			qe.Logger.Warning("Retrying query", map[string]interface{}{"error": err.Error(), "attempt": n})
		}),
		retry.Attempts(uint(retriesMax)),
		retry.DelayType(retry.BackOffDelay),
		retry.Delay(time.Duration(qe.retries_delay)),
		retry.LastErrorOnly(true),
	)
	if err != nil {
		qe.Logger.Debug("Error executing HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	if len(queryResult.Errors) > 0 {
		qe.Logger.Debug("Error executing query", map[string]interface{}{"error": queryResult.Errors})
		return nil, fmt.Errorf("error executing query: %s", queryResult.Errors)
	}

	if queryResult.Data == nil {
		qe.Logger.Debug("Error executing query", map[string]interface{}{"error": "no data"})
		return nil, errors.New("error executing query: no data")
	}

	jsonData, err := json.Marshal(queryResult.Data)
	if err != nil {
		qe.Logger.Debug("Error marshalling query result", map[string]interface{}{"error": err.Error(), "data": queryResult.Data})
		return nil, fmt.Errorf("error marshalling query result: %w. Data: %s", err, queryResult.Data)
	}

	if qe.CacheKey != "no-cache" {
		qe.cache.Set(qe.CacheKey, jsonData, qe.CacheTTL)
	}

	return jsonData, nil
}
