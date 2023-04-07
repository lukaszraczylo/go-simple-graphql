package main

import (
	"fmt"

	graphql "github.com/lukaszraczylo/go-simple-graphql"
)

func main() {
	gql := graphql.NewConnection()

	query := `query MyQuery {
		bots {
			bot_name
			ddd
		}
	}`
	_, err := gql.Query(query, nil, nil)
	if err != nil {
		fmt.Println("Error returned from query:", err)
	}

	query = `query MyQuery {
		badwords(distinct_on: word) {
			word
		}
	}`
	gql.Query(query, nil, nil)

	// Repeated to use the cached response
	query = `query MyQuery {
		badwords(distinct_on: word) {
			word
		}
	}`
	gql.Query(query, nil, nil)
}
