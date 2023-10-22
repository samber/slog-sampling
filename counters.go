package slogsampling

import (
	"sync/atomic"
	"time"

	"github.com/samber/lo"
)

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

func (c *counter) Inc(tick time.Duration) uint64 {
	// i prefer not using record.Time, because only the sampling middleware time is relevant
	tn := time.Now().UnixNano()
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

func newCounterWithMemory() *counterWithMemory {
	c := &counterWithMemory{
		resetAtAndPreviousCounter: atomic.Pointer[lo.Tuple2[int64, uint64]]{},
		counter:                   atomic.Uint64{},
	}
	c.resetAtAndPreviousCounter.Store(lo.ToPtr(lo.T2(int64(0), uint64(0))))
	return c
}

type counterWithMemory struct {
	resetAtAndPreviousCounter atomic.Pointer[lo.Tuple2[int64, uint64]] // it would be more memory-efficient with a dedicated struct, but i'm lazy
	counter                   atomic.Uint64
}

func (c *counterWithMemory) Inc(tick time.Duration) (n uint64, previousCycle uint64) {
	// i prefer not using record.Time, because only the sampling middleware time is relevant
	tn := time.Now().UnixNano()
	resetAtAndPreviousCounter := c.resetAtAndPreviousCounter.Load()
	if resetAtAndPreviousCounter.A > tn {
		return c.counter.Add(1), resetAtAndPreviousCounter.B
	}

	old := c.counter.Swap(1)

	newResetAfter := lo.T2(tn+tick.Nanoseconds(), old)
	if !c.resetAtAndPreviousCounter.CompareAndSwap(resetAtAndPreviousCounter, lo.ToPtr(newResetAfter)) {
		// We raced with another goroutine trying to reset, and it also reset
		// the counter to 1, so we need to reincrement the counter.
		return c.counter.Add(1), resetAtAndPreviousCounter.B // we should load again instead of returning this outdated value, but it's not a big deal
	}

	return 1, resetAtAndPreviousCounter.B
}
