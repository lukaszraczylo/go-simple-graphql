package gql

import (
	"os"
	"testing"

	"github.com/lukaszraczylo/go-simple-graphql/utils/logger"

	"github.com/stretchr/testify/assert"
)

func TestNewConnection(t *testing.T) {
	tests := []struct {
		want *BaseClient
		name string
	}{
		{
			name: "Test NewConnection",
			want: &BaseClient{
				endpoint:     "https://api.github.com/graphql",
				responseType: "string",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// temporarily set GRAPHQL_ENDPOINT env variable to https://api.github.com/graphql
			// and GRAPHQL_OUTPUT to string
			current := os.Getenv("GRAPHQL_ENDPOINT")
			defer os.Setenv("GRAPHQL_ENDPOINT", current)
			os.Setenv("GRAPHQL_ENDPOINT", "https://api.github.com/graphql")

			got := NewConnection()
			assert.Equal(t, tt.want.endpoint, got.endpoint)
			assert.Equal(t, tt.want.responseType, got.responseType)
		})
	}
}

func TestNewConnectionLogLevels(t *testing.T) {
	tests := []struct {
		want *BaseClient
		name string
		env  string
	}{
		{
			name: "Test NewConnection with log level set to debug",
			env:  "debug",
			want: &BaseClient{
				endpoint:     "https://api.github.com/graphql",
				responseType: "string",
				LoggingLevel: logger.Debug,
			},
		},

		{
			name: "Test NewConnection with log level set to info",
			env:  "info",
			want: &BaseClient{
				endpoint:     "https://api.github.com/graphql",
				responseType: "string",
				LoggingLevel: logger.Info,
			},
		},

		{
			name: "Test NewConnection with log level set to warn",
			env:  "warn",
			want: &BaseClient{
				endpoint:     "https://api.github.com/graphql",
				responseType: "string",
				LoggingLevel: logger.Warn,
			},
		},

		{
			name: "Test NewConnection with log level set to error",
			env:  "error",
			want: &BaseClient{
				endpoint:     "https://api.github.com/graphql",
				responseType: "string",
				LoggingLevel: logger.Error,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// temporarily set GRAPHQL_ENDPOINT env variable to https://api.github.com/graphql
			// and GRAPHQL_OUTPUT to string
			current := os.Getenv("GRAPHQL_ENDPOINT")
			defer os.Setenv("GRAPHQL_ENDPOINT", current)
			os.Setenv("GRAPHQL_ENDPOINT", "https://api.github.com/graphql")
			current_log_level := os.Getenv("LOG_LEVEL")
			defer os.Setenv("LOG_LEVEL", current_log_level)
			os.Setenv("LOG_LEVEL", tt.env)

			got := NewConnection()
			assert.Equal(t, tt.want.endpoint, got.endpoint)
			assert.Equal(t, tt.want.responseType, got.responseType)
			assert.Equal(t, tt.want.LoggingLevel, got.LoggingLevel)
		})
	}
}
