package gql

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type TestSuite struct {
	suite.Suite
}

func (suite *TestSuite) SetupTest() {
	os.Unsetenv("GRAPHQL_ENDPOINT")
}

func TestSuiteRun(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

func mockGraphQLServerResponses(responseQuery string) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte(responseQuery))
	}))
	GraphQLUrl = server.URL + "/v1/graphql"
	return server
}

func (suite *TestSuite) Test_queryBuilder() {
	type args struct {
		data      string
		variables interface{}
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Simple query",
			args: args{
				data:      `query listUserBots { tbl_bots { bot_name } }`,
				variables: nil,
			},
			want: `{"query":"query listUserBots { tbl_bots { bot_name } }","variables":null}`,
		},
		{
			name: "Advanced query",
			args: args{
				data: `query checkifUserIsAdmin($UserID: bigint, $GroupID: bigint) { tbl_user_group_admins(where: {is_admin: {_eq: true}, user_id: {_eq: $UserID}, group_id: {_eq: $GroupID}}) { id is_admin } }`,
				variables: map[string]interface{}{
					"UserID":  37,
					"GroupID": 11007,
				},
			},
			want: `{"query":"query checkifUserIsAdmin($UserID: bigint, $GroupID: bigint) { tbl_user_group_admins(where: {is_admin: {_eq: true}, user_id: {_eq: $UserID}, group_id: {_eq: $GroupID}}) { id is_admin } }","variables":{"GroupID":11007,"UserID":37}}`,
		},
	}
	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			gets, err := queryBuilder(tt.args.data, tt.args.variables)
			assert.Equal(suite.T(), tt.want, string(gets), "Unexpected query output in test: "+tt.name)
			assert.Nil(suite.T(), err)
		},
		)
	}
}

func (suite *TestSuite) Test_queryAgainstDatabaseExecution() {
	os.Unsetenv("GRAPHQL_ENDPOINT")
	jsonResponse := `{
		"data": {"tbl_user_group_admins":[{"id":109,"is_admin":true}]}
	}`
	server := mockGraphQLServerResponses(jsonResponse)
	defer server.Close()
	headers := map[string]interface{}{
		"x-hasura-user-id":   37,
		"x-hasura-user-uuid": "bde3262e-b42e-4151-ac10-d43fb38f44a5",
		"Authorization":      "bearer LaPotatoDiBanani",
	}
	variables := map[string]interface{}{
		"UserID":  37,
		"GroupID": 11007,
	}
	var query = `query checkifUserIsAdmin($UserID: bigint, $GroupID: bigint) {
		tbl_user_group_admins(where: {is_admin: {_eq: true}, user_id: {_eq: $UserID}, group_id: {_eq: $GroupID}}) {
			id
			is_admin
		}
	}`
	result, err := Query(query, variables, headers)
	if err != nil {
		suite.T().Log("Query execution errored. Is GQL server up?")
	}
	expected := `{"tbl_user_group_admins":[{"id":109,"is_admin":true}]}`
	assert.Equal(suite.T(), expected, string(result), "Query result execution should be equal")
}

func ExampleQuery() {
	variables := map[string]interface{}{
		"fileHash": "123deadc0w321",
	}
	var query = `query searchFileKnown($fileHash: String) {
		tbl_file_scans(where: {file_hash: {_eq: $fileHash}}) {
			porn
			racy
			violence
			virus
		}
	}`
	result, err := Query(query, variables, nil)
	if err != nil {
		fmt.Println("Query error", err)
		return
	}
	fmt.Println(result)
}

func (suite *TestSuite) Test_initialize() {
	tests := []struct {
		name           string
		expected       string
		env_endpoint   string
		local_endpoint string
	}{
		{
			name:         "Setting env variable for endpoint (ENV)",
			env_endpoint: "new-hasura.local/v1/graphql",
			expected:     "new-hasura.local/v1/graphql",
		},
		{
			name:           "Setting default endpoint (ENV)",
			env_endpoint:   "",
			local_endpoint: "",
			expected:       "http://127.0.0.1:9090/v1/graphql",
		},
		{
			name:         "Setting custom endpoint (ENV)",
			env_endpoint: "http://127.0.0.1:8080/v1/graphql",
			expected:     "http://127.0.0.1:8080/v1/graphql",
		},
		{
			name:           "Setting custom endpoint (VAR)",
			local_endpoint: "http://127.0.0.1:8090/v1/graphql",
			expected:       "http://127.0.0.1:8090/v1/graphql",
		},
		{
			name:           "Setting custom endpoint (VAR)",
			local_endpoint: "hasura-def.local/v1/graphql",
			expected:       "hasura-def.local/v1/graphql",
		},
	}
	backupEnv, present := os.LookupEnv("GRAPHQL_ENDPOINT")
	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			os.Unsetenv("GRAPHQL_ENDPOINT")
			GraphQLUrl = ""
			if tt.env_endpoint != "" {
				os.Setenv("GRAPHQL_ENDPOINT", tt.env_endpoint)
				t.Log(tt.env_endpoint, os.Getenv("GRAPHQL_ENDPOINT"))
			}
			if tt.local_endpoint != "" {
				os.Unsetenv("GRAPHQL_ENDPOINT")
				GraphQLUrl = tt.local_endpoint
			}
			prepare()
			assert.Equal(suite.T(), tt.expected, GraphQLUrl, "Unexpected value of env variable: "+tt.name)
		})
	}
	if present {
		os.Setenv("GRAPHQL_ENDPOINT", backupEnv)
	}
}
