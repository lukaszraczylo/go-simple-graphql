package gql

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func (suite *TestSuite) Test_GraphQL_queryBuilder() {

	type args struct {
		queryContent   string
		queryVariables interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "Valid query",
			args: args{
				queryContent:   `query listUserBots { tbl_bots { bot_name } }`,
				queryVariables: nil,
			},
			want:    []byte(`{"query":"query listUserBots { tbl_bots { bot_name } }","variables":null}`),
			wantErr: false,
		},
		{
			name: "Valid query with variables",
			args: args{
				queryContent:   `query listUserBots { tbl_bots { bot_name } }`,
				queryVariables: map[string]interface{}{"user_id": 1},
			},
			want:    []byte(`{"query":"query listUserBots { tbl_bots { bot_name } }","variables":{"user_id":1}}`),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			q := NewConnection()
			got, gotErr := q.queryBuilder(tt.args.queryContent, tt.args.queryVariables)
			assert.Equal(t, tt.want, got)
			if tt.wantErr {
				assert.Error(t, gotErr)
			}
		})
	}
}

func (suite *TestSuite) Test_GraphQL_Query() {
	if testing.Short() {
		suite.T().Skip("Skipping test in short / CI mode")
	}
	type args struct {
		queryContent   string
		queryVariables interface{}
	}
	tests := []struct {
		name       string
		endpoint   string
		isLocal    bool
		args       args
		wantResult string
		wantErr    bool
	}{
		{
			name:     "Valid query",
			endpoint: "http://hasura.local/v1/graphql",
			isLocal:  true,
			args: args{
				queryContent: `query listUserBots {
					tbl_bots(limit: 1) {
						bot_name
					}
				}`,
				queryVariables: nil,
			},
			wantResult: `{"tbl_bots":[{"bot_name":"littleMentionBot"}]}`,
			wantErr:    false,
		},
		{
			name:     "Invalid query",
			endpoint: "http://hasura.local/v1/graphql",
			isLocal:  true,
			args: args{
				queryContent: `query listUserBots {
					tbl_botz(limit: 1) {
						bot_name
					}
				}`,
				queryVariables: nil,
			},
			wantResult: ``,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			if tt.isLocal {
				os.Setenv("GRAPHQL_ENDPOINT", tt.endpoint)
			}
			g := NewConnection()
			gotResult, gotErr := g.Query(tt.args.queryContent, tt.args.queryVariables, nil)
			if tt.wantErr {
				assert.Error(t, gotErr)
			}
			assert.Equal(t, tt.wantResult, gotResult)
		})
	}
}
