package gql

import (
	"reflect"
	"testing"
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
				Query:     `query { viewer { login } }`,
				JsonQuery: []byte(`{"query":"query { viewer { login } }"}`),
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
				Query: `query { viewer { login } }`,
				Variables: map[string]interface{}{
					"login": "lukaszraczylo",
					"first": 10,
				},
				JsonQuery: []byte(`{"variables":{"first":10,"login":"lukaszraczylo"},"query":"query { viewer { login } }"}`),
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
		graphQLServer      string
		wantType           string
		wantErr            bool
	}{
		{
			name:          "TestBaseClient_Query_spacex_working",
			graphQLServer: "https://spacex-production.up.railway.app/",
			args: args{
				query: `query Dragons {
					dragons {
						name
					}
				}`,
				variables: nil,
				headers: map[string]interface{}{
					"x-apollo-operation-name": "Dragons/github.com/lukaszraczylo/go-simple-graphql",
					"content-type":            "application/json",
				},
			},
			wantReturned_value: string(`{"dragons":[{"name":"Dragon 1"},{"name":"Dragon 2"}]}`),
			wantErr:            false,
		},
		{
			name:          "TestBaseClient_Query_spacex_invalid_query",
			graphQLServer: "https://spacex-production.up.railway.app/",
			args: args{
				query: `query Dragons {
					dragons {
						name
					}
				`,
				variables: nil,
				headers: map[string]interface{}{
					"x-apollo-operation-name": "Dragons/github.com/lukaszraczylo/go-simple-graphql",
					"content-type":            "application/json",
				},
			},
			wantReturned_value: nil,
			wantErr:            true,
		},
		{
			name:          "TestBaseClient_Query_tgbotapp_working",
			graphQLServer: "https://telegram-bot.app/v1/graphql",
			args: args{
				query: `query {
					__type(name: "Query") {
						name
					}
				}`,
			},
			wantReturned_value: string(`{"__type":null}`),
			wantErr:            false,
		},
		{
			name:          "TestBaseClient_Query_tgbotapp_invalid_query",
			graphQLServer: "https://telegram-bot.app/v1/graphql",
			args: args{
				query: `query {
					__type(name: "Query") {
						name
					}
				`,
			},
			wantReturned_value: nil,
			wantErr:            true,
		},
		{
			name:          "TestBaseClient_Query_tgbotapp_invalid_wrong_field",
			graphQLServer: "https://telegram-bot.app/v1/graphql",
			args: args{
				query: `query {
					__type(name: "Query") {
						name2
					}
				}`,
			},
			wantReturned_value: nil,
			wantErr:            true,
		},
		{
			name:          "TestBaseClient_Query_tgbotapp_wrong_url",
			graphQLServer: "https://telegram-bot.app/v0/graphql",
			args: args{
				query: `query {
					__type(name: "Query") {
						name
					}
				}`,
			},
			wantReturned_value: nil,
			wantErr:            true,
		},
		{
			name:          "TestBaseClient_Query_tgbotapp_valid_query_mapstring",
			graphQLServer: "https://telegram-bot.app/v1/graphql",
			wantType:      "mapstring",
			args: args{
				query: `query {
					__type(name: "Query") {
						name
					}
				}`,
			},
			wantReturned_value: map[string]interface{}{"__type": nil},
			wantErr:            false,
		},
		{
			name:          "TestBaseClient_Query_tgbotapp_valid_query_byte",
			graphQLServer: "https://telegram-bot.app/v1/graphql",
			wantType:      "byte",
			args: args{
				query: `query {
					__type(name: "Query") {
						name
					}
				}`,
			},
			wantReturned_value: []byte(`{"__type":null}`),
			wantErr:            false,
		},
		{
			name:          "TestBaseClient_Query_tgbotapp_valid_query_invalid_type",
			graphQLServer: "https://telegram-bot.app/v1/graphql",
			wantType:      "potato",
			args: args{
				query: `query {
					__type(name: "Query") {
						name
					}
				}`,
			},
			wantReturned_value: nil,
			wantErr:            true,
		},
		{
			name:          "TestBaseClient_Query_tgbotapp_valid_query_cache",
			graphQLServer: "https://telegram-bot.app/v1/graphql",
			args: args{
				query: `query {
					__type(name: "Query") {
						name
					}
				}`,
				variables: map[string]interface{}{
					"gqlcache": true,
				},
			},
			wantReturned_value: string(`{"__type":null}`),
			wantErr:            false,
		},
		{
			name:          "TestBaseClient_Query_tgbotapp_valid_query_with_retry",
			graphQLServer: "https://telegram-bot.app/v0/graphql",
			wantType:      "byte",
			args: args{
				query: `query {
					__type(name: "Query") {
						name
					}
				}`,
				variables: map[string]interface{}{
					"gqlretries": true,
				},
			},
			wantReturned_value: nil,
			wantErr:            true,
		},
	}
	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			b := NewConnection()
			if tt.graphQLServer != "" {
				b.SetEndpoint(tt.graphQLServer)
			}
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
