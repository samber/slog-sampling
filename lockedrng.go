package slogsampling

import (
	"math/rand"
	"sync"
	"time"
)

// lockedRNG is safe to be shared by multiple goroutines.
type lockedRNG struct {
	mu  sync.Mutex
	rng *rand.Rand
}

// newLockedRNG returns a new lockedRNG seeded with the current time.
func newLockedRNG() *lockedRNG {
	return &lockedRNG{
		sync.Mutex{},
		rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (l *lockedRNG) Float64() float64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.rng.Float64()
}
