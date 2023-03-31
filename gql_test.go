package gql

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func mockGraphQLServerResponses(responseQuery string) (*httptest.Server, string) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte(responseQuery))
	}))
	serverURL := server.URL + "/v1/graphql"
	return server, serverURL
}

type TestSuite struct {
	suite.Suite
}

func (suite *TestSuite) SetupTest() {
	os.Unsetenv("GRAPHQL_ENDPOINT")
}

func TestSuiteRun(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

func (suite *TestSuite) TestNewConnection() {
	type args struct {
		endpoint string
	}
	tests := []struct {
		want *GraphQL
		name string
		args args
	}{
		{
			name: "New connection: Env variable endpoint",
			args: args{
				endpoint: "http://localhost:8080/graphql",
			},
			want: &GraphQL{
				Endpoint: "http://localhost:8080/graphql",
			},
		},
		{
			name: "New connection: No env variable endpoint",
			args: args{},
			want: &GraphQL{
				Endpoint: "http://127.0.0.1:9090/v1/graphql",
			},
		},
	}
	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			os.Setenv("GRAPHQL_ENDPOINT", tt.args.endpoint)
			got := NewConnection()
			assert.Equal(t, tt.want.Endpoint, got.Endpoint)
		})
	}
}
