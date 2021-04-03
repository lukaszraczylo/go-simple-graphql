# Simple GraphQL client

[![Run unit tests](https://github.com/lukaszraczylo/simple-gql-client/actions/workflows/test.yaml/badge.svg)](https://github.com/lukaszraczylo/simple-gql-client/actions/workflows/test.yaml) [![codecov](https://codecov.io/gh/lukaszraczylo/simple-gql-client/branch/master/graph/badge.svg?token=GS3IPOIWDH)](https://codecov.io/gh/lukaszraczylo/simple-gql-client) [![Go Reference](https://pkg.go.dev/badge/github.com/lukaszraczylo/simple-gql-client.svg)](https://pkg.go.dev/github.com/lukaszraczylo/simple-gql-client)

Ps. It's Hasura friendly.

- [Simple GraphQL client](#simple-graphql-client)
  - [Reasoning](#reasoning)
  - [Features](#features)
  - [Usage example](#usage-example)
    - [Setting GraphQL endpoint](#setting-graphql-endpoint)
    - [Example reader code](#example-reader-code)
  - [Working with results](#working-with-results)

## Reasoning

I've tried to run few graphQL clients with hasura, all of them required conversion of the data into
the appropriate structures, causing issues with non-existing types ( thanks to Hasura ), for example `bigint` which was difficult to export.
Therefore, I present you the simple client to which you can copy & paste your graphQL query, variables and you are good to go.

## Features

* Executing GraphQL queries as they are, without types declaration
* Compressing produced queries
* Support for additional headers
* Support for gzip compression ( built into Hasura )

## Usage example

### Setting GraphQL endpoint

You can set the endpoint variable within your code

```go
gql.GraphQLUrl = "http://127.0.0.1:9090/v1/graphql"
```

or as an environment variable `GRAPHQL_ENDPOINT=http://127.0.0.1:9090/v1/graphql`

### Example reader code


```go
import (
  fmt
  gql "github.com/lukaszraczylo/simple-gql-client"
)

headers := map[string]interface{}{
  "x-hasura-user-id":   37,
  "x-hasura-user-uuid": "bde3262e-b42e-4151-ac10-d43f0bef44a5",
}

variables := map[string]interface{}{
"fileHash": "123deadc0w321",
}
var query = `query searchFileKnown($fileHash: String) {
  tbl_file_scans(where: {file_hash: {_eq: $fileHash}}) {
  	porn
  	racy
  	violence
  	virus
  }
}`
result, err := Query(query, variables, nil)
if err != nil {
  fmt.Println("Query error", err)
  return
}
fmt.Println(result)
`
result := Query(query, variables, headers)
fmt.Println(result)
```

**Result**

```json
{"tbl_user_group_admins":[{"id":109,"is_admin":1}]}
```

## Working with results

I'm using an amazing library [tidwall/gjson](https://github.com/tidwall/gjson) to parse the results and extract information required in further steps and I strongly recommend this approach as the easiest and close to painless.

```go
result = gjson.Get(result, "tbl_user_group_admins.0.is_admin").Bool()
if result {
  fmt.Println("User is an admin")
}
```