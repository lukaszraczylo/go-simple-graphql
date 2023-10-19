package gql

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConnection(t *testing.T) {
	tests := []struct {
		want     *BaseClient
		cache    bool
		endpoint string
		output   string
		name     string
	}{
		{
			name:     "Test NewConnection https",
			endpoint: "https://api.github.com/graphql",
			output:   "string",
			cache:    true,
			want: &BaseClient{
				endpoint:     "https://api.github.com/graphql",
				responseType: "string",
			},
		},
		{
			name:     "Test NewConnection http",
			endpoint: "http://api.github.com/graphql",
			output:   "mapstring",
			cache:    false,
			want: &BaseClient{
				endpoint:     "http://api.github.com/graphql",
				responseType: "mapstring",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// temporarily set GRAPHQL_ENDPOINT env variable to https://api.github.com/graphql
			// and GRAPHQL_OUTPUT to string
			current := os.Getenv("GRAPHQL_ENDPOINT")
			defer os.Setenv("GRAPHQL_ENDPOINT", current)
			os.Setenv("GRAPHQL_ENDPOINT", tt.endpoint)
			got := NewConnection()
			got.SetEndpoint(tt.endpoint)
			got.SetOutput(tt.output)
			if tt.cache {
				got.enableCache()
			} else {
				got.disableCache()
			}

			assert.Equal(t, tt.want.endpoint, got.endpoint)
			assert.Equal(t, tt.want.responseType, got.responseType)
			assert.Equal(t, tt.cache, got.cache.enabled)
		})
	}
}

func TestBaseClient_SetEndpoint(t *testing.T) {
	type fields struct {
	}
	type args struct {
		endpoint string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "Test SetEndpoint",
			args: args{
				endpoint: "https://potato/graphql",
			},
			want: "https://potato/graphql",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewConnection()
			b.SetEndpoint(tt.args.endpoint)
			assert.Equal(t, tt.want, b.endpoint)
		})
	}
}
