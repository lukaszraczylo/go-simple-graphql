package gql

import (
	"bytes"
	"testing"
)

// Benchmark tiered buffer pools vs single pool
func BenchmarkBufferPools(b *testing.B) {
	// Test data of different sizes
	smallData := make([]byte, 512)
	mediumData := make([]byte, 2048)
	largeData := make([]byte, 8192)

	for i := range smallData {
		smallData[i] = byte(i % 256)
	}
	for i := range mediumData {
		mediumData[i] = byte(i % 256)
	}
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	b.Run("SmallBuffer_GetPut", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf := getBuffer(len(smallData))
			buf.Write(smallData)
			putBuffer(buf)
		}
	})

	b.Run("MediumBuffer_GetPut", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf := getBuffer(len(mediumData))
			buf.Write(mediumData)
			putBuffer(buf)
		}
	})

	b.Run("LargeBuffer_GetPut", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf := getBuffer(len(largeData))
			buf.Write(largeData)
			putBuffer(buf)
		}
	})

	// Compare with old single pool approach
	b.Run("SinglePool_Small", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf := bufferPool.Get().(*bytes.Buffer)
			buf.Reset()
			buf.Write(smallData)
			bufferPool.Put(buf)
		}
	})

	b.Run("SinglePool_Medium", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf := bufferPool.Get().(*bytes.Buffer)
			buf.Reset()
			buf.Write(mediumData)
			bufferPool.Put(buf)
		}
	})

	b.Run("SinglePool_Large", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf := bufferPool.Get().(*bytes.Buffer)
			buf.Reset()
			buf.Write(largeData)
			bufferPool.Put(buf)
		}
	})
}

// Benchmark buffer allocation patterns
func BenchmarkBufferAllocation(b *testing.B) {
	b.Run("TieredPools_Mixed", func(b *testing.B) {
		sizes := []int{256, 512, 1024, 2048, 4096, 8192, 16384}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			size := sizes[i%len(sizes)]
			buf := getBuffer(size)
			data := make([]byte, size)
			buf.Write(data)
			putBuffer(buf)
		}
	})

	b.Run("DirectAllocation_Mixed", func(b *testing.B) {
		sizes := []int{256, 512, 1024, 2048, 4096, 8192, 16384}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			size := sizes[i%len(sizes)]
			buf := bytes.NewBuffer(make([]byte, 0, size))
			data := make([]byte, size)
			buf.Write(data)
		}
	})
}

// Test buffer pool correctness
func TestBufferPoolCorrectness(t *testing.T) {
	testData := []byte("Hello, World! This is test data for buffer pool testing.")

	// Test small buffer
	buf := getBuffer(len(testData))
	buf.Write(testData)
	if buf.Len() != len(testData) {
		t.Errorf("Buffer length mismatch: expected %d, got %d", len(testData), buf.Len())
	}
	if !bytes.Equal(buf.Bytes(), testData) {
		t.Error("Buffer content mismatch")
	}
	putBuffer(buf)

	// Test that buffer is properly reset when returned to pool
	buf2 := getBuffer(10)
	if buf2.Len() != 0 {
		t.Error("Buffer not properly reset")
	}
	putBuffer(buf2)

	// Test large buffer
	largeData := make([]byte, 10000)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	largeBuf := getBuffer(len(largeData))
	largeBuf.Write(largeData)
	if largeBuf.Len() != len(largeData) {
		t.Errorf("Large buffer length mismatch: expected %d, got %d", len(largeData), largeBuf.Len())
	}
	putBuffer(largeBuf)
}

// Test buffer pool size selection
func TestBufferPoolSizeSelection(t *testing.T) {
	testCases := []struct {
		size         int
		expectedPool string
	}{
		{100, "small"},
		{1024, "small"},
		{1025, "medium"},
		{4096, "medium"},
		{4097, "large"},
		{16384, "large"},
		{20000, "large"},
	}

	for _, tc := range testCases {
		buf := getBuffer(tc.size)
		capacity := buf.Cap()

		var poolType string
		switch {
		case capacity <= 1024:
			poolType = "small"
		case capacity <= 4096:
			poolType = "medium"
		default:
			poolType = "large"
		}

		if poolType != tc.expectedPool {
			t.Errorf("Size %d: expected %s pool, got %s pool (capacity: %d)",
				tc.size, tc.expectedPool, poolType, capacity)
		}
		putBuffer(buf)
	}
}
