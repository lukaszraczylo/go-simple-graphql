package gql

import (
	"testing"
)

func Benchmark_GraphQL_Query(t *testing.B) {
	if testing.Short() {
		t.Skip("Skipping test in short / CI mode")
	}
	type args struct {
		queryContent   string
		queryVariables interface{}
	}
	tests := []struct {
		name        string
		endpoint    string
		isLocal     bool
		args        args
		wantResult  string
		mockedReply string
		wantErr     bool
	}{
		{
			name:     "Valid query",
			endpoint: "http://hasura.local/v1/graphql",
			isLocal:  false,
			args: args{
				queryContent: `query listUserBots {
					tbl_bots(limit: 1) {
						bot_name
					}
				}`,
				queryVariables: nil,
			},
			wantResult:  `{"tbl_bots":[{"bot_name":"littleMentionBot"}]}`,
			mockedReply: `{"data":{"tbl_bots":[{"bot_name":"littleMentionBot"}]}}`,
			wantErr:     false,
		},
		{
			name:     "Invalid query",
			endpoint: "http://hasura.local/v1/graphql",
			isLocal:  false,
			args: args{
				queryContent: `query listUserBots {
					tbl_botz(limit: 1) {
						bot_name
					}
				}`,
				queryVariables: nil,
			},
			mockedReply: `{"errors":[{"message":"Unknown field \"tbl_botz\" on type \"Query\""}]}`,
			wantResult:  ``,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.B) {
			g := NewConnection()
			server, serverURL := mockGraphQLServerResponses(tt.mockedReply)
			g.Endpoint = serverURL
			for n := 0; n < t.N; n++ {
				g.Query(tt.args.queryContent, tt.args.queryVariables, nil)
			}
			server.Close()
		})
	}
}
