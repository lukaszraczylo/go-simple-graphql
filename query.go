package gql

import (
	"fmt"

	"github.com/goccy/go-json"
	"github.com/gookit/goutil"
	"github.com/gookit/goutil/strutil"
)

func (b *BaseClient) convertToJSON(v any) []byte {
	jsonData, err := json.Marshal(v)
	if err != nil {
		b.Logger.Error("Can't convert to JSON", map[string]interface{}{"error": err.Error()})
		return nil
	}
	return jsonData
}

func (b *BaseClient) compileQuery(queryPartials ...any) *Query {
	q := &Query{}
	for _, partial := range queryPartials {
		switch val := partial.(type) {
		case string:
			q.Query = val
		case map[string]interface{}:
			q.Variables = val
		}
	}
	if q.Query == "" {
		b.Logger.Error("Can't compile query", map[string]interface{}{"error": "query is empty"})
		return nil
	}
	q.JsonQuery = b.convertToJSON(q)
	return q
}

func (b *BaseClient) Query(query string, variables map[string]interface{}, headers map[string]interface{}) (any, error) {
	compiledQuery := b.compileQuery(query, variables)
	if compiledQuery.JsonQuery == nil {
		b.Logger.Error("Can't compile query", map[string]interface{}{"error": "query is empty"})
		return nil, fmt.Errorf("can't compile query")
	}
	b.Logger.Debug("Compiled query", map[string]interface{}{"query": compiledQuery})

	enableCache, enableRetries, recompileRequired := compiledQuery.parseHeadersAndVariables(headers)
	if recompileRequired {
		compiledQuery = b.compileQuery(query, variables)
	}

	var queryHash string
	if (enableCache || b.cache_global) && strutil.HasPrefix(compiledQuery.Query, "query") {
		b.Logger.Debug("Cache enabled")
		queryHash = calculateHash(compiledQuery)
		cachedValue := b.cacheLookup(queryHash)
		if cachedValue != nil {
			b.Logger.Debug("Cache hit", map[string]interface{}{"query": compiledQuery})
			return cachedValue, nil
		}
		b.Logger.Debug("Cache miss", map[string]interface{}{"query": compiledQuery})
	}

	if enableRetries || b.retries_enable {
		b.Logger.Debug("Retries enabled")
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
	defer func() { q = nil }()

	rv, err := q.executeQuery()
	if err != nil {
		b.Logger.Error("Error executing query", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	returnedValue, err := q.decodeResponse(rv)
	return returnedValue, err
}

func (q *Query) parseHeadersAndVariables(headers map[string]interface{}) (enableCache, enableRetries, recompileRequired bool) {
	enableCache, _ = goutil.ToBool(searchForKeysInMapStringInterface(headers, "gqlcache"))
	enableRetries, _ = goutil.ToBool(searchForKeysInMapStringInterface(headers, "gqlretries"))

	if q.Variables != nil {
		varEnableCache, _ := goutil.ToBool(searchForKeysInMapStringInterface(q.Variables, "gqlcache"))
		varEnableRetries, _ := goutil.ToBool(searchForKeysInMapStringInterface(q.Variables, "gqlretries"))
		enableCache = enableCache || varEnableCache
		enableRetries = enableRetries || varEnableRetries
		delete(q.Variables, "gqlcache")
		delete(q.Variables, "gqlretries")
		recompileRequired = true
	}

	return enableCache, enableRetries, recompileRequired
}
