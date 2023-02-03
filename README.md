# Simple GraphQL client

[![Run unit tests](https://github.com/lukaszraczylo/simple-gql-client/actions/workflows/test.yaml/badge.svg)](https://github.com/lukaszraczylo/simple-gql-client/actions/workflows/test.yaml) [![codecov](https://codecov.io/gh/lukaszraczylo/simple-gql-client/branch/master/graph/badge.svg?token=GS3IPOIWDH)](https://codecov.io/gh/lukaszraczylo/simple-gql-client) [![Go Reference](https://pkg.go.dev/badge/github.com/lukaszraczylo/simple-gql-client.svg)](https://pkg.go.dev/github.com/lukaszraczylo/simple-gql-client)

Ps. It's Hasura friendly.

- [Simple GraphQL client](#simple-graphql-client)
  - [Reasoning](#reasoning)
  - [Features](#features)
  - [Usage example](#usage-example)
    - [Environment variables](#environment-variables)
    - [Cache](#cache)
    - [Example reader code](#example-reader-code)
    - [Tips](#tips)
  - [Working with results](#working-with-results)

## Reasoning

I've tried to run a few GraphQL clients with Hasura, all of them required conversion of the data into
the appropriate structures, causing issues with non-existing types ( thanks to Hasura ), for example, `bigint` which was difficult to export.
Therefore, I present you the simple client to which you can copy & paste your graphQL query, variables and you are good to go.

## Features

* Executing GraphQL queries as they are, without types declaration
* HTTP2 support!
* Support for additional headers
* Query cache

## Usage example

### Environment variables

* `GRAPHQL_ENDPOINT` - Your GraphQL endpoint. Default: `http://127.0.0.1:9090/v1/graphql`
* `GRAPHQL_CACHE` -  Should the query cache be enabled? Default: `false`
* `GRAPHQL_CACHE_TTL` -  Cache TTL in seconds for SELECT type of queries. Default: `5`
* `LOG_LEVEL` - Logging level. Default: `info`
* `RETRIES_ENABLE` - Should retries be enabled? Default: `false`
* `RETRIES_NUMBER` - Number of retries: Default: `1`
* `RETRIES_DELAY` - Delay in retries in milliseconds. Default: `250`

### Cache

You have two options to enable the cache:

* Use `GRAPHQL_CACHE` environment variable which will enable the cache globally. It may be desired if you want to use the cache for all queries.
* Add `gqlcache: true` header for your query which will enable the cache for this query only with `GRAPHQL_CACHE_TTL` TTL.

Example:

```go
headers := map[string]interface{}{
  ...
  "gqlcache": true,
}
```

### Example reader code


```go
import (
  fmt
  graphql "github.com/lukaszraczylo/go-simple-graphql"
)

func main() {
  headers := map[string]interface{}{
    "x-hasura-user-id":   37,
    "x-hasura-user-uuid": "bde1962e-b42e-1212-ac10-d43fa27f44a5",
  }

  variables := map[string]interface{}{
    "fileHash": "123deadc0w321",
  }

  query := `query searchFileKnown($fileHash: String) {
    tbl_file_scans(where: {file_hash: {_eq: $fileHash}}) {
    	racy
    	violence
    	virus
    }
  }`

  gql := graphql.NewConnection()
  result, err := gql.Query(query, variables, headers)
  if err != nil {
    fmt.Println("Query error", err)
    return
  }
  fmt.Println(result)
}
```

### Tips

* Connection handler ( `gql := graphql.NewConnection()` ) should be created once and reused in the application especially if you run dozens of queries per second. It will allow you also to use cache and http2 to its full potential.


**Result**

```json
{"tbl_user_group_admins":[{"id":109,"is_admin":1}]}
```

## Working with results

I'm using an amazing library [tidwall/gjson](https://github.com/tidwall/gjson) to parse the results and extract the information required in further steps and I strongly recommend this approach as the easiest and close to painless, for example:

```go
result := gjson.Get(result, "tbl_user_group_admins.0.is_admin").Bool()
if result {
  fmt.Println("User is an admin")
}
```