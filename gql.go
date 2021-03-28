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
	var qb requestBase
	qb.Query = data
	qb.Variables = variables
	j := new(bytes.Buffer)
	j2, _ := json.Marshal(qb)
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
	req.Header.SetContentType("application/json")
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
	body := res.Body()
	toReturn := gjson.Get(string(body), "data")
	if toReturn.String() == "" {
		err = errors.New(string(body))
		return "", err
	}
	fasthttp.ReleaseResponse(res)
	return toReturn.String(), err
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
