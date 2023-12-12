package gql

import "testing"

func Benchmark_compileQuery(bn *testing.B) {
	b := NewConnection()
	bn.ResetTimer()
	for i := 0; i < bn.N; i++ {
		b.compileQuery(
			`query { viewer { login } }`,
		)
	}
}

func Benchmark_convertToJson(bn *testing.B) {
	b := NewConnection()
	bn.ResetTimer()
	for i := 0; i < bn.N; i++ {
		b.convertToJSON(
			`query { viewer { login } }`,
		)
	}
}
