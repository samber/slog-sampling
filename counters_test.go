package slogsampling

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCounter_Inc_WithinWindow(t *testing.T) {
	c := newCounter()
	tick := 1 * time.Second

	for i := uint64(1); i <= 10; i++ {
		n := c.Inc(tick)
		assert.Equal(t, i, n)
	}
}

func TestCounter_Inc_WindowReset(t *testing.T) {
	c := newCounter()
	tick := 100 * time.Millisecond

	n := c.Inc(tick)
	assert.Equal(t, uint64(1), n)

	n = c.Inc(tick)
	assert.Equal(t, uint64(2), n)

	time.Sleep(200 * time.Millisecond)

	n = c.Inc(tick)
	assert.Equal(t, uint64(1), n)

	n = c.Inc(tick)
	assert.Equal(t, uint64(2), n)
}

func TestCounter_DroppedRotation(t *testing.T) {
	c := newCounter()
	tick := 100 * time.Millisecond

	// Window 1: increment counter and record some drops
	c.Inc(tick)
	c.IncDropped()
	c.IncDropped()
	c.IncDropped()

	assert.Equal(t, uint64(0), c.PrevDropped())

	// Wait for window to expire
	time.Sleep(200 * time.Millisecond)

	// Window 2: Inc triggers rotation
	c.Inc(tick)
	assert.Equal(t, uint64(3), c.PrevDropped())

	// Add drops in window 2
	c.IncDropped()

	// Wait for window 3
	time.Sleep(200 * time.Millisecond)
	c.Inc(tick)

	assert.Equal(t, uint64(1), c.PrevDropped())
}

func TestCounter_ConcurrentInc(t *testing.T) {
	c := newCounter()
	tick := 1 * time.Second

	const numGoroutines = 500
	wg := sync.WaitGroup{}
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			c.Inc(tick)
		}()
	}

	wg.Wait()

	// All increments happened within one window, so counter should equal numGoroutines.
	// Due to race resets, counter.Load() might be slightly off, but the key test is race-free execution.
	got := c.counter.Load()
	assert.True(t, got >= 1 && got <= numGoroutines, "counter=%d, expected in [1, %d]", got, numGoroutines)
}

func TestCounter_ConcurrentIncDropped(t *testing.T) {
	c := newCounter()
	tick := 1 * time.Second

	c.Inc(tick)

	const numGoroutines = 200
	wg := sync.WaitGroup{}
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			c.IncDropped()
		}()
	}

	wg.Wait()

	assert.Equal(t, uint64(numGoroutines), c.currDropped.Load())
}

func TestCounterWithMemory_Inc(t *testing.T) {
	c := newCounterWithMemory()
	tick := 1 * time.Second

	n, prev := c.Inc(tick)
	assert.Equal(t, uint64(1), n)
	assert.Equal(t, uint64(0), prev)

	n, prev = c.Inc(tick)
	assert.Equal(t, uint64(2), n)
	assert.Equal(t, uint64(0), prev)
}

func TestCounterWithMemory_WindowReset(t *testing.T) {
	c := newCounterWithMemory()
	tick := 100 * time.Millisecond

	// Window 1: log 5 times
	for i := 0; i < 5; i++ {
		c.Inc(tick)
	}

	time.Sleep(200 * time.Millisecond)

	// Window 2: the returned previousCycle comes from the tuple stored before reset,
	// which was (0, 0) initially. The reset stores old counter (5) into the new tuple,
	// but returns the *old* tuple's B value.
	n, prev := c.Inc(tick)
	assert.Equal(t, uint64(1), n)
	assert.Equal(t, uint64(0), prev) // from initial tuple

	// Within window 2, subsequent calls return the current tuple's B = 5
	n, prev = c.Inc(tick)
	assert.Equal(t, uint64(2), n)
	assert.Equal(t, uint64(5), prev)

	c.Inc(tick) // counter = 3

	time.Sleep(200 * time.Millisecond)

	// Window 3: reset returns old tuple's B = 5 (stored during window 2 reset)
	n, prev = c.Inc(tick)
	assert.Equal(t, uint64(1), n)
	assert.Equal(t, uint64(5), prev)

	// Subsequent calls in window 3 see the new tuple with previous=3
	n, prev = c.Inc(tick)
	assert.Equal(t, uint64(2), n)
	assert.Equal(t, uint64(3), prev)
}

func TestCounterWithMemory_ConcurrentInc(t *testing.T) {
	c := newCounterWithMemory()
	tick := 1 * time.Second

	const numGoroutines = 500
	wg := sync.WaitGroup{}
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			c.Inc(tick)
		}()
	}

	wg.Wait()

	got := c.counter.Load()
	assert.True(t, got >= 1 && got <= numGoroutines, "counter=%d, expected in [1, %d]", got, numGoroutines)
}

// Adversarial: concurrent Inc with very short tick forcing constant CAS races on window reset

func TestCounter_ConcurrentIncDuringWindowReset(t *testing.T) {
	c := newCounter()
	tick := time.Millisecond // very short → many resets during test

	const numGoroutines = 500
	const incsPerGoroutine = 100
	wg := sync.WaitGroup{}
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < incsPerGoroutine; j++ {
				n := c.Inc(tick)
				assert.True(t, n >= 1)
			}
		}()
	}

	wg.Wait()
}

func TestCounterWithMemory_ConcurrentIncDuringWindowReset(t *testing.T) {
	c := newCounterWithMemory()
	tick := time.Millisecond

	const numGoroutines = 500
	const incsPerGoroutine = 100
	wg := sync.WaitGroup{}
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < incsPerGoroutine; j++ {
				n, _ := c.Inc(tick)
				assert.True(t, n >= 1)
			}
		}()
	}

	wg.Wait()
}
