package gql

import (
	"fmt"
	"testing"
)

func Benchmark_calculateHash(b *testing.B) {
	client := NewConnection()
	query := &Query{
		Query:     "query { viewer { login } }",
		Variables: map[string]interface{}{"var1": "value1"},
	}
	query.JsonQuery = client.convertToJSON(query)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = calculateHash(query)
	}
}

func Benchmark_calculateHashLargeQuery(b *testing.B) {
	client := NewConnection()
	// Create a larger query to test hash performance with bigger payloads
	largeQuery := `query {
		viewer {
			login
			name
			email
			bio
			company
			location
			websiteUrl
			twitterUsername
			repositories(first: 100) {
				nodes {
					name
					description
					url
					stargazerCount
					forkCount
					primaryLanguage {
						name
						color
					}
					createdAt
					updatedAt
				}
			}
		}
	}`

	variables := make(map[string]interface{})
	for i := 0; i < 50; i++ {
		variables[fmt.Sprintf("var%d", i)] = fmt.Sprintf("value%d", i)
	}

	query := &Query{
		Query:     largeQuery,
		Variables: variables,
	}
	query.JsonQuery = client.convertToJSON(query)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = calculateHash(query)
	}
}
