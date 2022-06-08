package gql

import (
	"bytes"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/lukaszraczylo/pandati"
	retry "github.com/sethvargo/go-retry"
)

type requestBase struct {
	Query     string      `json:"query"`
	Variables interface{} `json:"variables"`
}

type queryResults struct {
	Data   interface{} `json:"data"`
	Errors []struct {
		Message interface{} `json:"message"`
	} `json:"errors"`
}

func (g *GraphQL) queryBuilder(queryContent string, queryVariables interface{}) ([]byte, error) {
	var qb = &requestBase{
		Query:     queryContent,
		Variables: queryVariables,
	}

	j2, err := json.Marshal(qb)
	if err != nil {
		g.Log.Critical("Unable to marshal the query", map[string]interface{}{"_error": err.Error(), "_query": queryContent, "_variables": queryVariables})
		return []byte{}, err
	}
	return j2, err
}

func (g *GraphQL) Query(queryContent string, queryVariables interface{}, queryHeaders map[string]interface{}) (responseContent string, err error) {
	ctx := context.Background()
	g.Log.Debug("Query details", map[string]interface{}{"_query": queryContent, "_variables": queryVariables})
	query, err := g.queryBuilder(queryContent, queryVariables)
	if err != nil {
		g.Log.Error("Unable to build the query", map[string]interface{}{"_error": err.Error()})
		return "", err
	}

	var body []byte
	var queryResult *queryResults
	queryHash := fmt.Sprintf("%x", md5.Sum(query))

	if g.Cache {
		g.Log.Debug("Checking the cache for the query", map[string]interface{}{"_query": queryHash})
		if entry, entryInfo, err := g.CacheStore.GetWithInfo(queryHash); err == nil {
			g.Log.Debug("Found the query in the cache", map[string]interface{}{"_query": queryHash})
			if pandati.IsZero(entryInfo.EntryStatus) {
				return string(entry), nil
			}
		} else {
			g.Log.Debug("Unable to find the query in the cache", map[string]interface{}{"_query": queryHash, "_error": err.Error()})
		}
	}

	httpRequest, err := http.NewRequest("POST", g.Endpoint, bytes.NewBuffer(query))
	// httpRequest.Header.Set("Content-Type", "application/json")
	if err != nil {
		g.Log.Error("Unable to create the request", map[string]interface{}{"_error": err.Error()})
		return "", err
	}

	for header, value := range queryHeaders {
		httpRequest.Header.Add(fmt.Sprintf("%v", header), fmt.Sprintf("%v", value))
	}

	var httpResponse *http.Response

	retry.Do(ctx, g.BackoffSetup, func(ctx context.Context) error {
		httpResponse, err = g.HttpClient.Do(httpRequest)
		if err != nil {
			g.Log.Debug("Unable to send the query RETRY", map[string]interface{}{"_error": err.Error()})
			return err
		}
		if httpResponse.StatusCode <= 200 && httpResponse.StatusCode >= 204 {
			return retry.RetryableError(errors.New(fmt.Sprintf("%v", httpResponse.StatusCode)))
		}

		defer io.Copy(ioutil.Discard, httpResponse.Body)
		defer httpResponse.Body.Close()

		body, err = ioutil.ReadAll(httpResponse.Body)
		if err != nil {
			g.Log.Debug("Unable to read the response", map[string]interface{}{"_error": err.Error()})
			return retry.RetryableError(errors.New(fmt.Sprintf("RETRY: body parsing %v", httpResponse.StatusCode)))
		}

		err = json.Unmarshal(body, &queryResult)
		if err != nil {
			g.Log.Debug("Unable to unmarshal the query", map[string]interface{}{"_error": err.Error()})
			return retry.RetryableError(errors.New(fmt.Sprintf("RETRY: body unmarshal %v", httpResponse.StatusCode)))
		}
		return nil
	})

	if !pandati.IsZero(queryResult.Errors) {
		g.Log.Error("Query returned error", map[string]interface{}{"_query": queryContent, "_variables": queryVariables, "_error": fmt.Sprintf("%v", queryResult.Errors), "_response_code": httpResponse.StatusCode})
		return "", fmt.Errorf("%v", queryResult.Errors)
	}

	if pandati.IsZero(queryResult.Data) {
		g.Log.Error("Query returned no data", map[string]interface{}{"_query": queryContent, "_variables": queryVariables, "_response_code": httpResponse.StatusCode})
		return "", errors.New("Query returned no data")
	}

	responseContent, err = json.MarshalToString(queryResult.Data)
	if err != nil {
		g.Log.Error("Invalid data result", map[string]interface{}{"_query": queryContent, "_variables": queryVariables, "_result": responseContent, "_response_code": httpResponse.StatusCode})
		return "", err
	}

	if g.Cache {
		g.Log.Debug("Caching the query", map[string]interface{}{"_query": queryHash})
		if queryContent[0:5] == "query" {
			err = g.CacheStore.Set(queryHash, []byte(responseContent))
			if err != nil {
				g.Log.Error("Unable to cache the query", map[string]interface{}{"_query": queryHash, "_error": err.Error()})
			}
		}
	}

	return
}
