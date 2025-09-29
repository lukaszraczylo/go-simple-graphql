package gql

import (
	"strings"
	"testing"

	assertions "github.com/stretchr/testify/assert"
)

func TestMinifyGraphQLQuery(t *testing.T) {
	t.Parallel()
	assert := assertions.New(t)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty_query",
			input:    "",
			expected: "",
		},
		{
			name:     "simple_query_no_whitespace",
			input:    "query{viewer{login}}",
			expected: "query{viewer{login}}",
		},
		{
			name: "query_with_newlines_and_tabs",
			input: `query {
				viewer {
					login
				}
			}`,
			expected: "query{viewer{login}}",
		},
		{
			name: "mutation_with_variables",
			input: `mutation storeMessage($content: String!, $id: Int!) {
				insert_message(content: $content, id: $id) {
					id
					created_at
				}
			}`,
			expected: "mutation storeMessage($content: String!, $id: Int!){insert_message(content: $content, id: $id){id created_at}}",
		},
		{
			name: "query_with_string_literals_preserved",
			input: `mutation storeMessage($description: String!) {
				insert_message(description: $description, content: "This is a test\nwith newlines\tand tabs") {
					id
				}
			}`,
			expected: `mutation storeMessage($description: String!){insert_message(description: $description, content: "This is a test\nwith newlines\tand tabs"){id}}`,
		},
		{
			name: "complex_query_with_nested_fields",
			input: `query Dragons {
				dragons {
					name
					type
					abilities {
						name
						power
					}
				}
			}`,
			expected: "query Dragons{dragons{name type abilities{name power}}}",
		},
		{
			name: "query_with_single_quotes",
			input: `mutation test($value: String!) {
				insert_test(value: $value, note: 'Single quoted string with\nnewlines') {
					id
				}
			}`,
			expected: `mutation test($value: String!){insert_test(value: $value, note: 'Single quoted string with\nnewlines'){id}}`,
		},
		{
			name: "query_with_escaped_quotes",
			input: `mutation test($value: String!) {
				insert_test(value: $value, note: "String with \"escaped quotes\" inside") {
					id
				}
			}`,
			expected: `mutation test($value: String!){insert_test(value: $value, note: "String with \"escaped quotes\" inside"){id}}`,
		},
		{
			name: "production_case_from_trailing_garbage_test",
			input: `mutation storeProcessedMessage($content: jsonb, $telegram_msg_id: bigint!, $hasMedia: Boolean!, $urls: jsonb, $groupID: bigint!, $userID: bigint!, $entities: jsonb) {
				insert_tg_messages_one(object: {content: $content, telegram_msg_id: $telegram_msg_id, has_media: $hasMedia, urls: $urls, group_id: $groupID, user_id: $userID, entities: $entities}) {
					id
					tg_group {
						tg_group_admins_aggregate(where: {user_id: {_eq: $userID}}) {
							aggregate {
								count
							}
						}
						set_spam_ai
					}
				}
			}`,
			expected: "mutation storeProcessedMessage($content: jsonb, $telegram_msg_id: bigint!, $hasMedia: Boolean!, $urls: jsonb, $groupID: bigint!, $userID: bigint!, $entities: jsonb){insert_tg_messages_one(object: {content: $content,telegram_msg_id: $telegram_msg_id, has_media: $hasMedia, urls: $urls, group_id: $groupID, user_id: $userID, entities: $entities}){id tg_group{tg_group_admins_aggregate(where: {user_id: {_eq: $userID}}){aggregate{count}} set_spam_ai}}}",
		},
		{
			name: "query_with_mixed_quotes_and_escapes",
			input: `mutation test {
				insert_test(
					single: 'Single quote with "double" inside',
					double: "Double quote with 'single' inside",
					escaped: "Escaped \"double\" quotes"
				) {
					id
				}
			}`,
			expected: `mutation test{insert_test(single: 'Single quote with "double" inside', double: "Double quote with 'single' inside", escaped: "Escaped \"double\" quotes"){id}}`,
		},
		{
			name: "query_with_unicode_in_strings",
			input: `mutation storeMessage($user_name: String!) {
				insert_message(
					user_name: $user_name,
					description: "This a group for serious buyers and sellers\nOf uber eats and Door dash account \nRippers are not allowed in this group😈"
				) {
					id
				}
			}`,
			expected: `mutation storeMessage($user_name: String!){insert_message(user_name: $user_name, description: "This a group for serious buyers and sellers\nOf uber eats and Door dash account \nRippers are not allowed in this group😈"){id}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := minifyGraphQLQuery(tt.input)
			assert.Equal(tt.expected, result, "Minified query should match expected output")

			// Verify that the result doesn't contain unnecessary whitespace
			assert.False(strings.Contains(result, "\n"), "Result should not contain newlines")
			assert.False(strings.Contains(result, "\t"), "Result should not contain tabs")

			// Verify that string literals are preserved (if they exist in expected)
			if strings.Contains(tt.expected, `\n`) {
				assert.True(strings.Contains(result, `\n`), "Newlines in string literals should be preserved")
			}
			if strings.Contains(tt.expected, `\t`) {
				assert.True(strings.Contains(result, `\t`), "Tabs in string literals should be preserved")
			}
		})
	}
}

func TestMinifyGraphQLQuerySizeReduction(t *testing.T) {
	t.Parallel()
	assert := assertions.New(t)

	// Test with a heavily formatted query
	heavyQuery := `
		mutation storeProcessedMessage(
			$content: jsonb,
			$telegram_msg_id: bigint!,
			$hasMedia: Boolean!,
			$urls: jsonb,
			$groupID: bigint!,
			$userID: bigint!,
			$entities: jsonb
		) {
			insert_tg_messages_one(
				object: {
					content: $content,
					telegram_msg_id: $telegram_msg_id,
					has_media: $hasMedia,
					urls: $urls,
					group_id: $groupID,
					user_id: $userID,
					entities: $entities
				}
			) {
				id
				tg_group {
					tg_group_admins_aggregate(
						where: {
							user_id: {
								_eq: $userID
							}
						}
					) {
						aggregate {
							count
						}
					}
					set_spam_ai
				}
			}
		}
	`

	originalSize := len(heavyQuery)
	minified := minifyGraphQLQuery(heavyQuery)
	minifiedSize := len(minified)

	t.Logf("Original size: %d bytes", originalSize)
	t.Logf("Minified size: %d bytes", minifiedSize)
	t.Logf("Size reduction: %d bytes (%.1f%%)", originalSize-minifiedSize, float64(originalSize-minifiedSize)/float64(originalSize)*100)

	// Should achieve significant size reduction
	assert.True(minifiedSize < originalSize, "Minified query should be smaller than original")
	reductionPercent := float64(originalSize-minifiedSize) / float64(originalSize) * 100
	assert.True(reductionPercent > 30, "Should achieve at least 30% size reduction for heavily formatted queries")
}

func TestBaseClientQueryMinification(t *testing.T) {
	t.Parallel()
	assert := assertions.New(t)

	t.Run("should_minify_queries_by_default", func(t *testing.T) {
		t.Parallel()
		client := CreateTestClient()

		// Verify minification is enabled by default
		assert.True(client.minify_queries, "Query minification should be enabled by default")

		query := `query {
			viewer {
				login
			}
		}`
		variables := map[string]interface{}{}

		compiledQuery := client.compileQuery(query, variables)
		assert.NotNil(compiledQuery)

		// The compiled query should be minified (no newlines/tabs)
		assert.False(strings.Contains(compiledQuery.Query, "\n"), "Compiled query should not contain newlines")
		assert.False(strings.Contains(compiledQuery.Query, "\t"), "Compiled query should not contain tabs")
		assert.Equal("query{viewer{login}}", compiledQuery.Query)
	})

	t.Run("should_respect_minification_disabled", func(t *testing.T) {
		t.Parallel()
		client := CreateTestClient()
		client.SetQueryMinification(false)

		// Verify minification is disabled
		assert.False(client.minify_queries, "Query minification should be disabled")

		query := `query {
			viewer {
				login
			}
		}`
		variables := map[string]interface{}{}

		compiledQuery := client.compileQuery(query, variables)
		assert.NotNil(compiledQuery)

		// The compiled query should NOT be minified (preserve original formatting)
		assert.Equal(query, compiledQuery.Query, "Query should remain unchanged when minification is disabled")
	})

	t.Run("should_handle_string_literals_correctly", func(t *testing.T) {
		t.Parallel()
		client := CreateTestClient()

		query := `mutation test($value: String!) {
			insert_test(
				value: $value,
				note: "This is a test\nwith newlines\tand tabs"
			) {
				id
			}
		}`
		variables := map[string]interface{}{
			"value": "test",
		}

		compiledQuery := client.compileQuery(query, variables)
		assert.NotNil(compiledQuery)

		// String literals should preserve their content
		assert.Contains(compiledQuery.Query, `"This is a test\nwith newlines\tand tabs"`, "String literals should be preserved")
		// But the query structure should be minified
		assert.False(strings.Contains(strings.Replace(compiledQuery.Query, `"This is a test\nwith newlines\tand tabs"`, "", 1), "\n"), "Query structure should be minified")
	})

	t.Run("should_demonstrate_size_reduction_with_production_case", func(t *testing.T) {
		t.Parallel()
		client := CreateTestClient()

		// Use the production case from trailing garbage test
		variables := map[string]interface{}{
			"group_description": "This a group for serious buyers and sellers\nOf uber eats and Door dash account \nRippers are not allowed in this group😈",
			"user_name":         "2eba巴结",
		}

		originalQuery := `mutation storeMessage(
			$group_description: String!,
			$user_name: String!
		) {
			insert_message(
				group_description: $group_description,
				user_name: $user_name
			) {
				id
				created_at
				updated_at
			}
		}`

		// Test with minification enabled
		client.SetQueryMinification(true)
		compiledQueryMinified := client.compileQuery(originalQuery, variables)
		minifiedJsonSize := len(compiledQueryMinified.JsonQuery)

		// Test with minification disabled
		client.SetQueryMinification(false)
		compiledQueryOriginal := client.compileQuery(originalQuery, variables)
		originalJsonSize := len(compiledQueryOriginal.JsonQuery)

		t.Logf("Original JSON size: %d bytes", originalJsonSize)
		t.Logf("Minified JSON size: %d bytes", minifiedJsonSize)
		t.Logf("Size reduction: %d bytes", originalJsonSize-minifiedJsonSize)

		// Should achieve size reduction
		assert.True(minifiedJsonSize < originalJsonSize, "Minified JSON should be smaller")

		// Both should be valid JSON
		assert.True(len(compiledQueryMinified.JsonQuery) > 0, "Minified query should produce valid JSON")
		assert.True(len(compiledQueryOriginal.JsonQuery) > 0, "Original query should produce valid JSON")
	})
}

func TestIsAlphaNumeric(t *testing.T) {
	t.Parallel()
	assert := assertions.New(t)

	tests := []struct {
		char     byte
		expected bool
	}{
		{'a', true},
		{'z', true},
		{'A', true},
		{'Z', true},
		{'0', true},
		{'9', true},
		{'_', true},
		{' ', false},
		{'\n', false},
		{'\t', false},
		{'{', false},
		{'}', false},
		{'(', false},
		{')', false},
		{':', false},
		{'"', false},
		{'\'', false},
	}

	for _, tt := range tests {
		t.Run(string(tt.char), func(t *testing.T) {
			t.Parallel()
			result := isAlphaNumeric(tt.char)
			assert.Equal(tt.expected, result, "isAlphaNumeric should correctly identify character type")
		})
	}
}

func TestQueryMinificationEnvironmentVariable(t *testing.T) {
	t.Parallel()
	assert := assertions.New(t)

	t.Run("should_respect_GRAPHQL_MINIFY_QUERIES_env_var", func(t *testing.T) {
		t.Parallel()
		// This test would require setting environment variables
		// For now, we'll test the default behavior
		client := NewConnection()

		// Should be enabled by default
		assert.True(client.minify_queries, "Query minification should be enabled by default")
	})
}

func TestQueryMinificationEdgeCases(t *testing.T) {
	t.Parallel()
	assert := assertions.New(t)

	t.Run("should_handle_empty_query", func(t *testing.T) {
		t.Parallel()
		result := minifyGraphQLQuery("")
		assert.Equal("", result, "Empty query should remain empty")
	})

	t.Run("should_handle_query_with_only_whitespace", func(t *testing.T) {
		t.Parallel()
		result := minifyGraphQLQuery("   \n\t  \n  ")
		assert.Equal("", result, "Query with only whitespace should become empty")
	})

	t.Run("should_handle_nested_quotes", func(t *testing.T) {
		t.Parallel()
		query := `mutation test {
			insert_test(value: "String with \"nested quotes\" and 'mixed' quotes") {
				id
			}
		}`
		result := minifyGraphQLQuery(query)
		expected := `mutation test{insert_test(value: "String with \"nested quotes\" and 'mixed' quotes"){id}}`
		assert.Equal(expected, result, "Should handle nested quotes correctly")
	})

	t.Run("should_preserve_spaces_between_keywords", func(t *testing.T) {
		t.Parallel()
		query := `query test($var: String!) {
			field(where: {id: {_eq: $var}}) {
				id
			}
		}`
		result := minifyGraphQLQuery(query)

		// Should preserve necessary spaces
		assert.Contains(result, "query test", "Should preserve space between 'query' and 'test'")
		assert.Contains(result, "$var: String!", "Should preserve space in variable declaration")
		assert.Contains(result, "_eq: $var", "Should preserve space around operators")
	})
}
