package gql

import (
	"encoding/json"
	"testing"

	assertions "github.com/stretchr/testify/assert"
)

// TestQueryJSONEncoding verifies that the Query struct serializes correctly
// without including the JsonQuery field, preventing "Trailing garbage" errors
func TestQueryJSONEncoding(t *testing.T) {
	t.Parallel()
	assert := assertions.New(t)
	client := CreateTestClient()

	query := &Query{
		Query: "query GetUser($id: ID!) { user(id: $id) { id name } }",
		Variables: map[string]interface{}{
			"id": "123",
		},
	}

	// Set JsonQuery to simulate the compiled query
	query.JsonQuery = client.convertToJSON(query)

	// Serialize the query to JSON
	jsonBytes, err := json.Marshal(query)
	assert.NoError(err)

	// Parse the JSON back to verify structure
	var parsed map[string]interface{}
	err = json.Unmarshal(jsonBytes, &parsed)
	assert.NoError(err)

	// Verify that only query and variables are included, NOT jsonQuery
	assert.Contains(parsed, "query")
	assert.Contains(parsed, "variables")
	assert.NotContains(parsed, "jsonQuery", "JsonQuery field should be excluded from JSON serialization")

	// Verify the content is correct
	assert.Equal(query.Query, parsed["query"])
	assert.Equal(query.Variables["id"], parsed["variables"].(map[string]interface{})["id"])

	// Verify the JSON is valid GraphQL format
	assert.True(json.Valid(jsonBytes), "Generated JSON should be valid")
}

// TestQueryJSONEncodingPreventsTrailingGarbage ensures the fix prevents the specific error
func TestQueryJSONEncodingPreventsTrailingGarbage(t *testing.T) {
	t.Parallel()
	assert := assertions.New(t)
	client := CreateTestClient()

	// Create a query with complex variables
	query := &Query{
		Query: "mutation storeMessage($content: String!, $userID: Int!) { insert_message(content: $content, user_id: $userID) { id } }",
		Variables: map[string]interface{}{
			"content": "Test message with special chars: {}[]\"'",
			"userID":  12345,
		},
	}

	// Compile the query (this sets JsonQuery)
	query.JsonQuery = client.convertToJSON(query)

	// Serialize to JSON - this should NOT include JsonQuery field
	jsonBytes, err := json.Marshal(query)
	assert.NoError(err)

	// The JSON should be valid and parseable
	var graphqlRequest map[string]interface{}
	err = json.Unmarshal(jsonBytes, &graphqlRequest)
	assert.NoError(err)

	// Should have exactly 2 fields: query and variables
	assert.Len(graphqlRequest, 2)
	assert.Contains(graphqlRequest, "query")
	assert.Contains(graphqlRequest, "variables")

	// The JSON should be suitable for sending to a GraphQL server
	jsonString := string(jsonBytes)
	assert.NotContains(jsonString, "jsonQuery", "JSON should not contain jsonQuery field")
	assert.NotContains(jsonString, "eyJ", "JSON should not contain base64 encoded data")
}
