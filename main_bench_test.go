package gql

import "testing"

func Benchmark_newConnection(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewConnection()
	}
}
