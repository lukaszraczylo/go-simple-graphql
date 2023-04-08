package gql

import (
	"context"
	"fmt"
	"reflect"
	"strconv"

	"github.com/lukaszraczylo/go-simple-graphql/utils/helpers"

	"github.com/gookit/goutil/strutil"

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
	query := c.NewQuery(queryContent, queryVariables)

	var queryHash string
	var cachedResponse []byte
	var cacheBaseClient BaseClient

	// Check for library specific headers
	if len(queryHeaders) > 0 {
		cacheBaseClient = reflect.ValueOf(*c).Interface().(BaseClient)
		queryHeaders = c.parseQueryHeaders(queryHeaders)
		defer func() {
			*c = cacheBaseClient
		}()
	}

	if c.cache.enabled {
		queryHash = strutil.Md5(fmt.Sprintf("%s-%+v", query.compiledQuery, queryHeaders))
		c.Logger.Debug(c, "Hash calculated;", "hash:", queryHash)

		cachedResponse = c.cacheLookup(queryHash)
		if cachedResponse != nil {
			c.Logger.Debug(c, "Found cached response")
			return c.decodeResponse(cachedResponse), nil
		} else {
			c.Logger.Debug(c, "No cached response found")
		}
	}

	response, err := c.executeQuery(query.compiledQuery, queryHeaders)
	if err != nil {
		c.Logger.Error(c, "Error while executing query;", "error", err.Error())
		return nil, err
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		c.Logger.Error(c, "Error while converting to json;", "error", err.Error())
		return nil, err
	}

	if c.cache.enabled {
		c.cache.client.Set(queryHash, jsonData)
	}

	return c.decodeResponse(jsonData), err
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

func (c *BaseClient) parseQueryHeaders(queryHeaders map[string]interface{}) (returnHeaders map[string]interface{}) {
	returnHeaders = make(map[string]interface{})
	for k, v := range queryHeaders {
		if k == "gqlcache" {
			c.cache.enabled, _ = strconv.ParseBool(fmt.Sprintf("%v", v))
			continue
		}
		if k == "gqlretries" {
			c.retries.enabled, _ = strconv.ParseBool(fmt.Sprintf("%v", v))
			continue
		}
		returnHeaders[k] = v
	}
	return returnHeaders
}
