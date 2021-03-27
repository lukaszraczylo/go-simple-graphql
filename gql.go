package gql

import (
	"bytes"
	"encoding/json"
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

func Query(query string, variables interface{}, headers map[string]interface{}) string {
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
		panic("Unable to execute request")
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
	}
	fasthttp.ReleaseResponse(res)
	return toReturn.String()
}

// func main() {
// headers := map[string]interface{}{
// 	"x-hasura-user-id":   37,
// 	"x-hasura-user-uuid": "bde3262e-b42e-4151-ac10-d43fb38f44a5",
// }
// variables := map[string]interface{}{
// 	"UserID":  37,
// 	"GroupID": 11007,
// }
// var query = `query checkifUserIsAdmin($UserID: bigint, $GroupID: bigint) {
// 	tbl_user_group_admins(where: {is_admin: {_eq: "1"}, user_id: {_eq: $UserID}, group_id: {_eq: $GroupID}}) {
// 		id
// 		is_admin
// 	}
// }`
// result := Query(query, variables, headers)
// fmt.Println(result)
// }
