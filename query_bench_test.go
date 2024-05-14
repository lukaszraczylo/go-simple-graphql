package gql

import (
	"testing"
)

func Benchmark_compileQuery(bn *testing.B) {
	b := NewConnection()
	bn.ResetTimer()
	for i := 0; i < bn.N; i++ {
		b.compileQuery(
			`query { viewer { login } }`,
		)
	}
}

func Benchmark_compileQueryWithVariables(bn *testing.B) {
	b := NewConnection()
	variables := map[string]interface{}{
		"var1": "value1",
	}
	bn.ResetTimer()
	for i := 0; i < bn.N; i++ {
		b.compileQuery(
			`query { viewer { login } }`,
			variables,
		)
	}
}

func Benchmark_convertToJson(bn *testing.B) {
	b := NewConnection()
	query := `query { viewer { login } }`
	bn.ResetTimer()
	for i := 0; i < bn.N; i++ {
		b.convertToJSON(query)
	}
}

func Benchmark_Query(bn *testing.B) {
	b := NewConnection()
	query := `query { viewer { login } }`
	variables := map[string]interface{}{
		"var1": "value1",
	}
	headers := map[string]interface{}{
		"Authorization": "Bearer token",
		"gqlcache":      "true",
	}
	bn.ResetTimer()
	for i := 0; i < bn.N; i++ {
		_, _ = b.Query(query, variables, headers)
	}
}

func Benchmark_QueryNoCacheNoRetry(bn *testing.B) {
	b := NewConnection()
	query := `query { viewer { login } }`
	variables := map[string]interface{}{
		"var1": "value1",
	}
	headers := map[string]interface{}{
		"Authorization": "Bearer token",
	}
	bn.ResetTimer()
	for i := 0; i < bn.N; i++ {
		_, _ = b.Query(query, variables, headers)
	}
}

func Benchmark_QueryWithEmptyVariables(bn *testing.B) {
	b := NewConnection()
	query := `query { viewer { login } }`
	headers := map[string]interface{}{
		"Authorization": "Bearer token",
		"gqlcache":      "true",
	}
	bn.ResetTimer()
	for i := 0; i < bn.N; i++ {
		_, _ = b.Query(query, nil, headers)
	}
}
