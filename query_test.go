package gql

import (
	"testing"

	"github.com/goccy/go-reflect"
)

func (suite *Tests) TestBaseClient_convertToJSON() {
	type args struct {
		v any
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "TestBaseClient_convertToJSON_query",
			args: args{
				v: `query { viewer { login } }`,
			},
			want: []byte(`"query { viewer { login } }"`),
		},
		{
			name: "TestBaseClient_convertToJSON_variables",
			args: args{
				v: map[string]interface{}{
					"login": "lukaszraczylo",
					"first": 10,
				},
			},
			want: []byte(`{"first":10,"login":"lukaszraczylo"}`),
		},
	}
	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			b := NewConnection()
			if got := b.convertToJSON(tt.args.v); !reflect.DeepEqual(got, tt.want) {
				assert.Equal(tt.want, got)
			}
		})
	}
}

func (suite *Tests) TestBaseClient_compileQuery() {
	type args struct {
		query_partials []any
	}
	tests := []struct {
		wantQ *Query
		name  string
		args  args
	}{
		{
			name: "TestBaseClient_compileQuery_query",
			args: args{
				query_partials: []any{
					`query { viewer { login } }`,
				},
			},
			wantQ: &Query{
				Query:     `query{viewer{login}}`,
				JsonQuery: []byte(`{"query":"query{viewer{login}}"}`),
			},
		},
		{
			name: "TestBaseClient_compileQuery_with_variables",
			args: args{
				query_partials: []any{
					`query { viewer { login } }`,
					map[string]interface{}{
						"login": "lukaszraczylo",
						"first": 10,
					},
				},
			},
			wantQ: &Query{
				Query: `query{viewer{login}}`,
				Variables: map[string]interface{}{
					"login": "lukaszraczylo",
					"first": 10,
				},
				JsonQuery: []byte(`{"variables":{"first":10,"login":"lukaszraczylo"},"query":"query{viewer{login}}"}`),
			},
		},
		{
			name: "TestBaseClient_compileQuery_variables_without_query",
			args: args{
				query_partials: []any{
					map[string]interface{}{
						"login": "lukaszraczylo",
						"first": 10,
					},
				},
			},
			wantQ: nil,
		},
	}
	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			b := NewConnection()
			got := b.compileQuery(tt.args.query_partials...)
			assert.Equal(tt.wantQ, got)
		})
	}
}

func (suite *Tests) TestBaseClient_Query() {
	type args struct {
		variables map[string]interface{}
		headers   map[string]interface{}
		query     string
	}
	tests := []struct {
		args               args
		wantReturned_value any
		name               string
		wantType           string
		wantErr            bool
	}{
		{
			name: "TestBaseClient_Query_viewer",
			args: args{
				query:     "query { viewer { login } }",
				variables: nil,
				headers: map[string]interface{}{
					"x-apollo-operation-name": "ViewerQuery",
					"content-type":            "application/json",
				},
			},
			wantReturned_value: `{"viewer":{"login":"mockuser"}}`, // Adjusted expected value
			wantErr:            false,
		},
		{
			name: "TestBaseClient_Query_dragons",
			args: args{
				query: `query Dragons {
													dragons {
																	name
													}
									}`,
				variables: nil,
				headers: map[string]interface{}{
					"x-apollo-operation-name": "DragonsQuery",
					"content-type":            "application/json",
				},
			},
			wantReturned_value: `{"dragons":[{"name":"Mock Dragon 1"},{"name":"Mock Dragon 2"}]}`, // Adjusted expected value
			wantErr:            false,
		},

		{
			name: "TestBaseClient_Query_invalid_query",
			args: args{
				query:     "query { potato { login } ",
				variables: nil,
				headers: map[string]interface{}{
					"x-apollo-operation-name": "InvalidQuery",
					"content-type":            "application/json",
				},
			},
			wantReturned_value: nil,
			wantErr:            true,
		},
	}
	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			b := NewConnection()
			b.SetEndpoint(mockServer.URL)
			b.SetHTTPClient(mockServer.Client())
			if tt.wantType != "" {
				b.SetOutput(tt.wantType)
			}
			gotReturned_value, err := b.Query(tt.args.query, tt.args.variables, tt.args.headers)
			if tt.wantErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
			assert.Equal(tt.wantReturned_value, gotReturned_value)
		})
	}
}

func (suite *Tests) TestBaseClient_convertToJSON_errors() {
	suite.T().Run("should handle JSON encoding errors", func(t *testing.T) {
		b := NewConnection()

		// Test with a value that can't be JSON encoded (function)
		result := b.convertToJSON(func() {})
		assert.Nil(result)
	})

	suite.T().Run("should handle Query type with variables", func(t *testing.T) {
		b := NewConnection()

		query := &Query{
			Query: "query { test }",
			Variables: map[string]interface{}{
				"var1": "value1",
				"var2": 123,
			},
		}

		result := b.convertToJSON(query)
		assert.NotNil(result)
		assert.Contains(string(result), "query { test }")
		assert.Contains(string(result), "var1")
		assert.Contains(string(result), "value1")
	})
}

func (suite *Tests) TestBaseClient_compileQuery_errors() {
	suite.T().Run("should handle empty query", func(t *testing.T) {
		b := NewConnection()

		result := b.compileQuery("")
		assert.Nil(result)
	})

	suite.T().Run("should handle no arguments", func(t *testing.T) {
		b := NewConnection()

		result := b.compileQuery()
		assert.Nil(result)
	})

	suite.T().Run("should handle invalid variable type", func(t *testing.T) {
		b := NewConnection()

		result := b.compileQuery("query { test }", "invalid_variables")
		assert.NotNil(result)
		assert.Equal("query{test}", result.Query)
		assert.Nil(result.Variables)
	})
}

