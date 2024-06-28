package gql

import (
	"time"

	"github.com/gookit/goutil/envutil"
	cache "github.com/lukaszraczylo/go-simple-graphql/cache"
	logging "github.com/lukaszraczylo/go-simple-graphql/logging"
)

func NewConnection() (b *BaseClient) {
	b = &BaseClient{
		endpoint:       envutil.Getenv("GRAPHQL_ENDPOINT", "https://api.github.com/graphql"),
		responseType:   envutil.Getenv("GRAPHQL_OUTPUT", "string"),
		Logger:         logging.New(),
		cache:          cache.New(time.Duration(envutil.GetInt("GRAPHQL_CACHE_TTL", 5)) * time.Second),
		cache_global:   envutil.GetBool("GRAPHQL_CACHE_ENABLED", false),
		retries_enable: envutil.GetBool("GRAPHQL_RETRIES_ENABLE", false),
		retries_delay:  time.Duration(envutil.GetInt("GRAPHQL_RETRIES_DELAY", 250) * int(time.Millisecond)),
		retries_number: envutil.GetInt("GRAPHQL_RETRIES_NUMBER", 3),
	}
	b.client = b.createHttpClient()
	b.Logger.Debug(&logging.LogMessage{
		Message: "Created new GraphQL client connection",
		Pairs: map[string]interface{}{
			"values": b,
		}})
	return b
}

func (b *BaseClient) SetEndpoint(endpoint string) {
	b.endpoint = endpoint
}

func (b *BaseClient) SetOutput(responseType string) {
	// allowed are byte, string, mapstring
	// check if responseType is allowed
	// TODO: implement
	b.responseType = responseType
}
