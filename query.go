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
	var queryHash string
	var cachedResponse []byte

	cacheBaseClient := c

	query := cacheBaseClient.NewQuery(queryContent, queryVariables)

	// Check for library specific headers
	if len(queryHeaders) > 0 {
		queryHeadersModified := cacheBaseClient.parseQueryHeaders(queryHeaders)
		// compare if there are any changes

		if !reflect.DeepEqual(queryHeadersModified, queryHeaders) {
			if cacheBaseClient.cache.enabled != c.cache.enabled && cacheBaseClient.cache.enabled {
				cacheBaseClient.Logger.Debug(cacheBaseClient, "Switching cache on as per single-request header")
				cbc := reflect.ValueOf(*c).Interface().(BaseClient)
				cacheBaseClient := &cbc
				cacheBaseClient.enableCache()
			}
		}
	}

	if cacheBaseClient.cache.enabled {
		queryHash = strutil.Md5(fmt.Sprintf("%s-%+v", query.compiledQuery, queryHeaders))
		cacheBaseClient.Logger.Debug(cacheBaseClient, "Hash calculated;", "hash:", queryHash)

		cachedResponse = c.cacheLookup(queryHash)
		if cachedResponse != nil {
			cacheBaseClient.Logger.Debug(cacheBaseClient, "Found cached response")
			return cacheBaseClient.decodeResponse(cachedResponse), nil
		} else {
			cacheBaseClient.Logger.Debug(cacheBaseClient, "No cached response found")
		}
	}

	response, err := cacheBaseClient.executeQuery(query.compiledQuery, queryHeaders)
	if err != nil {
		cacheBaseClient.Logger.Error(cacheBaseClient, "Error while executing query;", "error", err.Error())
		return nil, err
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		cacheBaseClient.Logger.Error(cacheBaseClient, "Error while converting to json;", "error", err.Error())
		return nil, err
	}

	if cacheBaseClient.cache.enabled && jsonData != nil && queryHash != "" {
		c.cache.client.Set(queryHash, jsonData)
	} else if cacheBaseClient.cache.enabled && jsonData == nil {
		cacheBaseClient.Logger.Warn(cacheBaseClient, "Response is empty")
	} else if cacheBaseClient.cache.enabled && queryHash == "" {
		cacheBaseClient.Logger.Warn(cacheBaseClient, "Query hash is empty")
	}

	return cacheBaseClient.decodeResponse(jsonData), err
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
