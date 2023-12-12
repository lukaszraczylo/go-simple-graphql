package gql

import (
	"testing"
)

func (suite *Tests) TestNewConnection() {
	tests := []struct {
		name string
	}{
		{
			name: "TestNewConnection",
		},
	}
	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			got := NewConnection()
			assert.NotNil(suite.T(), got)
		})
	}
}
