package gql

import (
	"bytes"
	"sync"
)

var (
	// Tiered buffer pools for different size requirements
	smallBufferPool = sync.Pool{
		New: func() interface{} {
			// Small buffers for typical GraphQL queries (1KB)
			b := bytes.NewBuffer(make([]byte, 0, 1024))
			return b
		},
	}

	mediumBufferPool = sync.Pool{
		New: func() interface{} {
			// Medium buffers for larger queries (4KB)
			b := bytes.NewBuffer(make([]byte, 0, 4096))
			return b
		},
	}

	largeBufferPool = sync.Pool{
		New: func() interface{} {
			// Large buffers for very large queries (16KB)
			b := bytes.NewBuffer(make([]byte, 0, 16384))
			return b
		},
	}

	// Backward compatibility - use medium pool as default
	bufferPool *sync.Pool

	// Pre-allocate error maps to reduce allocations
	errPairsPool = sync.Pool{
		New: func() interface{} {
			return make(map[string]interface{}, 1)
		},
	}
)

func init() {
	bufferPool = &mediumBufferPool
}

// getBuffer returns an appropriately sized buffer based on estimated size
func getBuffer(estimatedSize int) *bytes.Buffer {
	switch {
	case estimatedSize <= 1024:
		return smallBufferPool.Get().(*bytes.Buffer)
	case estimatedSize <= 4096:
		return mediumBufferPool.Get().(*bytes.Buffer)
	default:
		return largeBufferPool.Get().(*bytes.Buffer)
	}
}

// putBuffer returns a buffer to the appropriate pool based on its capacity
func putBuffer(buf *bytes.Buffer) {
	buf.Reset()
	capacity := buf.Cap()

	switch {
	case capacity <= 1024:
		smallBufferPool.Put(buf)
	case capacity <= 4096:
		mediumBufferPool.Put(buf)
	default:
		largeBufferPool.Put(buf)
	}
}
