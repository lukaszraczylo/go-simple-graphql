package libpack_cache

import (
	"testing"
	"time"
)

// Assume that New function initializes the cache and it is defined somewhere in the libpack_cache package.

func BenchmarkCacheSet(b *testing.B) {
	cache := New(5 * time.Second) // Use shorter TTL for tests
	defer cache.Stop()
	key := "benchmark-key"
	value := []byte("benchmark-value")

	b.ResetTimer() // Reset the timer to exclude the setup time from the benchmark

	for i := 0; i < b.N; i++ {
		cache.Set(key, value, 2*time.Second)
	}
}

func BenchmarkCacheGet(b *testing.B) {
	cache := New(5 * time.Second) // Use shorter TTL for tests
	defer cache.Stop()
	key := "benchmark-key"
	value := []byte("benchmark-value")
	cache.Set(key, value, 2*time.Second) // Pre-set a value to retrieve

	b.ResetTimer() // Start timing

	for i := 0; i < b.N; i++ {
		_, _ = cache.Get(key)
	}
}

func BenchmarkCacheExpire(b *testing.B) {
	cache := New(5 * time.Second) // Use shorter TTL for tests and reuse cache
	defer cache.Stop()
	value := []byte("benchmark-value")
	ttl := 5 * time.Millisecond // Setting a short TTL for quick expiration

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "benchmark-expire-key-" + string(rune(i)) // Use unique keys
		cache.Set(key, value, ttl)
		time.Sleep(ttl + time.Millisecond) // Wait for the key to expire with buffer
		_, _ = cache.Get(key)
	}
}
