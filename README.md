# Simple GraphQL client

[![Run unit tests](https://github.com/lukaszraczylo/simple-gql-client/actions/workflows/test.yaml/badge.svg)](https://github.com/lukaszraczylo/simple-gql-client/actions/workflows/test.yaml) [![codecov](https://codecov.io/gh/lukaszraczylo/simple-gql-client/branch/master/graph/badge.svg?token=GS3IPOIWDH)](https://codecov.io/gh/lukaszraczylo/simple-gql-client) [![Go Reference](https://pkg.go.dev/badge/github.com/lukaszraczylo/simple-gql-client.svg)](https://pkg.go.dev/github.com/lukaszraczylo/simple-gql-client)

Ps. It's Hasura friendly.

- [Simple GraphQL client](#simple-graphql-client)
  - [Reasoning](#reasoning)
  - [Features](#features)
  - [Usage example](#usage-example)
    - [Environment variables](#environment-variables)
    - [Modifiers on the fly](#modifiers-on-the-fly)
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
* HTTP/2 support for HTTPS endpoints (automatic)
* HTTP/1.1 for HTTP endpoints (compatible with all servers)
* Automatic retry with configurable error patterns
* Support for additional headers
* Query cache with compression

## Usage example

### Environment variables

* `GRAPHQL_ENDPOINT` - Your GraphQL endpoint. Default: `http://127.0.0.1:9090/v1/graphql`
* `GRAPHQL_CACHE_ENABLED` -  Should the query cache be enabled? Default: `false`
* `GRAPHQL_CACHE_TTL` -  Cache TTL in seconds for SELECT type of queries. Default: `5`
* `GRAPHQL_OUTPUT` - Output format. Default: `string`, available: `byte`, `string`, `mapstring`
* `LOG_LEVEL` - Logging level. Default: `info` available: `debug`, `info`, `warn`, `error`
* `GRAPHQL_RETRIES_ENABLE` - Should retries be enabled? Default: `false`
* `GRAPHQL_RETRIES_NUMBER` - Number of retries: Default: `3`
* `GRAPHQL_RETRIES_DELAY` - Delay in retries in milliseconds. Default: `250`
* `GRAPHQL_RETRIES_PATTERNS` - Comma-separated error patterns that trigger retries. Default: `postgres,connection,timeout,transaction,could not,temporarily unavailable,deadlock`

### Modifiers on the fly

* `gql.SetEndpoint('your-endpoint-url')` - modifies endpoint, without the need to set the environment variable
* `gql.SetOutput('byte')` - modifies output format, without the need to set the environment variable

### Retries

The library supports automatic retries for both HTTP-level errors and GraphQL errors that match configurable patterns.

**Retry behavior:**
- **HTTP errors** (status codes outside 200-299): Always retried if retries are enabled
- **GraphQL errors**: Only retried if the error message contains one of the configured patterns
- **Non-matching errors**: Fail immediately without retrying (e.g., validation errors, field not found)

**Configuring retryable patterns:**

By default, errors containing these patterns will trigger retries:
- `postgres` - Database connection/transaction errors
- `connection` - Network connection issues
- `timeout` - Request timeout errors
- `transaction` - Database transaction conflicts
- `could not` - Generic transient failures
- `temporarily unavailable` - Service unavailability
- `deadlock` - Database deadlocks

You can customize these patterns using the `GRAPHQL_RETRIES_PATTERNS` environment variable:

```bash
export GRAPHQL_RETRIES_PATTERNS="postgres,connection,timeout,custom-error-pattern"
```

**Example:** To handle flaky Hasura database connections, enable retries:

```bash
export GRAPHQL_RETRIES_ENABLE=true
export GRAPHQL_RETRIES_NUMBER=3
export GRAPHQL_RETRIES_DELAY=250
```

This will retry GraphQL queries up to 3 times when postgres transaction errors occur, with exponential backoff starting at 250ms.

### Cache

You have two options to enable the cache:

* Use `GRAPHQL_CACHE_ENABLED` environment variable which will enable the cache globally. It may be desired if you want to use the cache for all queries.
* Add `gqlcache: true` header for your query which will enable the cache for this query only with `GRAPHQL_CACHE_TTL` TTL.
* You can check the list of supported per-query modifiers below

Example:

```go
// following values passed as headers will modify behaviour of the query
// and disregard settings provided via environment variables
headers := map[string]interface{}{
  ...
  "gqlcache": true, // sets the cache as on for this query only
  "gqlretries": false, // disables retries for this query only
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

Currently attempting to switch to the fork of the [`ask` library](https://github.com/lukaszraczylo/ask)

Before, I used an amazing library [tidwall/gjson](https://github.com/tidwall/gjson) to parse the results and extract the information required in further steps and I strongly recommend this approach as the easiest and close to painless, for example:

```go
result := gjson.Get(result, "tbl_user_group_admins.0.is_admin").Bool()
if result {
  fmt.Println("User is an admin")
}
```