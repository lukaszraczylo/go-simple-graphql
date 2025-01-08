package gql

import (
	"bytes"
	"sync"
)

var (
	// Pre-sized buffer pool for better performance
	bufferPool = sync.Pool{
		New: func() interface{} {
			// Pre-allocate buffer with 4KB capacity
			b := bytes.NewBuffer(make([]byte, 0, 4096))
			return b
		},
	}

	// Pre-allocate error maps to reduce allocations
	errPairsPool = sync.Pool{
		New: func() interface{} {
			return make(map[string]interface{}, 1)
		},
	}
)
