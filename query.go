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
		b.Logger.Error("Can't convert to json;", map[string]interface{}{"error": err.Error()})
		return nil
	}
	return jsonData
}

func (b *BaseClient) compileQuery(query_partials ...any) *Query {
	q := new(Query)
	for _, partial := range query_partials {
		switch val := partial.(type) {
		case string:
			q.Query = val
		case map[string]interface{}:
			q.Variables = val
		}
	}
	if q.Query == "" {
		b.Logger.Error("Can't compile query;", map[string]interface{}{"error": "query is empty"})
		return nil
	}
	q.JsonQuery = b.convertToJSON(q)
	return q
}

func (b *BaseClient) Query(query string, variables map[string]interface{}, headers map[string]interface{}) (returned_value any, err error) {
	compiledQuery := b.compileQuery(query, variables)
	if compiledQuery.JsonQuery == nil {
		b.Logger.Error("Can't compile query;", map[string]interface{}{"error": "query is empty"})
		return nil, fmt.Errorf("Can't compile query")
	}

	b.Logger.Debug("Compiled query;", map[string]interface{}{"query": compiledQuery})
	enable_cache, enable_retries, recompile_required := compiledQuery.parseHeadersAndVariables(headers)

	if recompile_required {
		compiledQuery = b.compileQuery(query, variables)
	}

	var queryHash string

	if (enable_cache || b.cache_global) && strutil.HasPrefix(compiledQuery.Query, "query") {
		b.Logger.Debug("Cache enabled")
		queryHash = calculateHash(compiledQuery)
		cached_value := b.cacheLookup(queryHash)
		if cached_value != nil {
			b.Logger.Debug("Cache hit", map[string]interface{}{"query": compiledQuery})
			return cached_value, nil
		} else {
			b.Logger.Debug("Cache miss", map[string]interface{}{"query": compiledQuery})
		}
	}

	if enable_retries || b.retries_enable {
		b.Logger.Debug("Retries enabled")
	}

	q := &QueryExecutor{
		BaseClient: b,
		Query:      compiledQuery.JsonQuery,
		Headers:    headers,
		CacheKey: func() string {
			if queryHash != "" {
				return queryHash
			} else {
				return "no-cache"
			}
		}(),
		Retries: enable_retries || b.retries_enable,
	}
	defer func() { q = nil }()
	rv, err := q.executeQuery()
	if err != nil {
		b.Logger.Error("Error while executing query;", map[string]interface{}{"error": err.Error()})
		return nil, err
	}
	returned_value, err = q.decodeResponse(rv)
	return returned_value, err
}

func (q *Query) parseHeadersAndVariables(headers map[string]interface{}) (enable_cache bool, enable_retries bool, recompile_required bool) {
	if headers != nil {
		enable_cache, _ = goutil.ToBool(searchForKeysInMapStringInterface(headers, "gqlcache"))
		enable_retries, _ = goutil.ToBool(searchForKeysInMapStringInterface(headers, "gqlretries"))
	}
	if q.Variables != nil {
		enable_cache, _ = goutil.ToBool(searchForKeysInMapStringInterface(q.Variables, "gqlcache"))
		enable_retries, _ = goutil.ToBool(searchForKeysInMapStringInterface(q.Variables, "gqlretries"))
		delete(q.Variables, "gqlcache")
		delete(q.Variables, "gqlretries")
		recompile_required = true
	}
	return
}
