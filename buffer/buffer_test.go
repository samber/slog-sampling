package buffer

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

// bufferTestCase runs a standard test suite against any Buffer implementation.
func bufferTestCase_NewKey(t *testing.T, newBuf func(func(string) any) Buffer[string]) {
	buf := newBuf(func(k string) any { return "generated-" + k })

	v, isNew := buf.GetOrInsert("key1")
	assert.Equal(t, "generated-key1", v)
	assert.True(t, isNew)
}

func bufferTestCase_ExistingKey(t *testing.T, newBuf func(func(string) any) Buffer[string]) {
	buf := newBuf(func(k string) any { return new(atomic.Int64) })

	v1, _ := buf.GetOrInsert("key1")
	v1.(*atomic.Int64).Add(42)

	v2, _ := buf.GetOrInsert("key1")
	// The important thing: same value is returned (not a new instance)
	assert.Equal(t, int64(42), v2.(*atomic.Int64).Load())
}

func bufferTestCase_ManyKeys(t *testing.T, newBuf func(func(string) any) Buffer[string]) {
	buf := newBuf(func(k string) any { return k })

	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key-%d", i)
		v, _ := buf.GetOrInsert(key)
		assert.Equal(t, key, v)
	}
}

func bufferTestCase_ConcurrentSameKey(t *testing.T, newBuf func(func(string) any) Buffer[string]) {
	var generatorCalls atomic.Int64
	buf := newBuf(func(k string) any {
		generatorCalls.Add(1)
		return new(atomic.Int64)
	})

	const numGoroutines = 100
	wg := sync.WaitGroup{}
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			v, _ := buf.GetOrInsert("shared-key")
			v.(*atomic.Int64).Add(1)
		}()
	}

	wg.Wait()

	// The counter should reflect all goroutines' increments
	v, _ := buf.GetOrInsert("shared-key")
	got := v.(*atomic.Int64).Load()
	assert.True(t, got >= 1 && got <= numGoroutines,
		"counter=%d, expected in [1, %d]", got, numGoroutines)
}

func bufferTestCase_ConcurrentManyKeys(t *testing.T, newBuf func(func(string) any) Buffer[string]) {
	buf := newBuf(func(k string) any { return new(atomic.Int64) })

	const numGoroutines = 100
	const keysPerGoroutine = 100
	wg := sync.WaitGroup{}
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		goroutineIdx := i
		go func() {
			defer wg.Done()
			for j := 0; j < keysPerGoroutine; j++ {
				key := fmt.Sprintf("g%d-k%d", goroutineIdx, j)
				v, _ := buf.GetOrInsert(key)
				v.(*atomic.Int64).Add(1)
			}
		}()
	}

	wg.Wait()
	// No panic, no race = success
}

// Unlimited buffer tests

func TestUnlimitedBuffer_NewKey(t *testing.T) {
	bufferTestCase_NewKey(t, NewUnlimitedBuffer[string]())
}

func TestUnlimitedBuffer_ExistingKey(t *testing.T) {
	bufferTestCase_ExistingKey(t, NewUnlimitedBuffer[string]())
}

func TestUnlimitedBuffer_ManyKeys(t *testing.T) {
	bufferTestCase_ManyKeys(t, NewUnlimitedBuffer[string]())
}

func TestUnlimitedBuffer_ConcurrentSameKey(t *testing.T) {
	bufferTestCase_ConcurrentSameKey(t, NewUnlimitedBuffer[string]())
}

func TestUnlimitedBuffer_ConcurrentManyKeys(t *testing.T) {
	bufferTestCase_ConcurrentManyKeys(t, NewUnlimitedBuffer[string]())
}

// LRU buffer tests

func TestLRUBuffer_NewKey(t *testing.T) {
	bufferTestCase_NewKey(t, NewLRUBuffer[string](100))
}

func TestLRUBuffer_ExistingKey(t *testing.T) {
	bufferTestCase_ExistingKey(t, NewLRUBuffer[string](100))
}

