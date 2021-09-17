// Package / library or rather wrapper for GraphQL queries execution in the painless way.
// Using it is as easy as copy / paste the query itself and set appropriate variables in.
//
// Library supports basic error reporting on unsuccessful queries and setting appropriate headers.
package gql

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/lukaszraczylo/pandati"
	"github.com/tidwall/gjson"
	"github.com/valyala/fasthttp"
)

// Endpoint of your GraphQL server to query
// this variable can be overwritten by setting env variable, for example:
// GRAPHQL_ENDPOINT=http://hasura.local/v1/graphql
var GraphQLUrl string

type requestBase struct {
	Query     string      `json:"query"`
	Variables interface{} `json:"variables"`
}

func prepare() {
	value, present := os.LookupEnv("GRAPHQL_ENDPOINT")
	if present {
		GraphQLUrl = value
	} else if !present && pandati.IsZero(GraphQLUrl) {
		GraphQLUrl = "http://127.0.0.1:9090/v1/graphql"
		fmt.Println("Setting default endpoint", GraphQLUrl)
	} else {
		fmt.Println("GraphQL endpoint not set.")
	}
}

// queryBuilder takes query data (string) and variables (interface) as a parameter and assembles
// graphQL query. It also compacts the JSON result. Function returns query as []byte and error if anything went wrong.
func queryBuilder(data string, variables interface{}) ([]byte, error) {
	var err error
	var qb = &requestBase{
		Query:     data,
		Variables: variables,
	}
	j := new(bytes.Buffer)
	j2, err := json.Marshal(qb)
	if err != nil {
		return []byte{}, err
	}
	if err = json.Compact(j, j2); err != nil {
		return []byte{}, err
	}
	return j.Bytes(), err
}

// Query allows you to execute the GraphQL query.
// Query is a string ( copy paste from Hasura or any other query builder )
// Variables and Headers are maps of strings ( see the example )
// Function returns whatever specified query returns and/or error.
func Query(query string, variables interface{}, headers map[string]interface{}) (string, error) {
	prepare()
	var err error
	readyQuery, err := queryBuilder(query, variables)
	if err != nil {
		return "", err
	}
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.Header.SetContentType("application/json")
	for header, value := range headers {
		req.Header.Add(fmt.Sprintf("%v", header), fmt.Sprintf("%v", value))
	}
	req.Header.SetMethod("POST")
	req.SetBody(readyQuery)
	req.SetRequestURI(GraphQLUrl)
	res := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(res)
	if err := fasthttp.Do(req, res); err != nil {
		return "", err
	}
	body := res.Body()
	toReturn := gjson.Get(string(body), "data")
	if toReturn.String() == "" {
		err = errors.New(string(body))
		return "", err
	}
	return toReturn.String(), err
}
