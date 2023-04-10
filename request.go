package gql

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/avast/retry-go/v4"
)

func (c *BaseClient) executeQuery(query []byte, headers any) (result any, err error) {
	var queryResult queryResults
	httpRequest, err := http.NewRequest("POST", c.endpoint, bytes.NewBuffer(query))
	if err != nil {
		c.Logger.Error(c, "Error while creating http request;", "error", err.Error())
		return
	}

	for key, value := range headers.(map[string]interface{}) {
		httpRequest.Header.Add(key, fmt.Sprintf("%s", value))
	}

	var retries_available = c.retries.max
	if !c.retries.enabled {
		retries_available = 1
	}

	var httpResponse *http.Response

	err = retry.Do(
		func() error {
			httpResponse, err = c.client.Do(httpRequest)
			if err != nil {
				c.Logger.Error(c, "Error while executing http request;", "error", err.Error())
				return err
			}
			defer io.Copy(io.Discard, httpResponse.Body) // equivalent to `cp body /dev/null`
			defer httpResponse.Body.Close()

			if httpResponse.StatusCode <= 200 && httpResponse.StatusCode >= 204 {
				return err
			}

			body, err := ioutil.ReadAll(httpResponse.Body)
			if err != nil {
				c.Logger.Error(c, "Error while reading http response;", "error", err.Error())
				return err
			}

			err = json.Unmarshal(body, &queryResult)
			if err != nil {
				c.Logger.Error(c, "Error while unmarshalling http response;", "error", err.Error())
				return err
			}

			return nil
		},
		retry.OnRetry(func(n uint, err error) {
			c.Logger.Error(c, "Retrying query", "error", err.Error())
		}),
		retry.Attempts(uint(retries_available)),
		retry.DelayType(retry.BackOffDelay),
		retry.Delay(time.Duration(c.retries.delay)*time.Second),
		retry.LastErrorOnly(true),
	)

	if len(queryResult.Errors) > 0 {
		return nil, fmt.Errorf("%v", queryResult.Errors)
	}
	return queryResult.Data, nil
}

func (q *queryExecutor) execute() {
	if q.should_cache {
		cachedResponse := q.client.cacheLookup(q.hash)
		if cachedResponse != nil {
			q.client.Logger.Debug(q.client, "Found cached response")
			q.result.data = q.client.decodeResponse(cachedResponse)
			return // return cached response
		} else {
			q.client.Logger.Debug(q.client, "No cached response found")
		}
	}

	if q.retries_enabled {
		q.client.Logger.Debug(q.client, "Retries enabled")
		q.client.retries.enabled = true
	}

	response, err := q.client.executeQuery(q.query, q.headers)
	if err != nil {
		q.client.Logger.Error(q.client, "Error while executing query;", "error", err.Error())
		q.result.errors = err
		return
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		q.client.Logger.Error(q.client, "Error while converting to json;", "error", err.Error())
		q.result.errors = err
		return
	}

	if q.should_cache {
		q.client.cache.client.Set(q.hash, jsonData, time.Duration(q.client.cache.ttl)*time.Second)
	}

	q.result.data = q.client.decodeResponse(jsonData)
}

func (q *queryExecutor) done() {
	q = nil
}
