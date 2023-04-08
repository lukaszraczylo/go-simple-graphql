package gql

import (
	"context"
	"fmt"
	"strconv"

	"github.com/gookit/goutil/strutil"
	"github.com/lukaszraczylo/go-simple-graphql/utils/helpers"

	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func (c *BaseClient) convertToJson(v any) []byte {
	json, err := json.Marshal(v)
	if err != nil {
		c.Logger.Error(c, "Can't convert to json;", "error", err.Error())
		return nil
	}
	return json
}

func (c *BaseClient) NewQuery(q ...any) *Query {
	query := &Query{}

	if len(q) > 0 {
		for _, v := range q {
			switch val := v.(type) {
			case string:
				query.query = val
			case map[string]interface{}:
				query.variables = val
			case context.Context:
				query.context = val
			}
		}
	}

	query.compiledQuery = c.convertToJson(request{
		Query:     query.query,
		Variables: query.variables,
	})

	c.Logger.Debug(c, "Clearing previously prepared variables and query")
	query.variables = nil

	if query.context == nil {
		query.context = context.Background()
	}

	if c.validate {
		c.Logger.Warn(c, "Validating query is not active")
	}

	query.query = ""
	c.Logger.Debug(c, "Query prepared;", "query", helpers.BytesToString(query.compiledQuery))

	return query
}

// Query is a function that sends a query to the server
// It takes 3 arguments:
// 1. queryContent: the query string
// 2. queryVariables: the variables for the query
// 3. queryHeaders: the headers for the query
// It looks a bit weird because of the backward compatibility

func (c *BaseClient) Query(queryContent string, queryVariables interface{}, queryHeaders map[string]interface{}) (any, error) {

	compiledQuery := c.NewQuery(queryContent, queryVariables)
	parseQueryHeaders, enabledCache, headersModified := c.parseQueryHeaders(queryHeaders)

	var should_cache bool
	var queryHash string

	if headersModified {
		should_cache = enabledCache
	} else {
		should_cache = c.cache.enabled
	}

	if should_cache {
		queryHash = strutil.Md5(fmt.Sprintf("%s-%+v", compiledQuery.compiledQuery, queryHeaders))
	}

	q := &queryExecutor{
		client:       c,
		query:        compiledQuery.compiledQuery,
		headers:      parseQueryHeaders,
		context:      context.Background(),
		should_cache: should_cache,
		hash:         queryHash,
	}

	q.execute()
	defer q.done()

	return q.result.data, q.result.errors
}

func (c *BaseClient) decodeResponse(jsonData []byte) any {
	switch c.responseType {
	case "mapstring":
		var response map[string]interface{}
		err := json.Unmarshal(jsonData, &response)
		if err != nil {
			c.Logger.Error(c, "Error while converting to map[string]interface{};", "error", err.Error())
			return nil
		}
		return response
	case "string":
		return helpers.BytesToString(jsonData)
	case "byte":
		return jsonData
	default:
		c.Logger.Error(c, "Unknown response type", "response", c.responseType)
		return nil
	}
}

func (c *BaseClient) parseQueryHeaders(queryHeaders map[string]interface{}) (returnHeaders map[string]interface{}, cache_enabled bool, headers_modified bool) {
	returnHeaders = make(map[string]interface{})

	for k, v := range queryHeaders {
		if k == "gqlcache" {
			cache_enabled, _ = strconv.ParseBool(fmt.Sprintf("%v", v))
			headers_modified = true
			continue
		}
		if k == "gqlretries" {
			c.retries.enabled, _ = strconv.ParseBool(fmt.Sprintf("%v", v))
			continue
		}
		returnHeaders[k] = v
	}
	return returnHeaders, cache_enabled, headers_modified
}
