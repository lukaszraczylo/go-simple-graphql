package gql

import (
	"testing"
	"time"

	cache "github.com/lukaszraczylo/go-simple-graphql/cache"
)

// Benchmark cache compression with different data sizes
func BenchmarkCacheCompression(b *testing.B) {
	c := cache.New(5 * time.Second) // Use shorter TTL for tests
	defer c.Stop()

	// Small data (under 1KB threshold)
	smallData := make([]byte, 512)
	for i := range smallData {
		smallData[i] = byte(i % 256)
	}

	// Medium data (over 1KB threshold)
	mediumData := make([]byte, 2048)
	for i := range mediumData {
		mediumData[i] = byte(i % 256)
	}

	// Large data (much larger than threshold)
	largeData := make([]byte, 8192)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	// Highly compressible data
	compressibleData := make([]byte, 4096)
	for i := range compressibleData {
		compressibleData[i] = 'A' // Repeating character compresses well
	}

	b.Run("SmallData_Set", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			c.Set("small", smallData, time.Minute)
		}
	})

	b.Run("SmallData_Get", func(b *testing.B) {
		c.Set("small", smallData, time.Minute)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = c.Get("small")
		}
	})

	b.Run("MediumData_Set", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			c.Set("medium", mediumData, time.Minute)
		}
	})

	b.Run("MediumData_Get", func(b *testing.B) {
		c.Set("medium", mediumData, time.Minute)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = c.Get("medium")
		}
	})

	b.Run("LargeData_Set", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			c.Set("large", largeData, time.Minute)
		}
	})

	b.Run("LargeData_Get", func(b *testing.B) {
		c.Set("large", largeData, time.Minute)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = c.Get("large")
		}
	})

	b.Run("CompressibleData_Set", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			c.Set("compressible", compressibleData, time.Minute)
		}
	})

	b.Run("CompressibleData_Get", func(b *testing.B) {
		c.Set("compressible", compressibleData, time.Minute)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = c.Get("compressible")
		}
	})
}

// Benchmark cache memory usage with compression
func BenchmarkCacheMemoryUsage(b *testing.B) {
	c := cache.New(5 * time.Second) // Use shorter TTL for tests
	defer c.Stop()

	// Test data that compresses well
	testData := make([]byte, 4096)
	for i := range testData {
		testData[i] = byte(i%10 + '0') // Repeating pattern
	}

	b.Run("WithCompression", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := "key" + string(rune(i%1000))
			c.Set(key, testData, time.Minute)
		}
	})
}

// Test compression effectiveness
func TestCompressionEffectiveness(t *testing.T) {
	c := cache.New(5 * time.Second) // Use shorter TTL for tests
	defer c.Stop()

	// Highly compressible data
	compressibleData := make([]byte, 4096)
	for i := range compressibleData {
		compressibleData[i] = 'A'
	}

	// Random data (less compressible)
	randomData := make([]byte, 4096)
	for i := range randomData {
		randomData[i] = byte(i % 256)
	}

	// Small data (under threshold)
	smallData := make([]byte, 512)
	for i := range smallData {
		smallData[i] = byte(i % 256)
	}

	// Test compressible data
	c.Set("compressible", compressibleData, time.Minute)
	retrieved, ok := c.Get("compressible")
	if !ok {
		t.Error("Failed to retrieve compressible data")
	}
	if len(retrieved) != len(compressibleData) {
		t.Errorf("Data length mismatch: expected %d, got %d", len(compressibleData), len(retrieved))
	}

	// Test random data
	c.Set("random", randomData, time.Minute)
	retrieved, ok = c.Get("random")
	if !ok {
		t.Error("Failed to retrieve random data")
	}
	if len(retrieved) != len(randomData) {
		t.Errorf("Data length mismatch: expected %d, got %d", len(randomData), len(retrieved))
	}

	// Test small data (should not be compressed)
	c.Set("small", smallData, time.Minute)
	retrieved, ok = c.Get("small")
	if !ok {
		t.Error("Failed to retrieve small data")
	}
	if len(retrieved) != len(smallData) {
		t.Errorf("Data length mismatch: expected %d, got %d", len(smallData), len(retrieved))
	}
}
