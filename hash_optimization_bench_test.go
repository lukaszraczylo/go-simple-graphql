package gql

import (
	"crypto/md5"
	"encoding/hex"
	"hash/fnv"
	"testing"
)

// Benchmark comparing MD5 vs FNV hash performance
func BenchmarkHashAlgorithms(b *testing.B) {
	testData := []byte(`{"query":"query GetUser($id: ID!) { user(id: $id) { id name email profile { avatar bio } posts { id title content createdAt } } }","variables":{"id":"12345"}}`)

	b.Run("MD5", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			hash := md5.Sum(testData)
			_ = hex.EncodeToString(hash[:])
		}
	})

	b.Run("FNV64a", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			hash := fnv.New64a()
			hash.Write(testData)
			_ = hex.EncodeToString(hash.Sum(nil))
		}
	})
}

// Benchmark the new calculateHash function
func BenchmarkCalculateHash(b *testing.B) {
	query := &Query{
		Query: "query GetUser($id: ID!) { user(id: $id) { id name email profile { avatar bio } posts { id title content createdAt } } }",
		Variables: map[string]interface{}{
			"id": "12345",
		},
		JsonQuery: []byte(`{"query":"query GetUser($id: ID!) { user(id: $id) { id name email profile { avatar bio } posts { id title content createdAt } } }","variables":{"id":"12345"}}`),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = calculateHash(query)
	}
}

// Benchmark hash collision rate (quality test)
func BenchmarkHashCollisions(b *testing.B) {
	queries := make([]*Query, 1000)
	for i := 0; i < 1000; i++ {
		queries[i] = &Query{
			Query: "query GetUser($id: ID!) { user(id: $id) { id name email } }",
			Variables: map[string]interface{}{
				"id": i,
			},
			JsonQuery: []byte(`{"query":"query GetUser($id: ID!) { user(id: $id) { id name email } }","variables":{"id":` + string(rune(i)) + `}}`),
		}
	}

	b.Run("FNV_Collision_Test", func(b *testing.B) {
		hashes := make(map[string]bool)
		collisions := 0

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, query := range queries {
				hash := calculateHash(query)
				if hashes[hash] {
					collisions++
				} else {
					hashes[hash] = true
				}
			}
		}
		b.ReportMetric(float64(collisions), "collisions")
	})
}
