package gql

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/avast/retry-go/v4"
)

func (qe *QueryExecutor) executeQuery() ([]byte, error) {
	httpRequest, err := http.NewRequest("POST", qe.endpoint, bytes.NewBuffer(qe.Query))
	if err != nil {
		qe.Logger.Error("Can't create HTTP request;", map[string]interface{}{"error": err})
		return nil, err
	}

	for key, value := range qe.Headers {
		strValue, ok := value.(string)
		if !ok {
			strValue = fmt.Sprintf("%v", value)
		}
		httpRequest.Header.Set(key, strValue)
	}

	var retries_max int

	if qe.Retries {
		qe.Logger.Debug("Retries enabled - setting max retries", map[string]interface{}{"retries": qe.retries_number})
		retries_max = qe.retries_number
	} else {
		qe.Logger.Debug("Retries disabled - setting max retries", map[string]interface{}{"retries": 1})
		retries_max = 1
	}

	var httpResponse *http.Response
	var queryResult queryResults

	err = retry.Do(
		func() error {
			httpResponse, err = qe.client.Do(httpRequest)
			if err != nil {
				qe.Logger.Debug("Error while executing http request", map[string]interface{}{"error": err.Error()})
				return err
			}
			defer func() {
				_, err := io.Copy(io.Discard, httpResponse.Body)
				if err != nil {
					qe.Logger.Debug("Error while discarding http response body;", map[string]interface{}{"error": err.Error()})
				}
				httpResponse.Body.Close()
			}()

			if httpResponse.StatusCode < 200 || httpResponse.StatusCode >= 204 {
				return fmt.Errorf("HTTP error - unacceptable status code: \"%s\" for \"%s\"", httpResponse.Status, httpRequest.URL)
			}

			var reader io.ReadCloser
			encoding := httpResponse.Header.Get("Content-Encoding")
			if encoding == "gzip" {
				reader, err = gzip.NewReader(httpResponse.Body)
				if err != nil {
					qe.Logger.Debug("Error while creating gzip reader;", map[string]interface{}{"error": err.Error()})
					return fmt.Errorf("Error while creating gzip reader: %s", err.Error())
				}
				defer reader.Close()
			} else {
				reader = httpResponse.Body
			}

			body, err := io.ReadAll(reader)
			if err != nil {
				qe.Logger.Debug("Error while reading http response;", map[string]interface{}{"error": err.Error()})
				return fmt.Errorf("Error while reading http response: %s", err.Error())
			}

			err = json.Unmarshal(body, &queryResult)
			if err != nil {
				qe.Logger.Debug("Error while unmarshalling http response;", map[string]interface{}{"error": err.Error()})
				return fmt.Errorf("Error while unmarshalling http response: %s", err.Error())
			}
			return nil
		},
		retry.OnRetry(func(n uint, err error) {
			qe.Logger.Warning("Retrying query", map[string]interface{}{"error": err.Error(), "attempt": n})
		}),
		retry.Attempts(uint(retries_max)),
		retry.DelayType(retry.BackOffDelay),
		retry.Delay(time.Duration(qe.retries_delay)),
		retry.LastErrorOnly(true),
	)

	if err != nil {
		qe.Logger.Debug("Error while executing http request - target server", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	if len(queryResult.Errors) > 0 {
		qe.Logger.Debug("Error while executing query;", map[string]interface{}{"error": queryResult.Errors})
		return nil, fmt.Errorf("Error while executing query: %s", queryResult.Errors)
	}

	if queryResult.Data == nil {
		qe.Logger.Debug("Error while executing query", map[string]interface{}{"error": "no data"})
		return nil, fmt.Errorf("Error while executing query: no data")
	}

	json_data, err := json.Marshal(queryResult.Data)
	if err != nil {
		qe.Logger.Debug("Error while marshalling query result;", map[string]interface{}{"error": err.Error(), "data": queryResult.Data})
		return nil, fmt.Errorf("Error while marshalling query result: %s. Data: %s", err.Error(), queryResult.Data)
	}

	if qe.CacheKey != "no-cache" {
		qe.cache.Set(qe.CacheKey, json_data, qe.CacheTTL)
	}

	return json_data, nil
}
