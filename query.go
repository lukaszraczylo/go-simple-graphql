package gql

import (
	"bytes"
	"fmt"

	"github.com/goccy/go-json"
	"github.com/gookit/goutil"
	"github.com/gookit/goutil/strutil"
	libpack_logger "github.com/lukaszraczylo/go-simple-graphql/logging"
)

func (b *BaseClient) convertToJSON(v any) []byte {
	// Reuse buffer to reduce allocations
	buf := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buf)
	buf.Reset()

	// Use json.Marshal to avoid adding extra newline
	jsonData, err := json.Marshal(v)
	if err != nil {
		b.Logger.Error(&libpack_logger.LogMessage{
			Message: "Can't convert to JSON",
			Pairs:   map[string]interface{}{"error": err.Error()},
		})
		return nil
	}

	// Copy the bytes to the buffer
	buf.Write(jsonData)

	// Copy the bytes to a new slice before returning
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result
}

func (b *BaseClient) compileQuery(queryPartials ...any) *Query {
	var query string
	var variables map[string]interface{}
	for _, partial := range queryPartials {
		switch val := partial.(type) {
		case string:
			query = val
		case map[string]interface{}:
			variables = val
		}
	}

	if query == "" {
		b.Logger.Error(&libpack_logger.LogMessage{
			Message: "Can't compile query",
			Pairs:   map[string]interface{}{"error": "query is empty"},
		})
		return nil
	}

	jsonQuery := b.convertToJSON(&Query{Query: query, Variables: variables})
	return &Query{
		Query:     query,
		Variables: variables,
		JsonQuery: jsonQuery,
	}
}

func (b *BaseClient) Query(query string, variables map[string]interface{}, headers map[string]interface{}) (any, error) {
	compiledQuery := b.compileQuery(query, variables)
	if compiledQuery.JsonQuery == nil {
		b.Logger.Error(&libpack_logger.LogMessage{
			Message: "Can't compile query",
			Pairs:   map[string]interface{}{"error": "query is empty"},
		})
		return nil, fmt.Errorf("can't compile query")
	}
	b.Logger.Debug(&libpack_logger.LogMessage{
		Message: "Compiled query",
		Pairs:   map[string]interface{}{"query": compiledQuery},
	})

	enableCache, enableRetries, recompileRequired := compiledQuery.parseHeadersAndVariables(headers)
	if recompileRequired {
		compiledQuery = b.compileQuery(query, variables)
	}

	var queryHash string
	if (enableCache || b.cache_global) && strutil.HasPrefix(compiledQuery.Query, "query") {
		b.Logger.Debug(&libpack_logger.LogMessage{
			Message: "Cache enabled",
			Pairs:   nil,
		})
		queryHash = calculateHash(compiledQuery)
		if cachedValue := b.cacheLookup(queryHash); cachedValue != nil {
			b.Logger.Debug(&libpack_logger.LogMessage{
				Message: "Cache hit",
				Pairs:   map[string]interface{}{"query": compiledQuery},
			})
			return b.decodeResponse(cachedValue)
		}
		b.Logger.Debug(&libpack_logger.LogMessage{
			Message: "Cache miss",
			Pairs:   map[string]interface{}{"query": compiledQuery},
		})
	}

	q := &QueryExecutor{
		BaseClient: b,
		Query:      compiledQuery.JsonQuery,
		Headers:    headers,
		CacheKey: func() string {
			if queryHash != "" {
				return queryHash
			}
			return "no-cache"
		}(),
		Retries: enableRetries || b.retries_enable,
	}

	rv, err := q.executeQuery()
	if err != nil {
		b.Logger.Error(&libpack_logger.LogMessage{
			Message: "Error executing query",
			Pairs:   map[string]interface{}{"error": err.Error()},
		})
		return nil, err
	}

	return b.decodeResponse(rv)
}

func (q *Query) parseHeadersAndVariables(headers map[string]interface{}) (enableCache, enableRetries, recompileRequired bool) {
	enableCache, _ = goutil.ToBool(searchForKeysInMapStringInterface(headers, "gqlcache"))
	enableRetries, _ = goutil.ToBool(searchForKeysInMapStringInterface(headers, "gqlretries"))

	if q.Variables != nil {
		varEnableCache, _ := goutil.ToBool(searchForKeysInMapStringInterface(q.Variables, "gqlcache"))
		varEnableRetries, _ := goutil.ToBool(searchForKeysInMapStringInterface(q.Variables, "gqlretries"))
		enableCache = enableCache || varEnableCache
		enableRetries = enableRetries || varEnableRetries

		if varEnableCache || varEnableRetries {
			delete(q.Variables, "gqlcache")
			delete(q.Variables, "gqlretries")
			recompileRequired = true
		}
	}

	return enableCache, enableRetries, recompileRequired
}
