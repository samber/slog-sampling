package slogsampling

import (
	"sync/atomic"
	"time"
)

const countersPerLevel = 4096

func newCounter() *counter {
	return &counter{
		resetAt: atomic.Int64{},
		counter: atomic.Uint64{},
	}
}

type counter struct {
	resetAt atomic.Int64
	counter atomic.Uint64
}

func (c *counter) Inc(t time.Time, tick time.Duration) uint64 {
	tn := t.UnixNano()
	resetAfter := c.resetAt.Load()
	if resetAfter > tn {
		return c.counter.Add(1)
	}

	c.counter.Store(1)

	newResetAfter := tn + tick.Nanoseconds()
	if !c.resetAt.CompareAndSwap(resetAfter, newResetAfter) {
		// We raced with another goroutine trying to reset, and it also reset
		// the counter to 1, so we need to reincrement the counter.
		return c.counter.Add(1)
	}

	return 1
}
