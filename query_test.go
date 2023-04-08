package gql

import (
	"math/rand"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBaseClient_convertToJson(t *testing.T) {
	type fields struct {
	}
	type args struct {
		v any
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []byte
	}{
		{
			name:   "Test convertToJson",
			fields: fields{},
			args: args{
				v: map[string]interface{}{
					"query": "query { hello }",
				},
			},
			want: []byte(`{"query":"query { hello }"}`),
		},
		{
			name:   "Test convertToJson with variables",
			fields: fields{},
			args: args{
				v: map[string]interface{}{
					"query":     "query { hello }",
					"variables": map[string]interface{}{"name": "John"},
				},
			},
			want: []byte(`{"query":"query { hello }","variables":{"name":"John"}}`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &BaseClient{}
			if got := c.convertToJson(tt.args.v); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BaseClient.convertToJson() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBaseClient_NewQuery(t *testing.T) {
	type fields struct {
	}
	type args struct {
		q []any
	}
	tests := []struct {
		fields fields
		want   *Query
		name   string
		args   args
	}{
		{
			name:   "Test NewQuery",
			fields: fields{},
			args: args{
				q: []any{
					"query { hello }",
				},
			},
			want: &Query{
				compiledQuery: []byte(`{"variables":null,"query":"query { hello }"}`),
			},
		},
		{
			name:   "Test NewQuery with variables",
			fields: fields{},
			args: args{
				q: []any{
					"query { hello }",
					map[string]interface{}{"name": "John"},
				},
			},
			want: &Query{
				compiledQuery: []byte(`{"variables":{"name":"John"},"query":"query { hello }"}`),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewConnection()
			gotQuery := c.NewQuery(tt.args.q...)
			assert.Equal(t, tt.want.query, gotQuery.query)
			assert.Equal(t, tt.want.variables, gotQuery.variables)
			assert.Equal(t, tt.want.compiledQuery, gotQuery.compiledQuery)
		})
	}
}

func TestBaseClient_decodeResponse(t *testing.T) {
	type fields struct {
		responseType string
	}
	type args struct {
		jsonData []byte
	}
	tests := []struct {
		fields fields
		want   any
		name   string
		args   args
	}{
		{
			name: "Test decodeResponse - string",
			fields: fields{
				responseType: "string",
			},
			args: args{
				jsonData: []byte(`{"data":{"hello":"world"}}`),
			},
			want: `{"data":{"hello":"world"}}`,
		},
		{
			name: "Test decodeResponse - byte",
			fields: fields{
				responseType: "byte",
			},
			args: args{
				jsonData: []byte(`{"data":{"hello":"world"}}`),
			},
			want: []byte(`{"data":{"hello":"world"}}`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewConnection()
			c.responseType = tt.fields.responseType
			if got := c.decodeResponse(tt.args.jsonData); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BaseClient.decodeResponse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBaseClient_Query(t *testing.T) {
	type fields struct {
		graphql_endpoint string
		repeat           int
	}
	type args struct {
		queryVariables interface{}
		queryHeaders   map[string]interface{}
		queryContent   string
	}
	tests := []struct {
		want    any
		args    args
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Test Query - failing",
			fields: fields{
				graphql_endpoint: "https://spacex-production.up.railway.app/",
			},
			args: args{
				queryContent: "query { hello }",
			},
			want:    ``,
			wantErr: true,
		},
		{
			name: "Test Query - success",
			fields: fields{
				graphql_endpoint: "https://spacex-production.up.railway.app/",
			},
			args: args{
				queryContent: `query Dragons {
					dragons {
						name
					}
				}`,
				queryHeaders: map[string]interface{}{
					"x-apollo-operation-name": "Missions-Potato-Test-Golang",
					"content-type":            "application/json",
				},
			},
			wantErr: false,
			want:    `{"dragons":[{"name":"Dragon 1"},{"name":"Dragon 2"}]}`,
		},
		{
			name: "Test Query - success",
			fields: fields{
				repeat:           2,
				graphql_endpoint: "https://spacex-production.up.railway.app/",
			},
			args: args{
				queryContent: `query Dragons {
					dragons {
						name
						first_flight
					}
				}`,
				queryHeaders: map[string]interface{}{
					"x-apollo-operation-name": "Missions-Potato-Test-Golang-Cached",
					"content-type":            "application/json",
				},
			},
			wantErr: false,
			want:    `{"dragons":[{"first_flight":"2010-12-08","name":"Dragon 1"},{"first_flight":"2019-03-02","name":"Dragon 2"}]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewConnection()

			if tt.fields.graphql_endpoint != "" {
				c.endpoint = tt.fields.graphql_endpoint
			}

			repeat := 1
			if tt.fields.repeat != 0 {
				repeat = tt.fields.repeat
			}

			for i := 0; i < repeat; i++ {

				got, err := c.Query(tt.args.queryContent, tt.args.queryVariables, tt.args.queryHeaders)

				if err != nil && tt.wantErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, tt.want, got)
				}
			}
		})
	}
}

/// benchmarks

func BenchmarkBaseClient_convertToJson(b *testing.B) {
	c := NewConnection()
	query := c.NewQuery("query { hello }")
	for i := 0; i < b.N; i++ {
		_ = c.convertToJson(query)
	}
}

func BenchmarkNewQuery(b *testing.B) {
	c := NewConnection()
	for i := 0; i < b.N; i++ {
		_ = c.NewQuery("query { hello }")
	}
}

func BenchmarkDecodeResponse(b *testing.B) {
	c := NewConnection()
	for i := 0; i < b.N; i++ {
		_ = c.decodeResponse([]byte(`{"data":{"hello":"world"}}`))
	}
}

func BenchmarkQueryNoCache(b *testing.B) {
	c := NewConnection()
	c.cache.enabled = false
	for i := 0; i < b.N; i++ {
		_, _ = c.Query(`query MyQuery {
			bots {
				bot_name
			}
		}`, nil, nil)
	}
}

func BenchmarkQueryWithCache(b *testing.B) {
	c := NewConnection()
	c.cache.enabled = true
	for i := 0; i < b.N; i++ {
		_, _ = c.Query(`query MyQuery {
			bots {
				bot_name
			}
		}`, nil, nil)
	}
}

func TestBaseClient_parseQueryHeaders(t *testing.T) {
	type fields struct {
	}
	type args struct {
		queryHeaders map[string]interface{}
	}
	tests := []struct {
		name              string
		fields            fields
		args              args
		wantReturnHeaders map[string]interface{}
	}{
		{
			name:   "Test parseQueryHeaders - no change",
			fields: fields{},
			args: args{
				queryHeaders: map[string]interface{}{
					"x-test-header":      "test",
					"x-test-header-next": true,
				},
			},
			wantReturnHeaders: map[string]interface{}{
				"x-test-header":      "test",
				"x-test-header-next": true,
			},
		},
		{
			name:   "Test parseQueryHeaders - lib-headers removed",
			fields: fields{},
			args: args{
				queryHeaders: map[string]interface{}{
					"x-test-header": "test",
					"gqlcache":      true,
				},
			},
			wantReturnHeaders: map[string]interface{}{
				"x-test-header": "test",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewConnection()
			if gotReturnHeaders := c.parseQueryHeaders(tt.args.queryHeaders); !reflect.DeepEqual(gotReturnHeaders, tt.wantReturnHeaders) {
				t.Errorf("BaseClient.parseQueryHeaders() = %v, want %v", gotReturnHeaders, tt.wantReturnHeaders)
			}
		})
	}
}

func TestBaseClient_QueryCache(t *testing.T) {
	type fields struct {
		graphql_endpoint string
		cache_enabled    bool
	}
	type args struct {
		queryVariables interface{}
		queryHeaders   map[string]interface{}
		queryContent   string
	}
	tests := []struct {
		want    any
		args    args
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Test QueryCache - enabled",
			fields: fields{
				graphql_endpoint: "https://spacex-production.up.railway.app/",
				cache_enabled:    true,
			},
			args: args{
				queryContent: `query Dragons {
					dragons {
						name
					}
				}`,
				queryHeaders: map[string]interface{}{
					"x-apollo-operation-name": "Missions-Potato-Test-Golang",
					"content-type":            "application/json",
				},
			},
		},
		{
			name: "Test QueryCache - disabled",
			fields: fields{
				graphql_endpoint: "https://spacex-production.up.railway.app/",
				cache_enabled:    false,
			},
			args: args{
				queryContent: `query Dragons {
					dragons {
						name
					}
				}`,
				queryHeaders: map[string]interface{}{
					"x-apollo-operation-name": "Missions-Potato-Test-Golang",
					"content-type":            "application/json",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewConnection()
			c.cache.enabled = tt.fields.cache_enabled

			if tt.fields.graphql_endpoint != "" {
				c.endpoint = tt.fields.graphql_endpoint
			}

			_, err := c.Query(tt.args.queryContent, tt.args.queryVariables, tt.args.queryHeaders)
			if !tt.wantErr {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestBaseClient_QueryCacheRandomizedRace(t *testing.T) {
	type fields struct {
		graphql_endpoint string
		cache_enabled    bool
	}
	type args struct {
		queryVariables interface{}
		queryHeaders   map[string]interface{}
		queryContent   string
	}
	tests := []struct {
		want    any
		args    args
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Test QueryCache - enabled",
			fields: fields{
				graphql_endpoint: "https://spacex-production.up.railway.app/",
			},
			args: args{
				queryContent: `query Dragons {
					dragons {
						name
					}
				}`,
				queryHeaders: map[string]interface{}{
					"x-apollo-operation-name": "Missions-Potato-Test-Golang",
					"content-type":            "application/json",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewConnection()
			if tt.fields.graphql_endpoint != "" {
				c.endpoint = tt.fields.graphql_endpoint
			}

			for i := 0; i < 10; i++ {
				c.cache.enabled = rand.Intn(2) == 0
				go func() {
					_, err := c.Query(tt.args.queryContent, tt.args.queryVariables, tt.args.queryHeaders)
					if !tt.wantErr {
						assert.NoError(t, err)
					} else {
						assert.Error(t, err)
					}
				}()
			}
		})
	}
}

func TestBaseClient_QueryCacheRandomizedRaceViaHeader(t *testing.T) {
	type fields struct {
		graphql_endpoint string
		cache_enabled    bool
	}
	type args struct {
		queryVariables interface{}
		queryHeaders   map[string]interface{}
		queryContent   string
	}
	tests := []struct {
		want    any
		args    args
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Test QueryCache - enabled",
			fields: fields{
				graphql_endpoint: "https://spacex-production.up.railway.app/",
			},
			args: args{
				queryContent: `query Dragons {
					dragons {
						name
					}
				}`,
				queryHeaders: map[string]interface{}{
					"x-apollo-operation-name": "Missions-Potato-Test-Golang",
					"content-type":            "application/json",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewConnection()
			if tt.fields.graphql_endpoint != "" {
				c.endpoint = tt.fields.graphql_endpoint
			}

			for i := 0; i < 10; i++ {
				tt.args.queryHeaders["gqlcache"] = rand.Intn(2) == 0
				go func() {
					_, err := c.Query(tt.args.queryContent, tt.args.queryVariables, tt.args.queryHeaders)
					if !tt.wantErr {
						assert.NoError(t, err)
					} else {
						assert.Error(t, err)
					}
				}()
			}
		})
	}
}
