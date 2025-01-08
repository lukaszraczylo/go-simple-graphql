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
	buf := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buf)
	buf.Reset()

	// Use json.NewEncoder for better performance with buffers
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false) // Reduce unnecessary escaping

	if err := enc.Encode(v); err != nil {
		errPairs := errPairsPool.Get().(map[string]interface{})
		errPairs["error"] = err.Error()
		b.Logger.Error(&libpack_logger.LogMessage{
			Message: "Can't convert to JSON",
			Pairs:   errPairs,
		})
		errPairsPool.Put(errPairs)
		return nil
	}

	// Get the buffer bytes directly, trimming the trailing newline
	bytes := buf.Bytes()
	if len(bytes) > 0 && bytes[len(bytes)-1] == '\n' {
		bytes = bytes[:len(bytes)-1]
	}

	// Make a copy of the bytes since the buffer will be reused
	result := make([]byte, len(bytes))
	copy(result, bytes)
	return result
}

func (b *BaseClient) compileQuery(queryPartials ...any) *Query {
	var query string
	var variables map[string]interface{}

	// Pre-allocate the query with an estimated size
	if len(queryPartials) > 0 {
		if str, ok := queryPartials[0].(string); ok {
			query = str
		}
	}

	// Only allocate variables map if we have more than one partial
	if len(queryPartials) > 1 {
		if vars, ok := queryPartials[1].(map[string]interface{}); ok {
			variables = vars
		}
	}

	if query == "" {
		errPairs := errPairsPool.Get().(map[string]interface{})
		errPairs["error"] = "query is empty"
		b.Logger.Error(&libpack_logger.LogMessage{
			Message: "Can't compile query",
			Pairs:   errPairs,
		})
		errPairsPool.Put(errPairs)
		return nil
	}

	// Construct query object once
	q := &Query{
		Query:     query,
		Variables: variables,
	}
	q.JsonQuery = b.convertToJSON(q)
	return q
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
