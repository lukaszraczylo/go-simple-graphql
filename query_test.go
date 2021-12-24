package gql

import (
	"fmt"
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
	g := NewConnection()

	type args struct {
		queryContent   string
		queryVariables interface{}
		queryHeaders   map[string]interface{}
	}
	tests := []struct {
		name          string
		endpoint      string
		isLocal       bool
		cache_enabled bool
		args          args
		wantResult    string
		wantErr       bool
	}{
		{
			name:          "Valid query, no cache",
			endpoint:      "https://hasura.local/v1/graphql",
			isLocal:       true,
			cache_enabled: false,
			args: args{
				queryContent: `query listUserBots {
					tbl_bots(limit: 1) {
						bot_name
					}
				}`,
				queryVariables: nil,
			},
			wantResult: `{"tbl_bots":[{"bot_name":"ThetaGuardianBot"}]}`,
			wantErr:    false,
		},
		{
			name:          "Valid query, with cache empty",
			endpoint:      "https://hasura.local/v1/graphql",
			isLocal:       true,
			cache_enabled: true,
			args: args{
				queryContent: `query listUserBots {
					tbl_bots(limit: 1) {
						bot_name
					}
				}`,
				queryVariables: nil,
			},
			wantResult: `{"tbl_bots":[{"bot_name":"ThetaGuardianBot"}]}`,
			wantErr:    false,
		},
		{
			name:          "Valid query, with cache filled",
			endpoint:      "https://hasura.local/v1/graphql",
			isLocal:       true,
			cache_enabled: true,
			args: args{
				queryContent: `query listUserBots {
					tbl_bots(limit: 1) {
						bot_name
					}
				}`,
				queryVariables: nil,
			},
			wantResult: `{"tbl_bots":[{"bot_name":"ThetaGuardianBot"}]}`,
			wantErr:    false,
		},
		{
			name:     "Invalid query",
			endpoint: "https://hasura.local/v1/graphql",
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
		{
			name:     "Valid query to https endpoint",
			endpoint: "https://web-dev.telegram-bot.app/v1/graphql",
			isLocal:  false,
			args: args{
				queryContent: `query packages_prices {
					available_packages: tbl_available_packages(where: {enabled: {_eq: true}}) {
						package_name
						package_discount
						package_price
						package_size
						package_type
					}
				}`,
				queryVariables: nil,
			},
			wantResult: `{"available_packages":[{"package_discount":0,"package_name":"media_1000","package_price":750,"package_size":1000,"package_type":"media"},{"package_discount":5,"package_name":"media_5000","package_price":3563,"package_size":5000,"package_type":"media"},{"package_discount":10,"package_name":"media_10000","package_price":6750,"package_size":10000,"package_type":"media"},{"package_discount":15,"package_name":"media_25000","package_price":15938,"package_size":25000,"package_type":"media"},{"package_discount":0,"package_name":"voice_500","package_price":1000,"package_size":500,"package_type":"voice"},{"package_discount":10,"package_name":"voice_1000","package_price":1800,"package_size":1000,"package_type":"voice"},{"package_discount":15,"package_name":"voice_2500","package_price":4250,"package_size":2500,"package_type":"voice"}]}`,
			wantErr:    false,
		},
		{
			name:     "Valid query to github endpoint",
			endpoint: "https://api.github.com/graphql",
			isLocal:  false,
			args: args{
				queryContent: `query {
					repository(name: "semver-generator", owner: "lukaszraczylo", followRenames: true) {
						releases(last: 2) {
							nodes {
								tag {
									name
								}
							}
						}
					}
				}`,
				queryVariables: nil,
				queryHeaders: map[string]interface{}{
					"Authorization": "Bearer " + os.Getenv("GITHUB_TOKEN"),
				},
			},
			wantResult: `{"repository":{"releases":{"nodes":[`,
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			// if tt.isLocal {
			// 	os.Setenv("GRAPHQL_ENDPOINT", tt.endpoint)
			// }
			g.Endpoint = tt.endpoint
			g.Cache = tt.cache_enabled
			gotResult, gotErr := g.Query(tt.args.queryContent, tt.args.queryVariables, tt.args.queryHeaders)
			if tt.wantErr {
				assert.Error(t, gotErr)
			}
			assert.Contains(t, gotResult, tt.wantResult)
		})
	}
	fmt.Println(g.CacheStore.Capacity())
}
