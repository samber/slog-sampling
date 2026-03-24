package buffer

import (
	"fmt"
	"sync/atomic"
	"testing"
)

func benchmarkBufferGetOrInsert(b *testing.B, newBuf func(generator func(string) any) Buffer[string]) {
	buf := newBuf(func(k string) any {
		return new(atomic.Int64)
	})
	key := "bench-key"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.GetOrInsert(key)
	}
}

func benchmarkBufferGetOrInsert_ManyKeys(b *testing.B, newBuf func(generator func(string) any) Buffer[string]) {
	buf := newBuf(func(k string) any {
		return new(atomic.Int64)
	})
	keys := make([]string, 1000)
	for i := range keys {
		keys[i] = fmt.Sprintf("key-%d", i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.GetOrInsert(keys[i%len(keys)])
	}
}

func benchmarkBufferGetOrInsert_Parallel(b *testing.B, newBuf func(generator func(string) any) Buffer[string]) {
	buf := newBuf(func(k string) any {
		return new(atomic.Int64)
	})
	keys := make([]string, 100)
	for i := range keys {
		keys[i] = fmt.Sprintf("key-%d", i)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			buf.GetOrInsert(keys[i%len(keys)])
			i++
		}
	})
}

// Unlimited buffer

func BenchmarkUnlimitedBuffer_GetOrInsert(b *testing.B) {
	benchmarkBufferGetOrInsert(b, NewUnlimitedBuffer[string]())
}

func BenchmarkUnlimitedBuffer_GetOrInsert_ManyKeys(b *testing.B) {
	benchmarkBufferGetOrInsert_ManyKeys(b, NewUnlimitedBuffer[string]())
}

func BenchmarkUnlimitedBuffer_GetOrInsert_Parallel(b *testing.B) {
	benchmarkBufferGetOrInsert_Parallel(b, NewUnlimitedBuffer[string]())
}

// LRU buffer

func BenchmarkLRUBuffer_GetOrInsert(b *testing.B) {
	benchmarkBufferGetOrInsert(b, NewLRUBuffer[string](1000))
}

func BenchmarkLRUBuffer_GetOrInsert_ManyKeys(b *testing.B) {
	benchmarkBufferGetOrInsert_ManyKeys(b, NewLRUBuffer[string](1000))
}

func BenchmarkLRUBuffer_GetOrInsert_Parallel(b *testing.B) {
	benchmarkBufferGetOrInsert_Parallel(b, NewLRUBuffer[string](1000))
}

// LFU buffer

func BenchmarkLFUBuffer_GetOrInsert(b *testing.B) {
	benchmarkBufferGetOrInsert(b, NewLFUBuffer[string](1000))
}

func BenchmarkLFUBuffer_GetOrInsert_ManyKeys(b *testing.B) {
	benchmarkBufferGetOrInsert_ManyKeys(b, NewLFUBuffer[string](1000))
}

func BenchmarkLFUBuffer_GetOrInsert_Parallel(b *testing.B) {
	benchmarkBufferGetOrInsert_Parallel(b, NewLFUBuffer[string](1000))
}

// ARC buffer

func BenchmarkARCBuffer_GetOrInsert(b *testing.B) {
	benchmarkBufferGetOrInsert(b, NewARCBuffer[string](1000))
}

func BenchmarkARCBuffer_GetOrInsert_ManyKeys(b *testing.B) {
	benchmarkBufferGetOrInsert_ManyKeys(b, NewARCBuffer[string](1000))
}

func BenchmarkARCBuffer_GetOrInsert_Parallel(b *testing.B) {
	benchmarkBufferGetOrInsert_Parallel(b, NewARCBuffer[string](1000))
}
