package gql

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/tidwall/gjson"
	"github.com/valyala/fasthttp"
)

var GraphQLUrl = "http://127.0.0.1:9090/v1/graphql"

type requestBase struct {
	Query     string      `json:"query"`
	Variables interface{} `json:"variables"`
}

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

func Query(query string, variables interface{}, headers map[string]interface{}) (string, error) {
	var err error
	readyQuery, err := queryBuilder(query, variables)
	if err != nil {
		return "", err
	}
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.Header.SetContentType("application/json")
	for header, value := range headers {
		req.Header.Set(fmt.Sprintf("%v", header), fmt.Sprintf("%v", value))
	}
	req.Header.SetMethodBytes([]byte("POST"))
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
