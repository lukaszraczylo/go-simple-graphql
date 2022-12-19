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
			endpoint:      "https://telegram-bot.app/v1/graphql",
			isLocal:       true,
			cache_enabled: false,
			args: args{
				queryContent: `query packages_prices {
					sub_group_packages_aggregate(where: {enabled: {_eq: true}}) {
						aggregate {
							count
						}
					}
				}`,
				queryVariables: nil,
			},
			wantResult: `{"sub_group_packages_aggregate":{"aggregate":{"count":4}}}`,
			wantErr:    false,
		},
		{
			name:          "Valid query, no cache #2",
			endpoint:      "https://telegram-bot.app/v1/graphql",
			isLocal:       true,
			cache_enabled: false,
			args: args{
				queryContent: `query packages_prices {
					sub_group_packages_aggregate(where: {enabled: {_eq: true}}) {
						aggregate {
							count
						}
					}
				}`,
				queryVariables: nil,
			},
			wantResult: `{"sub_group_packages_aggregate":{"aggregate":{"count":4}}}`,
			wantErr:    false,
		},
		{
			name:          "Valid query, with cache empty",
			endpoint:      "https://telegram-bot.app/v1/graphql",
			isLocal:       true,
			cache_enabled: true,
			args: args{
				queryContent: `query packages_prices {
					sub_group_packages_aggregate(where: {enabled: {_eq: true}}) {
						aggregate {
							count
						}
					}
				}`,
				queryVariables: nil,
			},
			wantResult: `{"sub_group_packages_aggregate":{"aggregate":{"count":4}}}`,
			wantErr:    false,
		},
		{
			name:          "Valid query, with cache filled",
			endpoint:      "https://telegram-bot.app/v1/graphql",
			isLocal:       true,
			cache_enabled: true,
			args: args{
				queryContent: `query packages_prices {
					sub_group_packages_aggregate(where: {enabled: {_eq: true}}) {
						aggregate {
							count
						}
					}
				}`,
				queryVariables: nil,
			},
			wantResult: `{"sub_group_packages_aggregate":{"aggregate":{"count":4}}}`,
			wantErr:    false,
		},
		{
			name:     "Invalid query",
			endpoint: "https://telegram-bot.app/v1/graphql",
			isLocal:  true,
			args: args{
				queryContent: `query packages_pricez {
					sub_group_packages_aggregate(where: {enabled: {_eq: true}}) {
						aggregate {
							count
					}
				}`,
				queryVariables: nil,
			},
			wantResult: ``,
			wantErr:    true,
		},
		{
			name:     "Valid query to https endpoint",
			endpoint: "https://telegram-bot.app/v1/graphql",
			isLocal:  false,
			args: args{
				queryContent: `query packages_prices {
					sub_group_packages_aggregate(where: {enabled: {_eq: true}}) {
						aggregate {
							count
						}
					}
				}`,
				queryVariables: nil,
			},
			wantResult: `{"sub_group_packages_aggregate":{"aggregate":{"count":4}}}`,
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
