package gql

import (
	"fmt"
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
				data: `query checkifUserIsAdmin($UserID: bigint, $GroupID: bigint) { tbl_user_group_admins(where: {is_admin: {_eq: "1"}, user_id: {_eq: $UserID}, group_id: {_eq: $GroupID}}) { id is_admin } }`,
				variables: map[string]interface{}{
					"UserID":  37,
					"GroupID": 11007,
				},
			},
			want: `{"query":"query checkifUserIsAdmin($UserID: bigint, $GroupID: bigint) { tbl_user_group_admins(where: {is_admin: {_eq: \"1\"}, user_id: {_eq: $UserID}, group_id: {_eq: $GroupID}}) { id is_admin } }","variables":{"GroupID":11007,"UserID":37}}`,
		},
	}
	for _, tt := range tests {
		assert := assert.New(t)
		t.Run(tt.name, func(t *testing.T) {
			gets := string(queryBuilder(tt.args.data, tt.args.variables))
			assert.Equal(tt.want, gets, fmt.Sprintf("Unexpected query output in test %s", tt.name))
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
	}
	variables := map[string]interface{}{
		"UserID":  37,
		"GroupID": 11007,
	}
	var query = `query checkifUserIsAdmin($UserID: bigint, $GroupID: bigint) {
		tbl_user_group_admins(where: {is_admin: {_eq: "1"}, user_id: {_eq: $UserID}, group_id: {_eq: $GroupID}}) {
			id
			is_admin
		}
	}`
	result := Query(query, variables, headers)
	expected := `{"tbl_user_group_admins":[{"id":109,"is_admin":1}]}`
	assert.Equal(expected, string(result), "Query result execution should be equal")
}
