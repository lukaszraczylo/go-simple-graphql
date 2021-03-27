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

func queryBuilder(data string, variables interface{}) []byte {
	var qb requestBase
	qb.Query = data
	qb.Variables = variables
	j := new(bytes.Buffer)
	j2, _ := json.Marshal(qb)
	if err := json.Compact(j, j2); err != nil {
		fmt.Println(err)
		panic("Unable to process the query")
	}
	// Attemt to decrease size of the generated JSON
	compactedBuffer := new(bytes.Buffer)
	err := json.Compact(compactedBuffer, j.Bytes())
	if err != nil {
		fmt.Println(err)
		panic("Unable to compress json")
	}
	return []byte(compactedBuffer.String())
}

func Query(query string, variables interface{}, headers map[string]interface{}) (string, error) {
	var err error
	readyQuery := queryBuilder(query, variables)
	req := fasthttp.AcquireRequest()
	req.Header.SetContentType("application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	for header, value := range headers {
		req.Header.Set(fmt.Sprintf("%v", header), fmt.Sprintf("%v", value))
	}
	req.Header.SetMethodBytes([]byte("POST"))
	req.SetBody(readyQuery)
	req.SetRequestURI(GraphQLUrl)

	res := fasthttp.AcquireResponse()
	if err := fasthttp.Do(req, res); err != nil {
		return "", err
	}
	fasthttp.ReleaseRequest(req)

	contentEncoding := res.Header.Peek("Content-Encoding")
	var body []byte
	if bytes.EqualFold(contentEncoding, []byte("gzip")) {
		body, _ = res.BodyGunzip()
	} else {
		body = res.Body()
	}

	toReturn := gjson.Get(string(body), "data")
	if toReturn.String() == "" {
		fmt.Println("Probably not what you expect:", string(body))
		return string(body), errors.New(fmt.Sprintf("Probably not what you expect: %s", body))
	}
	fasthttp.ReleaseResponse(res)
	return toReturn.String(), err
}
