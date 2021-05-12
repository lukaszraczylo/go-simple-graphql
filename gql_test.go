package gql

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_queryBuilder(t *testing.T) {
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
	GraphQLUrl = "http://hasura.local/v1/graphql"
	assert := assert.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gets, err := queryBuilder(tt.args.data, tt.args.variables)
			assert.Equal(tt.want, string(gets), "Unexpected query output in test: "+tt.name)
			assert.Nil(err)
		},
		)
	}
}

func Test_queryAgainstDatabaseExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short / CI mode")
	}
	assert := assert.New(t)
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
	GraphQLUrl = "http://hasura.local/v1/graphql"
	result, err := Query(query, variables, headers)
	if err != nil {
		t.Log("Query execution errored. Is GQL server up?")
	}
	expected := `{"tbl_user_group_admins":[{"id":109,"is_admin":true}]}`
	assert.Equal(expected, string(result), "Query result execution should be equal")
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

func Test_initialize(t *testing.T) {
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
			local_endpoint: "http://127.0.0.1:8080/v1/graphql",
			expected:       "http://127.0.0.1:8080/v1/graphql",
		},
		{
			name:           "Setting custom endpoint (VAR)",
			local_endpoint: "hasura-def.local/v1/graphql",
			expected:       "hasura-def.local/v1/graphql",
		},
	}
	assert := assert.New(t)
	backupEnv, present := os.LookupEnv("GRAPHQL_ENDPOINT")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			assert.Equal(tt.expected, GraphQLUrl, "Unexpected value of env variable: "+tt.name)
			os.Unsetenv("GRAPHQL_ENDPOINT")
		})
	}
	if present {
		os.Setenv("GRAPHQL_ENDPOINT", backupEnv)
	}
}
