package gql

import (
	"testing"
	"time"

	cache "github.com/lukaszraczylo/go-simple-graphql/cache"
)

// Performance regression tests to ensure optimizations maintain their benefits
func TestPerformanceRegression(t *testing.T) {
	// Test hash performance
	t.Run("HashPerformance", func(t *testing.T) {
		query := &Query{
			Query: "query GetUser($id: ID!) { user(id: $id) { id name email } }",
			Variables: map[string]interface{}{
				"id": "12345",
			},
			JsonQuery: []byte(`{"query":"query GetUser($id: ID!) { user(id: $id) { id name email } }","variables":{"id":"12345"}}`),
		}

		start := time.Now()
		for i := 0; i < 10000; i++ {
			_ = calculateHash(query)
		}
		duration := time.Since(start)

		// Should complete 10k hashes in under 40ms (accounts for race detection + coverage overhead)
		if duration > 40*time.Millisecond {
			t.Errorf("Hash performance regression: 10k hashes took %v, expected < 40ms", duration)
		}
	})

	// Test buffer pool performance
	t.Run("BufferPoolPerformance", func(t *testing.T) {
		start := time.Now()
		for i := 0; i < 10000; i++ {
			buf := getBuffer(1024)
			buf.WriteString("test data")
			putBuffer(buf)
		}
		duration := time.Since(start)

		// Should complete 10k buffer operations in under 30ms (accounts for race detection + coverage overhead)
		if duration > 30*time.Millisecond {
			t.Errorf("Buffer pool performance regression: 10k operations took %v, expected < 30ms", duration)
		}
	})

	// Test cache performance
	t.Run("CachePerformance", func(t *testing.T) {
		c := cache.New(5 * time.Second) // Use shorter TTL for tests
		defer c.Stop()

		testData := make([]byte, 512) // Small data that won't be compressed
		for i := range testData {
			testData[i] = byte(i % 256)
		}

		start := time.Now()
		for i := 0; i < 10000; i++ {
			key := "key" + string(rune(i%100))
			c.Set(key, testData, time.Minute)
			_, _ = c.Get(key)
		}
		duration := time.Since(start)

		// Should complete 10k cache operations in under 50ms
		if duration > 50*time.Millisecond {
			t.Errorf("Cache performance regression: 10k operations took %v, expected < 50ms", duration)
		}
	})

	// Test processFlags performance
	t.Run("ProcessFlagsPerformance", func(t *testing.T) {
		variables := map[string]interface{}{
			"id":         "123",
			"gqlcache":   true,
			"gqlretries": true,
		}
		headers := map[string]interface{}{}

		start := time.Now()
		for i := 0; i < 10000; i++ {
			_, _, _ = processFlags(variables, headers)
		}
		duration := time.Since(start)

		// Should complete 10k flag processing operations in under 30ms (accounts for race detection + coverage overhead)
		if duration > 30*time.Millisecond {
			t.Errorf("ProcessFlags performance regression: 10k operations took %v, expected < 30ms", duration)
		}
	})

	// Test query compilation performance
	t.Run("QueryCompilationPerformance", func(t *testing.T) {
		client := CreateTestClient()

		query := "query GetUser($id: ID!) { user(id: $id) { id name email } }"
		variables := map[string]interface{}{"id": "123"}

		start := time.Now()
		for i := 0; i < 1000; i++ {
			_ = client.compileQuery(query, variables)
		}
		duration := time.Since(start)

		// Should complete 1k query compilations in under 65ms (accounts for minification overhead)
		if duration > 65*time.Millisecond {
			t.Errorf("Query compilation performance regression: 1k operations took %v, expected < 65ms", duration)
		}
	})
}

// Benchmark comprehensive workflow to ensure no regression
func BenchmarkComprehensiveWorkflow(b *testing.B) {
	client := CreateTestClient()

	query := "query GetUser($id: ID!) { user(id: $id) { id name email profile { avatar bio } } }"
	variables := map[string]interface{}{
		"id":         "123",
		"gqlcache":   true,
		"gqlretries": true,
	}
	headers := map[string]interface{}{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Process flags
		enableCache, enableRetries, cleanedVariables := processFlags(variables, headers)

		// Compile query
		compiledQuery := client.compileQuery(query, cleanedVariables)

		// Calculate hash
		if enableCache {
			_ = calculateHash(compiledQuery)
		}

		// Use variables to avoid optimization
		_ = enableRetries
	}
}

// Memory allocation benchmark
func BenchmarkMemoryAllocations(b *testing.B) {
	client := CreateTestClient()

	query := "query GetUser($id: ID!) { user(id: $id) { id name } }"
	variables := map[string]interface{}{"id": "123"}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		compiledQuery := client.compileQuery(query, variables)
		_ = calculateHash(compiledQuery)
	}
}
