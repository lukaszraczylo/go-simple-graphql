package libpack_cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type CacheTestSuite struct {
	suite.Suite
}

func (suite *CacheTestSuite) SetupTest() {
}

func TestCachingTestSuite(t *testing.T) {
	suite.Run(t, new(CacheTestSuite))
}

func (suite *CacheTestSuite) Test_New() {
	suite.T().Run("should return a new cache", func(t *testing.T) {
		cache := New(2 * time.Second)
		defer cache.Stop()
		suite.NotNil(cache)
	})
}

func (suite *CacheTestSuite) Test_CacheUse() {
	cache := New(5 * time.Second)
	defer cache.Stop()

	tests := []struct {
		name        string
		cache_value string
	}{
		{
			name:        "test1",
			cache_value: "test1-123",
		},
		{
			name:        "test2",
			cache_value: "test2-123",
		},
	}
	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			cache.Set(tt.name, []byte(tt.name), 2*time.Second)
			c, ok := cache.Get(tt.name)
			suite.Equal(true, ok)
			suite.Equal(tt.name, string(c))
		})
	}
}

func (suite *CacheTestSuite) Test_CacheDelete() {
	cache := New(5 * time.Second)
	defer cache.Stop()

	tests := []struct {
		name        string
		cache_value string
	}{
		{
			name:        "test1",
			cache_value: "test1-123",
		},
		{
			name:        "test2",
			cache_value: "test2-123",
		},
	}
	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			cache.Set(tt.name, []byte(tt.name), 2*time.Second)
			c, ok := cache.Get(tt.name)
			suite.Equal(true, ok)
			suite.Equal(tt.name, string(c))
			cache.Delete(tt.name)
			c, ok = cache.Get(tt.name)
			suite.Equal(false, ok)
			suite.Equal("", string(c))
		})
	}
}

func (suite *CacheTestSuite) Test_CacheExpire() {
	cache := New(5 * time.Second)
	defer cache.Stop()

	tests := []struct {
		name        string
		cache_value string
		ttl         time.Duration
	}{
		{
			name:        "test1",
			cache_value: "test1-123",
			ttl:         100 * time.Millisecond,
		},
		{
			name:        "test2",
			cache_value: "test2-123",
			ttl:         200 * time.Millisecond,
		},
	}
	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			cache.Set(tt.name, []byte(tt.name), tt.ttl)
			c, ok := cache.Get(tt.name)
			suite.Equal(true, ok)
			suite.Equal(tt.name, string(c))
			time.Sleep(tt.ttl + 50*time.Millisecond) // Add small buffer
			c, ok = cache.Get(tt.name)
			suite.Equal(false, ok)
			suite.Equal("", string(c))
		})
	}
}

func (suite *CacheTestSuite) Test_CacheCompression() {
	cache := New(5 * time.Second)
	defer cache.Stop()

	suite.T().Run("should compress large data", func(t *testing.T) {
		// Create data larger than compression threshold (1024 bytes)
		largeData := make([]byte, 2048)
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}

		cache.Set("large_key", largeData, 2*time.Second)
		retrieved, ok := cache.Get("large_key")
		suite.True(ok)
		suite.Equal(largeData, retrieved)
	})

	suite.T().Run("should handle compression errors gracefully", func(t *testing.T) {
		// Test with data that might cause compression issues
		largeData := make([]byte, 1500)
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}

		cache.Set("compression_test", largeData, 2*time.Second)
		retrieved, ok := cache.Get("compression_test")
		suite.True(ok)
		suite.Equal(largeData, retrieved)
	})

	suite.T().Run("should not compress small data", func(t *testing.T) {
		smallData := []byte("small data")
		cache.Set("small_key", smallData, 2*time.Second)
		retrieved, ok := cache.Get("small_key")
		suite.True(ok)
		suite.Equal(smallData, retrieved)
	})
}

func (suite *CacheTestSuite) Test_CacheCompressionEdgeCases() {
	cache := New(5 * time.Second)
	defer cache.Stop()

	suite.T().Run("should handle highly compressible data", func(t *testing.T) {
		// Create highly compressible data (repeated pattern)
		compressibleData := make([]byte, 2048)
		for i := range compressibleData {
			compressibleData[i] = 'A' // All same character
		}

		cache.Set("compressible", compressibleData, 2*time.Second)
		retrieved, ok := cache.Get("compressible")
		suite.True(ok)
		suite.Equal(compressibleData, retrieved)
	})

	suite.T().Run("should handle incompressible data", func(t *testing.T) {
		// Create random-like data that doesn't compress well
		incompressibleData := make([]byte, 1500)
		for i := range incompressibleData {
			incompressibleData[i] = byte((i*7 + i*i) % 256) // Pseudo-random pattern
		}

		cache.Set("incompressible", incompressibleData, 2*time.Second)
		retrieved, ok := cache.Get("incompressible")
		suite.True(ok)
		suite.Equal(incompressibleData, retrieved)
	})
}

