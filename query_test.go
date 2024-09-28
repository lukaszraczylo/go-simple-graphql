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
