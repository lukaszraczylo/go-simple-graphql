package gql

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/lukaszraczylo/pandati"
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

	// j := new(bytes.Buffer)
	j2, err := json.Marshal(qb)
	if err != nil {
		g.Log.Critical("Unable to marshal the query", map[string]interface{}{"_error": err.Error(), "_query": queryContent, "_variables": queryVariables})
		return []byte{}, err
	}
	// if err = json.Compact(j, j2); err != nil {
	// 	g.Log.Critical("Unable to compact the query", map[string]interface{}{"error": err.Error()})
	// 	return []byte{}, err
	// }
	return j2, err
}

func (g *GraphQL) Query(queryContent string, queryVariables interface{}, queryHeaders map[string]interface{}) (responseContent string, err error) {
	g.Log.Debug("Query details", map[string]interface{}{"_query": queryContent, "_variables": queryVariables})
	query, err := g.queryBuilder(queryContent, queryVariables)
	if err != nil {
		g.Log.Error("Unable to build the query", map[string]interface{}{"_error": err.Error()})
		return "", err
	}
	httpRequest, err := http.NewRequest("POST", g.Endpoint, bytes.NewBuffer(query))
	httpRequest.Header.Add("Content-Type", "application/json")
	for header, value := range queryHeaders {
		httpRequest.Header.Add(fmt.Sprintf("%v", header), fmt.Sprintf("%v", value))
	}

	httpResponse, err := g.HttpClient.Do(httpRequest)
	if err != nil {
		g.Log.Error("Unable to send the query", map[string]interface{}{"_error": err.Error()})
		return "", err
	}
	defer io.Copy(ioutil.Discard, httpResponse.Body)
	defer httpResponse.Body.Close()

	body, err := ioutil.ReadAll(httpResponse.Body)
	if err != nil {
		g.Log.Critical("Unable to read the response", map[string]interface{}{"_error": err.Error()})
		return "", err
	}
	var queryResult *queryResults
	err = json.Unmarshal(body, &queryResult)
	if err != nil {
		g.Log.Error("Unable to unmarshal the query", map[string]interface{}{"_error": err.Error()})
		return "", err
	}

	if !pandati.IsZero(queryResult.Errors) {
		g.Log.Error("Query returned error", map[string]interface{}{"_query": queryContent, "_variables": queryVariables, "_error": fmt.Sprintf("%v", queryResult.Errors)})
		return "", fmt.Errorf("%v", queryResult.Errors[0].Message)
	}

	if pandati.IsZero(queryResult.Data) {
		g.Log.Error("Query returned no data", map[string]interface{}{"_query": queryContent, "_variables": queryVariables})
		return "", errors.New("Query returned no data")
	}

	responseContent, err = json.MarshalToString(queryResult.Data)
	if err != nil {
		g.Log.Error("Invalid data result", map[string]interface{}{"_query": queryContent, "_variables": queryVariables, "_result": responseContent})
		return "", err
	}

	return
}