func (suite *Tests) Test_processFlags() {
	tests := []struct {
		name            string
		variables       map[string]interface{}
		headers         map[string]interface{}
		wantCache       bool
		wantRetries     bool
		wantCleanedVars map[string]interface{}
	}{
		{
			name:      "cache enabled in headers",
			variables: nil,
			headers: map[string]interface{}{
				"gqlcache": true,
			},
			wantCache:       true,
			wantRetries:     false,
			wantCleanedVars: nil,
		},
		{
			name:      "retries enabled in headers",
			variables: nil,
			headers: map[string]interface{}{
				"gqlretries": true,
			},
			wantCache:       false,
			wantRetries:     true,
			wantCleanedVars: nil,
		},
		{
			name: "cache enabled in variables",
			variables: map[string]interface{}{
				"gqlcache": true,
				"test":     "value",
			},
			headers:     nil,
			wantCache:   true,
			wantRetries: false,
			wantCleanedVars: map[string]interface{}{
				"test": "value",
			},
		},
		{
			name: "retries enabled in variables",
			variables: map[string]interface{}{
				"gqlretries": true,
				"test":       "value",
			},
			headers:     nil,
			wantCache:   false,
			wantRetries: true,
			wantCleanedVars: map[string]interface{}{
				"test": "value",
			},
		},
		{
			name: "both flags in variables",
			variables: map[string]interface{}{
				"gqlcache":   true,
				"gqlretries": false,
				"test":       "value",
			},
			headers:     nil,
			wantCache:   true,
			wantRetries: false,
			wantCleanedVars: map[string]interface{}{
				"test": "value",
			},
		},
		{
			name: "flags in both headers and variables",
			variables: map[string]interface{}{
				"gqlcache": false,
				"test":     "value",
			},
			headers: map[string]interface{}{
				"gqlretries": true,
			},
			wantCache:   false,
			wantRetries: true,
			wantCleanedVars: map[string]interface{}{
				"test": "value",
			},
		},
		{
			name:            "no flags",
			variables:       map[string]interface{}{"test": "value"},
			headers:         nil,
			wantCache:       false,
			wantRetries:     false,
			wantCleanedVars: map[string]interface{}{"test": "value"},
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			gotCache, gotRetries, gotCleanedVars := processFlags(tt.variables, tt.headers)
			assert.Equal(tt.wantCache, gotCache)
			assert.Equal(tt.wantRetries, gotRetries)
			assert.Equal(tt.wantCleanedVars, gotCleanedVars)
		})
	}
}

func (suite *Tests) TestBaseClient_Query_caching() {
	suite.T().Run("should use cache for query operations", func(t *testing.T) {
		b := NewConnection()
		b.SetEndpoint(mockServer.URL)
		b.SetHTTPClient(mockServer.Client())

		query := "query { viewer { login } }"
		variables := map[string]interface{}{
			"gqlcache": true,
		}

		// First call should hit the server
		result1, err := b.Query(query, variables, nil)
		assert.NoError(err)
		assert.NotNil(result1)

		// Second call should use cache
		result2, err := b.Query(query, variables, nil)
		assert.NoError(err)
		assert.Equal(result1, result2)
	})

	suite.T().Run("should not cache mutations", func(t *testing.T) {
		b := NewConnection()
		b.SetEndpoint(mockServer.URL)
		b.SetHTTPClient(mockServer.Client())

		mutation := "mutation { updateUser { id } }"
		variables := map[string]interface{}{
			"gqlcache": true,
		}

		// Mutations should not be cached even with cache flag
		_, err := b.Query(mutation, variables, nil)
		assert.Error(err) // Mock server doesn't handle mutations
	})
}

func (suite *Tests) TestBaseClient_Query_retries() {
	suite.T().Run("should handle retries flag", func(t *testing.T) {
		b := NewConnection()
		b.SetEndpoint(mockServer.URL)
		b.SetHTTPClient(mockServer.Client())

		query := "query { viewer { login } }"
		variables := map[string]interface{}{
			"gqlretries": true,
		}

		result, err := b.Query(query, variables, nil)
		assert.NoError(err)
		assert.NotNil(result)
	})
}

func (suite *Tests) TestBaseClient_Query_globalCache() {
	suite.T().Run("should respect global cache setting", func(t *testing.T) {
		b := NewConnection()
		b.SetEndpoint(mockServer.URL)
		b.SetHTTPClient(mockServer.Client())
		b.cache_global = true

		query := "query { viewer { login } }"

		// Should use cache even without explicit flag
		result1, err := b.Query(query, nil, nil)
		assert.NoError(err)
		assert.NotNil(result1)

		result2, err := b.Query(query, nil, nil)
		assert.NoError(err)
		assert.Equal(result1, result2)
	})
}

func (suite *Tests) TestBaseClient_Query_emptyQuery() {
	suite.T().Run("should handle empty query", func(t *testing.T) {
		b := NewConnection()
		b.SetEndpoint(mockServer.URL)
		b.SetHTTPClient(mockServer.Client())

		result, err := b.Query("", nil, nil)
		assert.Error(err)
		assert.Nil(result)
		assert.Contains(err.Error(), "can't compile query")
	})
}
