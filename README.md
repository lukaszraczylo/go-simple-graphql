# Simple GraphQL client

[![Run unit tests](https://github.com/lukaszraczylo/simple-gql-client/actions/workflows/test.yaml/badge.svg)](https://github.com/lukaszraczylo/simple-gql-client/actions/workflows/test.yaml)

It's Hasura friendly.

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

```go
import (
  fmt
  gql "github.com/lukaszraczylo/simple-gql-client"
)

// (...)

gql.GraphQLUrl = "http://127.0.0.1:9090/v1/graphql"

headers := map[string]interface{}{
  "x-hasura-user-id":   37,
  "x-hasura-user-uuid": "bde3262e-b42e-4151-ac10-d43fb38f44a5",
}
variables := map[string]interface{}{
  "UserID":  37,
  "GroupID": 11007,
}
var query = `query checkifUserIsAdmin($UserID: bigint, $GroupID: bigint) {
  tbl_user_group_admins(where: {is_admin: {_eq: "1"}, user_id: {_eq: $UserID}, group_id: {_eq: $GroupID}}) {
    id
    is_admin
  }
}`
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