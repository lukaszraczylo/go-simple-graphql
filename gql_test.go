package gql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_simpleQueryBuilder(t *testing.T) {
	assert := assert.New(t)
	q := `query listUserBots {
		tbl_bots {
    	bot_name
  	}
	}`
	result := queryBuilder(q, nil)
	expected := `{"query":"query listUserBots {\n\t\ttbl_bots {\n    \tbot_name\n  \t}\n\t}","variables":null}`
	assert.Equal(expected, string(result), "Produced simple query should be equal")
}

func Test_advancedQueryBuilder(t *testing.T) {
	assert := assert.New(t)
	q := `query checkifUserIsAdmin($UserID: bigint, $GroupID: bigint) {
		tbl_user_group_admins(where: {is_admin: {_eq: "1"}, user_id: {_eq: $UserID}, group_id: {_eq: $GroupID}}) {
			id
			is_admin
		}
	}`
	v := map[string]interface{}{
		"UserID":  37,
		"GroupID": 11007,
	}
	result := queryBuilder(q, v)
	expected := `{"query":"query checkifUserIsAdmin($UserID: bigint, $GroupID: bigint) {\n\t\ttbl_user_group_admins(where: {is_admin: {_eq: \"1\"}, user_id: {_eq: $UserID}, group_id: {_eq: $GroupID}}) {\n\t\t\tid\n\t\t\tis_admin\n\t\t}\n\t}","variables":{"GroupID":11007,"UserID":37}}`
	assert.Equal(expected, string(result), "Produced advanced query should be equal")
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