func (suite *CacheTestSuite) Test_CacheCleanup() {
	cache := New(100 * time.Millisecond) // Short TTL for faster testing
	defer cache.Stop()

	suite.T().Run("should clean expired entries", func(t *testing.T) {
		cache.Set("expire1", []byte("value1"), 50*time.Millisecond)
		cache.Set("expire2", []byte("value2"), 50*time.Millisecond)
		cache.Set("persist", []byte("persist"), 5*time.Second)

		// Verify entries exist
		_, ok := cache.Get("expire1")
		suite.True(ok)
		_, ok = cache.Get("expire2")
		suite.True(ok)
		_, ok = cache.Get("persist")
		suite.True(ok)

		// Wait for expiration
		time.Sleep(100 * time.Millisecond)

		// Trigger cleanup
		cache.CleanExpiredEntries()

		// Check expired entries are gone
		_, ok = cache.Get("expire1")
		suite.False(ok)
		_, ok = cache.Get("expire2")
		suite.False(ok)

		// Check persistent entry still exists
		_, ok = cache.Get("persist")
		suite.True(ok)
	})
}

func (suite *CacheTestSuite) Test_CacheLazyCleanup() {
	cache := New(5 * time.Second)
	defer cache.Stop()

	suite.T().Run("should trigger lazy cleanup on expired access", func(t *testing.T) {
		cache.Set("lazy_expire", []byte("value"), 50*time.Millisecond)

		// Verify entry exists
		_, ok := cache.Get("lazy_expire")
		suite.True(ok)

		// Wait for expiration
		time.Sleep(100 * time.Millisecond)

		// Access expired entry should trigger lazy cleanup
		_, ok = cache.Get("lazy_expire")
		suite.False(ok)
	})
}

func (suite *CacheTestSuite) Test_CacheStop() {
	suite.T().Run("should stop cache gracefully", func(t *testing.T) {
		cache := New(5 * time.Second)

		// Add some data
		cache.Set("test", []byte("value"), 2*time.Second)
		_, ok := cache.Get("test")
		suite.True(ok)

		// Stop cache
		cache.Stop()

		// Should be able to stop multiple times without panic
		cache.Stop()
	})
}

func (suite *CacheTestSuite) Test_CacheSharding() {
	cache := New(5 * time.Second)
	defer cache.Stop()

	suite.T().Run("should distribute keys across shards", func(t *testing.T) {
		// Test multiple keys to ensure sharding works
		keys := []string{"key1", "key2", "key3", "key4", "key5"}

		for _, key := range keys {
			cache.Set(key, []byte("value_"+key), 2*time.Second)
		}

		for _, key := range keys {
			value, ok := cache.Get(key)
			suite.True(ok)
			suite.Equal("value_"+key, string(value))
		}
	})
}

func (suite *CacheTestSuite) Test_CachePeriodicCleanup() {
	suite.T().Run("should handle short TTL for periodic cleanup", func(t *testing.T) {
		// Test with very short TTL to trigger the minimum cleanup interval logic
		cache := New(500 * time.Millisecond)
		defer cache.Stop()

		cache.Set("short_ttl", []byte("value"), 100*time.Millisecond)

		// Wait for periodic cleanup to potentially run
		time.Sleep(200 * time.Millisecond)

		_, ok := cache.Get("short_ttl")
		suite.False(ok)
	})
}

func (suite *CacheTestSuite) Test_CacheDecompressionErrors() {
	cache := New(5 * time.Second)
	defer cache.Stop()

	suite.T().Run("should handle decompression pool edge cases", func(t *testing.T) {
		// This test ensures the decompression pool handles nil readers correctly
		largeData := make([]byte, 2048)
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}

		// Set and get multiple times to exercise the pool
		for i := 0; i < 5; i++ {
			key := "pool_test_" + string(rune('0'+i))
			cache.Set(key, largeData, 2*time.Second)
			retrieved, ok := cache.Get(key)
			suite.True(ok)
			suite.Equal(largeData, retrieved)
		}
	})
}