func TestLRUBuffer_ManyKeys(t *testing.T) {
	bufferTestCase_ManyKeys(t, NewLRUBuffer[string](100))
}

func TestLRUBuffer_ConcurrentSameKey(t *testing.T) {
	bufferTestCase_ConcurrentSameKey(t, NewLRUBuffer[string](100))
}

func TestLRUBuffer_ConcurrentManyKeys(t *testing.T) {
	bufferTestCase_ConcurrentManyKeys(t, NewLRUBuffer[string](100))
}

func TestLRUBuffer_Eviction(t *testing.T) {
	var generatorCalls atomic.Int64
	buf := NewLRUBuffer[string](5)(func(k string) any {
		generatorCalls.Add(1)
		return "val-" + k
	})

	// Fill buffer with 5 keys
	for i := 0; i < 5; i++ {
		buf.GetOrInsert(fmt.Sprintf("key-%d", i))
	}
	calls := generatorCalls.Load()
	assert.Equal(t, int64(5), calls)

	// Insert 5 more keys, evicting old ones
	for i := 5; i < 10; i++ {
		buf.GetOrInsert(fmt.Sprintf("key-%d", i))
	}

	// Re-access evicted keys — generator should be called again
	generatorCalls.Store(0)
	for i := 0; i < 5; i++ {
		buf.GetOrInsert(fmt.Sprintf("key-%d", i))
	}
	assert.True(t, generatorCalls.Load() > 0, "expected generator to be called for evicted keys")
}

// LFU buffer tests

func TestLFUBuffer_NewKey(t *testing.T) {
	bufferTestCase_NewKey(t, NewLFUBuffer[string](100))
}

func TestLFUBuffer_ExistingKey(t *testing.T) {
	bufferTestCase_ExistingKey(t, NewLFUBuffer[string](100))
}

func TestLFUBuffer_ManyKeys(t *testing.T) {
	bufferTestCase_ManyKeys(t, NewLFUBuffer[string](100))
}

func TestLFUBuffer_ConcurrentSameKey(t *testing.T) {
	bufferTestCase_ConcurrentSameKey(t, NewLFUBuffer[string](100))
}

func TestLFUBuffer_ConcurrentManyKeys(t *testing.T) {
	bufferTestCase_ConcurrentManyKeys(t, NewLFUBuffer[string](100))
}

func TestLFUBuffer_Eviction(t *testing.T) {
	var generatorCalls atomic.Int64
	buf := NewLFUBuffer[string](5)(func(k string) any {
		generatorCalls.Add(1)
		return "val-" + k
	})

	// Fill + overflow
	for i := 0; i < 10; i++ {
		buf.GetOrInsert(fmt.Sprintf("key-%d", i))
	}

	// Some keys should have been evicted and re-generated
	assert.True(t, generatorCalls.Load() >= 10, "expected at least 10 generator calls for 10 unique keys")
}

// ARC buffer tests

func TestARCBuffer_NewKey(t *testing.T) {
	bufferTestCase_NewKey(t, NewARCBuffer[string](100))
}

func TestARCBuffer_ExistingKey(t *testing.T) {
	bufferTestCase_ExistingKey(t, NewARCBuffer[string](100))
}

func TestARCBuffer_ManyKeys(t *testing.T) {
	bufferTestCase_ManyKeys(t, NewARCBuffer[string](100))
}

func TestARCBuffer_ConcurrentSameKey(t *testing.T) {
	bufferTestCase_ConcurrentSameKey(t, NewARCBuffer[string](100))
}

func TestARCBuffer_ConcurrentManyKeys(t *testing.T) {
	bufferTestCase_ConcurrentManyKeys(t, NewARCBuffer[string](100))
}

func TestARCBuffer_Eviction(t *testing.T) {
	var generatorCalls atomic.Int64
	buf := NewARCBuffer[string](5)(func(k string) any {
		generatorCalls.Add(1)
		return "val-" + k
	})

	for i := 0; i < 10; i++ {
		buf.GetOrInsert(fmt.Sprintf("key-%d", i))
	}

	assert.True(t, generatorCalls.Load() >= 10)
}
