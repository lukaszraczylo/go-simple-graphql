package libpack_cache

import (
	"bytes"
	"compress/gzip"
	"hash/fnv"
	"io"
	"sync"
	"time"
)

type CacheEntry struct {
	ExpiresAt    time.Time
	Value        []byte
	IsCompressed bool
}

const shardCount = 256 // Must be power of 2

type shard struct {
	entries map[string]CacheEntry
	sync.RWMutex
}

type Cache struct {
	compressPool   sync.Pool
	decompressPool sync.Pool
	shards         [shardCount]*shard
	globalTTL      time.Duration
	cleanupChan    chan struct{}
	stopChan       chan struct{}
	stopped        bool
	mu             sync.RWMutex
}

// getShard returns the appropriate shard for a given key
func (c *Cache) getShard(key string) *shard {
	hash := fnv.New32a()
	hash.Write([]byte(key))
	return c.shards[hash.Sum32()%shardCount]
}

func New(globalTTL time.Duration) *Cache {
	cache := &Cache{
		globalTTL:   globalTTL,
		cleanupChan: make(chan struct{}, 1),
		stopChan:    make(chan struct{}),
		compressPool: sync.Pool{
			New: func() interface{} {
				w := gzip.NewWriter(nil)
				return w
			},
		},
		decompressPool: sync.Pool{
			New: func() interface{} {
				r, _ := gzip.NewReader(bytes.NewReader([]byte{}))
				return r
			},
		},
	}

	// Initialize shards
	for i := 0; i < shardCount; i++ {
		cache.shards[i] = &shard{
			entries: make(map[string]CacheEntry),
		}
	}

	go cache.lazyCleanupWorker()
	go cache.periodicCleanupRoutine(globalTTL)
	return cache
}

func (c *Cache) lazyCleanupWorker() {
	for {
		select {
		case <-c.cleanupChan:
			c.mu.RLock()
			if c.stopped {
				c.mu.RUnlock()
				return
			}
			c.mu.RUnlock()
			c.CleanExpiredEntries()
		case <-c.stopChan:
			return
		}
	}
}

func (c *Cache) periodicCleanupRoutine(globalTTL time.Duration) {
	// Use a shorter cleanup interval for better responsiveness, but not less than 1 second
	cleanupInterval := globalTTL / 4
	if cleanupInterval < time.Second {
		cleanupInterval = time.Second
	}

	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.mu.RLock()
			if c.stopped {
				c.mu.RUnlock()
				return
			}
			c.mu.RUnlock()
			c.CleanExpiredEntries()
		case <-c.stopChan:
			return
		}
	}
}

func (c *Cache) triggerLazyCleanup() {
	c.mu.RLock()
	if c.stopped {
		c.mu.RUnlock()
		return
	}
	c.mu.RUnlock()

	select {
	case c.cleanupChan <- struct{}{}:
		// Cleanup triggered
	default:
		// Cleanup already pending, skip
	}
}

func (c *Cache) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stopped {
		return // Already stopped
	}

	c.stopped = true
	close(c.stopChan)
}

func (c *Cache) Set(key string, value []byte, ttl time.Duration) {
	shard := c.getShard(key)
	shard.Lock()
	defer shard.Unlock()

	const compressionThreshold = 1024 // 1KB threshold
	var finalValue []byte
	var isCompressed bool

	if len(value) >= compressionThreshold {
		compressedValue, err := c.compress(value)
		if err != nil {
			// If compression fails, store uncompressed
			finalValue = value
			isCompressed = false
		} else {
			// Only use compression if it actually reduces size
			if len(compressedValue) < len(value) {
				finalValue = compressedValue
				isCompressed = true
			} else {
				finalValue = value
				isCompressed = false
			}
		}
	} else {
		finalValue = value
		isCompressed = false
	}

	shard.entries[key] = CacheEntry{
		Value:        finalValue,
		ExpiresAt:    time.Now().Add(ttl),
		IsCompressed: isCompressed,
	}
}

func (c *Cache) Get(key string) ([]byte, bool) {
	shard := c.getShard(key)
	shard.RLock()
	entry, ok := shard.entries[key]
	if !ok {
		shard.RUnlock()
		return nil, false
	}

	if entry.ExpiresAt.Before(time.Now()) {
		shard.RUnlock()
		// Trigger lazy cleanup instead of immediate deletion
		c.triggerLazyCleanup()
		return nil, false
	}
	shard.RUnlock()

	var value []byte
	var err error

	if entry.IsCompressed {
		value, err = c.decompress(entry.Value)
		if err != nil {
			return nil, false
		}
	} else {
		value = entry.Value
	}

	return value, true
}

func (c *Cache) Delete(key string) {
	shard := c.getShard(key)
	shard.Lock()
	delete(shard.entries, key)
	shard.Unlock()
}

func (c *Cache) CleanExpiredEntries() {
	now := time.Now()
	for _, shard := range c.shards {
		shard.Lock()
		for key, entry := range shard.entries {
			if entry.ExpiresAt.Before(now) {
				delete(shard.entries, key)
			}
		}
		shard.Unlock()
	}
}

func (c *Cache) compress(data []byte) ([]byte, error) {
	w := c.compressPool.Get().(*gzip.Writer)
	defer c.compressPool.Put(w)

	var buf bytes.Buffer
	w.Reset(&buf)
	if _, err := w.Write(data); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (c *Cache) decompress(data []byte) ([]byte, error) {
	r, ok := c.decompressPool.Get().(*gzip.Reader)
	if !ok || r == nil {
		// If r is nil or type assertion fails, create a new gzip.Reader
		var err error
		r, err = gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, err // Handle the error if gzip.NewReader fails
		}
	} else {
		// Reset the existing reader with new data
		if err := r.Reset(bytes.NewReader(data)); err != nil {
			return nil, err // Handle the error if Reset fails
		}
	}
	defer r.Close()

	// Ensure the reader is returned to the pool
	defer c.decompressPool.Put(r)

	// Read all the data from the reader
	decompressedData, err := io.ReadAll(r)
	if err != nil {
		return nil, err // Handle the error if reading fails
	}
	return decompressedData, nil
}
