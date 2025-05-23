package gql

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type QueryOptimizationTestSuite struct {
	suite.Suite
	client *BaseClient
}

func (suite *QueryOptimizationTestSuite) SetupTest() {
	suite.client = CreateTestClient()
}

func (suite *QueryOptimizationTestSuite) TearDownTest() {
	// No need to stop shared cache - it will be cleaned up globally
}

func (suite *QueryOptimizationTestSuite) TestProcessFlags() {
	tests := []struct {
		name             string
		variables        map[string]interface{}
		headers          map[string]interface{}
		expectedCache    bool
		expectedRetries  bool
		expectedVarCount int
	}{
		{
			name:             "No flags",
			variables:        map[string]interface{}{"id": "123"},
			headers:          map[string]interface{}{},
			expectedCache:    false,
			expectedRetries:  false,
			expectedVarCount: 1,
		},
		{
			name:             "Cache flag in headers",
			variables:        map[string]interface{}{"id": "123"},
			headers:          map[string]interface{}{"gqlcache": true},
			expectedCache:    true,
			expectedRetries:  false,
			expectedVarCount: 1,
		},
		{
			name:             "Retries flag in headers",
			variables:        map[string]interface{}{"id": "123"},
			headers:          map[string]interface{}{"gqlretries": true},
			expectedCache:    false,
			expectedRetries:  true,
			expectedVarCount: 1,
		},
		{
			name:             "Cache flag in variables",
			variables:        map[string]interface{}{"id": "123", "gqlcache": true},
			headers:          map[string]interface{}{},
			expectedCache:    true,
			expectedRetries:  false,
			expectedVarCount: 1, // gqlcache should be removed
		},
		{
			name:             "Retries flag in variables",
			variables:        map[string]interface{}{"id": "123", "gqlretries": true},
			headers:          map[string]interface{}{},
			expectedCache:    false,
			expectedRetries:  true,
			expectedVarCount: 1, // gqlretries should be removed
		},
		{
			name:             "Both flags in variables",
			variables:        map[string]interface{}{"id": "123", "gqlcache": true, "gqlretries": true},
			headers:          map[string]interface{}{},
			expectedCache:    true,
			expectedRetries:  true,
			expectedVarCount: 1, // both flags should be removed
		},
		{
			name:             "Flags in both headers and variables",
			variables:        map[string]interface{}{"id": "123", "gqlcache": false},
			headers:          map[string]interface{}{"gqlcache": true, "gqlretries": true},
			expectedCache:    true, // header value OR variable value
			expectedRetries:  true,
			expectedVarCount: 1, // gqlcache should be removed from variables
		},
		{
			name:             "Nil variables",
			variables:        nil,
			headers:          map[string]interface{}{"gqlcache": true},
			expectedCache:    true,
			expectedRetries:  false,
			expectedVarCount: 0,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			enableCache, enableRetries, cleanedVariables := processFlags(tt.variables, tt.headers)

			suite.Equal(tt.expectedCache, enableCache, "Cache flag mismatch")
			suite.Equal(tt.expectedRetries, enableRetries, "Retries flag mismatch")

			if tt.expectedVarCount == 0 {
				suite.Nil(cleanedVariables, "Variables should be nil")
			} else {
				suite.Len(cleanedVariables, tt.expectedVarCount, "Variable count mismatch")
				suite.NotContains(cleanedVariables, "gqlcache", "gqlcache should be removed")
				suite.NotContains(cleanedVariables, "gqlretries", "gqlretries should be removed")
			}
		})
	}
}

func (suite *QueryOptimizationTestSuite) TestDoubleCompilationFix() {
	// Test that query is only compiled once even with flags in variables
	query := "query GetUser($id: ID!) { user(id: $id) { id name } }"
	variables := map[string]interface{}{
		"id":         "123",
		"gqlcache":   true,
		"gqlretries": true,
	}

	// This should not cause double compilation
	compiledQuery := suite.client.compileQuery(query, variables)
	suite.NotNil(compiledQuery)
	suite.Equal("query GetUser($id: ID!){user(id: $id){id name}}", compiledQuery.Query)

	// Variables should still contain the flags at this point (they're cleaned in Query method)
	suite.Contains(compiledQuery.Variables, "gqlcache")
	suite.Contains(compiledQuery.Variables, "gqlretries")
}

func (suite *QueryOptimizationTestSuite) TestQueryMethodOptimization() {
	// Test that processFlags works correctly without making HTTP requests
	variables := map[string]interface{}{
		"id":         "123",
		"gqlcache":   true,
		"gqlretries": true,
	}
	headers := map[string]interface{}{}

	// Test processFlags directly
	enableCache, enableRetries, cleanedVariables := processFlags(variables, headers)

	suite.True(enableCache, "Cache should be enabled")
	suite.True(enableRetries, "Retries should be enabled")
	suite.Len(cleanedVariables, 1, "Should have only id variable")
	suite.Equal("123", cleanedVariables["id"], "ID should be preserved")
	suite.NotContains(cleanedVariables, "gqlcache", "gqlcache should be removed")
	suite.NotContains(cleanedVariables, "gqlretries", "gqlretries should be removed")
}

// Benchmark the optimization impact
func BenchmarkQueryCompilation(b *testing.B) {
	client := CreateTestClient()

	query := "query GetUser($id: ID!) { user(id: $id) { id name email profile { avatar bio } } }"

	b.Run("WithoutFlags", func(b *testing.B) {
		variables := map[string]interface{}{"id": "123"}
		headers := map[string]interface{}{}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			processFlags(variables, headers)
		}
	})

	b.Run("WithFlags", func(b *testing.B) {
		variables := map[string]interface{}{
			"id":         "123",
			"gqlcache":   true,
			"gqlretries": true,
		}
		headers := map[string]interface{}{}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			processFlags(variables, headers)
		}
	})

	b.Run("CompileQuery", func(b *testing.B) {
		variables := map[string]interface{}{"id": "123"}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			client.compileQuery(query, variables)
		}
	})
}

func TestQueryOptimizationSuite(t *testing.T) {
	suite.Run(t, new(QueryOptimizationTestSuite))
}
