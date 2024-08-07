package libpack_cache

import (
	"testing"
	"time"
)

// Assume that New function initializes the cache and it is defined somewhere in the libpack_cache package.

func BenchmarkCacheSet(b *testing.B) {
	cache := New(30 * time.Second) // Initializing the cache with a TTL of 30 seconds
	key := "benchmark-key"
	value := []byte("benchmark-value")

	b.ResetTimer() // Reset the timer to exclude the setup time from the benchmark

	for i := 0; i < b.N; i++ {
		cache.Set(key, value, 5*time.Second)
	}
}

func BenchmarkCacheGet(b *testing.B) {
	cache := New(30 * time.Second) // Initializing the cache
	key := "benchmark-key"
	value := []byte("benchmark-value")
	cache.Set(key, value, 5*time.Second) // Pre-set a value to retrieve

	b.ResetTimer() // Start timing

	for i := 0; i < b.N; i++ {
		_, _ = cache.Get(key)
	}
}

func BenchmarkCacheExpire(b *testing.B) {
	key := "benchmark-expire-key"
	value := []byte("benchmark-value")
	ttl := 5 * time.Millisecond // Setting a short TTL for quick expiration

	for i := 0; i < b.N; i++ {
		cache := New(30 * time.Second)
		cache.Set(key, value, ttl)
		time.Sleep(ttl) // Wait for the key to expire
		_, _ = cache.Get(key)
	}
}
