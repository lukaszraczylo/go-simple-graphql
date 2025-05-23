package gql

import (
	"fmt"
	"strings"

	"github.com/goccy/go-json"
	"github.com/gookit/goutil"
	"github.com/gookit/goutil/strutil"
	libpack_logger "github.com/lukaszraczylo/go-simple-graphql/logging"
)

// minifyGraphQLQuery removes unnecessary whitespace from GraphQL queries while preserving string literals
func minifyGraphQLQuery(query string) string {
	if query == "" {
		return query
	}

	// Pre-allocate with estimated final size (typically 70-80% of original)
	var result strings.Builder
	result.Grow(len(query) * 3 / 4)

	inString := false
	stringChar := byte(0)       // Track whether we're in single or double quotes
	queryBytes := []byte(query) // Convert to bytes once for faster access
	length := len(queryBytes)

	i := 0
	for i < length {
		char := queryBytes[i]

		// Handle string literals - preserve everything inside quotes
		if !inString && (char == '"' || char == '\'') {
			inString = true
			stringChar = char
			result.WriteByte(char)
			i++
		} else if inString && char == stringChar {
			// Check if this quote is escaped (optimized version)
			escaped := false
			backslashCount := 0
			for j := i - 1; j >= 0 && queryBytes[j] == '\\'; j-- {
				backslashCount++
			}
			escaped = backslashCount%2 == 1

			if !escaped {
				inString = false
				stringChar = 0
			}
			result.WriteByte(char)
			i++
		} else if inString {
			// Inside string literal - preserve everything including whitespace
			result.WriteByte(char)
			i++
		} else if isWhitespace(char) {
			// Handle whitespace outside strings - optimized version
			// Skip all consecutive whitespace
			for i < length && isWhitespace(queryBytes[i]) {
				i++
			}

			if i < length && result.Len() > 0 {
				// Get previous character from result buffer
				resultStr := result.String()
				prevChar := resultStr[len(resultStr)-1]
				nextChar := queryBytes[i]

				// Optimized space preservation logic
				needsSpace := false

				// Between alphanumeric characters (field names, variables, types)
				if isAlphaNumeric(prevChar) && isAlphaNumeric(nextChar) {
					needsSpace = true
				} else if prevChar == '}' && isAlphaNumeric(nextChar) {
					// Between closing brace and field name: "} field_name"
					needsSpace = true
				} else if prevChar == ':' && (isAlphaNumeric(nextChar) || nextChar == '$' || nextChar == '"' || nextChar == '\'' || nextChar == '{') {
					// Around colons in type definitions and arguments: "$var: Type", "field: value"
					needsSpace = true
				} else if (isAlphaNumeric(prevChar) || prevChar == '_') && nextChar == ':' {
					needsSpace = true
				} else if prevChar == ',' && nextChar == '$' {
					// After commas in argument lists: "arg1: value1, arg2: value2" - only for variables
					needsSpace = true
				} else if prevChar == ',' && isAlphaNumeric(nextChar) {
					// Special case: space after comma when followed by field that has colon
					// Look ahead to see if this field has a colon (limited lookahead for performance)
					colonFound := false
					lookAheadLimit := min(i+15, length) // Reduced lookahead for performance
					for j := i + 1; j < lookAheadLimit; j++ {
						if queryBytes[j] == ':' {
							colonFound = true
							break
						}
						if !isAlphaNumeric(queryBytes[j]) && queryBytes[j] != '_' {
							break
						}
					}
					if colonFound {
						needsSpace = true
					}
				} else if (prevChar == '!' && isAlphaNumeric(nextChar)) || (isAlphaNumeric(prevChar) && nextChar == '!') {
					// Around exclamation marks in type definitions: "Type !"
					needsSpace = true
				}

				if needsSpace {
					result.WriteByte(' ')
				}
			}
		} else {
			// Regular character - just copy it
			result.WriteByte(char)
			i++
		}
	}

	return result.String()
}

// isAlphaNumeric checks if a character is alphanumeric or underscore
func isAlphaNumeric(char byte) bool {
	return (char >= 'a' && char <= 'z') ||
		(char >= 'A' && char <= 'Z') ||
		(char >= '0' && char <= '9') ||
		char == '_'
}

// isWhitespace checks if a character is whitespace
func isWhitespace(char byte) bool {
	return char == ' ' || char == '\t' || char == '\n' || char == '\r'
}

func (b *BaseClient) convertToJSON(v any) []byte {
	// Estimate size based on query structure for better buffer selection
	estimatedSize := 512 // Base size
	if query, ok := v.(*Query); ok {
		estimatedSize += len(query.Query)
		if query.Variables != nil {
			estimatedSize += len(query.Variables) * 50 // Rough estimate per variable
		}
	}

	buf := getBuffer(estimatedSize)
	buf.Reset() // Ensure buffer is clean before use
	defer putBuffer(buf)

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

func processFlags(variables map[string]interface{}, headers map[string]interface{}) (enableCache, enableRetries bool, cleanedVariables map[string]interface{}) {
	// Start with original variables
	cleanedVariables = variables

	// Check headers first
	enableCache, _ = goutil.ToBool(searchForKeysInMapStringInterface(headers, "gqlcache"))
	enableRetries, _ = goutil.ToBool(searchForKeysInMapStringInterface(headers, "gqlretries"))

	// Check variables and clean them if needed
	if variables != nil {
		varEnableCache, _ := goutil.ToBool(searchForKeysInMapStringInterface(variables, "gqlcache"))
		varEnableRetries, _ := goutil.ToBool(searchForKeysInMapStringInterface(variables, "gqlretries"))

		enableCache = enableCache || varEnableCache
		enableRetries = enableRetries || varEnableRetries

		// Clean variables if flags are present (regardless of their value)
		_, hasCacheFlag := variables["gqlcache"]
		_, hasRetriesFlag := variables["gqlretries"]

		if hasCacheFlag || hasRetriesFlag {
			cleanedVariables = make(map[string]interface{})
			for k, v := range variables {
				if k != "gqlcache" && k != "gqlretries" {
					cleanedVariables[k] = v
				}
			}
		}
	}

	return enableCache, enableRetries, cleanedVariables
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

	// Apply query minification if enabled (default: true)
	finalQuery := query
	if b.minify_queries {
		originalSize := len(query)
		minifiedQuery := minifyGraphQLQuery(query)
		minifiedSize := len(minifiedQuery)

		// Log the minification results if there was a reduction
		if originalSize != minifiedSize {
			b.Logger.Debug(&libpack_logger.LogMessage{
				Message: "GraphQL query minified",
				Pairs: map[string]interface{}{
					"original_size":  originalSize,
					"minified_size":  minifiedSize,
					"size_reduction": originalSize - minifiedSize,
					"reduction_pct":  float64(originalSize-minifiedSize) / float64(originalSize) * 100,
				},
			})
		}
		finalQuery = minifiedQuery
	}

	// Construct query object once with final query
	q := &Query{
		Query:     finalQuery,
		Variables: variables,
	}
	q.JsonQuery = b.convertToJSON(q)
	return q
}

func (b *BaseClient) Query(query string, variables map[string]interface{}, headers map[string]interface{}) (any, error) {
	// Process flags before compilation to avoid recompilation
	enableCache, enableRetries, cleanedVariables := processFlags(variables, headers)

	// Compile query once with cleaned variables
	compiledQuery := b.compileQuery(query, cleanedVariables)
	if compiledQuery == nil || compiledQuery.JsonQuery == nil {
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
