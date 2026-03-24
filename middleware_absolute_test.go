package slogsampling

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	slogmulti "github.com/samber/slog-multi"
	"github.com/stretchr/testify/assert"
)

func TestAbsoluteSampling_Basic(t *testing.T) {
	var buf bytes.Buffer

	logger := slog.New(
		slogmulti.
			Pipe(AbsoluteSamplingOption{
				Tick: 5 * time.Second,
				Max:  10,
			}.NewMiddleware()).
			Handler(slog.NewTextHandler(&buf, nil)),
	)

	for i := 0; i < 20; i++ {
		logger.Info("test message")
	}

	numLines := bytes.Count(buf.Bytes(), []byte("\n"))
	// First window is deterministic: p=0, so n > Max && p <= Max → drop all above Max
	assert.Equal(t, 10, numLines, "numLines=%d, expected exactly Max(10)", numLines)
}

func TestAbsoluteSampling_UnderMax(t *testing.T) {
	var buf bytes.Buffer

	logger := slog.New(
		slogmulti.
			Pipe(AbsoluteSamplingOption{
				Tick: 5 * time.Second,
				Max:  100,
			}.NewMiddleware()).
			Handler(slog.NewTextHandler(&buf, nil)),
	)

	for i := 0; i < 50; i++ {
		logger.Info("test message")
	}

	numLines := bytes.Count(buf.Bytes(), []byte("\n"))
	// First window with no previous overflow: previous=0 <= Max, so logs above Max get dropped.
	// But under Max, all should be accepted.
	assert.True(t, numLines == 50, "numLines=%d, expected 50", numLines)
}

func TestAbsoluteSampling_AdaptiveRate(t *testing.T) {
	var accepted atomic.Int64

	logger := slog.New(
		slogmulti.
			Pipe(AbsoluteSamplingOption{
				Tick: 100 * time.Millisecond,
				Max:  10,
				OnAccepted: func(_ context.Context, _ slog.Record) {
					accepted.Add(1)
				},
			}.NewMiddleware()).
			Handler(slog.NewTextHandler(&bytes.Buffer{}, nil)),
	)

	// Window 1: send 100 logs → first 10 accepted (n <= Max), rest dropped (n > Max && p <= Max)
	for i := 0; i < 100; i++ {
		logger.Info("test message")
	}

	window1Accepted := accepted.Load()
	// In window 1: previous=0, so n > Max && p(0) <= Max → drop all above Max deterministically
	assert.Equal(t, int64(10), window1Accepted, "window1=%d, expected exactly Max(10)", window1Accepted)

	time.Sleep(200 * time.Millisecond)

	// Window 2: previous=100 > Max=10 → rate limit = Max/previous = 10%
	accepted.Store(0)
	for i := 0; i < 100; i++ {
		logger.Info("test message")
	}

	window2Accepted := accepted.Load()
	// Rate ~ 10%, so expect ~10 accepted out of 100 (with generous bounds)
	assert.True(t, window2Accepted >= 1 && window2Accepted <= 30,
		"window2=%d, expected ~10 (10%% rate)", window2Accepted)
}

func TestAbsoluteSampling_MultipleKeys(t *testing.T) {
	var accepted atomic.Int64

	logger := slog.New(
		slogmulti.
			Pipe(AbsoluteSamplingOption{
				Tick: 5 * time.Second,
				Max:  5,
				OnAccepted: func(_ context.Context, _ slog.Record) {
					accepted.Add(1)
				},
			}.NewMiddleware()).
			Handler(slog.NewTextHandler(&bytes.Buffer{}, nil)),
	)

	// 3 different messages × 10 logs each
	for i := 0; i < 10; i++ {
		logger.Info("message-A")
		logger.Info("message-B")
		logger.Info("message-C")
	}

	got := accepted.Load()
	// First window is deterministic: p=0, so each key accepts exactly Max=5 → 3×5=15
	assert.Equal(t, int64(15), got, "accepted=%d, expected exactly 15 (3 keys × Max 5)", got)
}

func TestAbsoluteSampling_Hooks(t *testing.T) {
	var accepted, dropped atomic.Int64

	logger := slog.New(
		slogmulti.
			Pipe(AbsoluteSamplingOption{
				Tick: 5 * time.Second,
				Max:  5,
				OnAccepted: func(_ context.Context, _ slog.Record) {
					accepted.Add(1)
				},
				OnDropped: func(_ context.Context, _ slog.Record) {
					dropped.Add(1)
				},
			}.NewMiddleware()).
			Handler(slog.NewTextHandler(&bytes.Buffer{}, nil)),
	)

	const total = 20
	for i := 0; i < total; i++ {
		logger.Info("test message")
	}

	a := accepted.Load()
	d := dropped.Load()
	assert.Equal(t, int64(total), a+d, "accepted(%d) + dropped(%d) should equal total(%d)", a, d, total)
}

func TestAbsoluteSampling_Race(t *testing.T) {
	const numGoroutines = 500

	var accepted atomic.Int64

	logger := slog.New(
		slogmulti.
			Pipe(AbsoluteSamplingOption{
				Tick: 5 * time.Second,
				Max:  100,
				OnAccepted: func(_ context.Context, _ slog.Record) {
					accepted.Add(1)
				},
			}.NewMiddleware()).
			Handler(slog.NewTextHandler(&bytes.Buffer{}, nil)),
	)

	wg := sync.WaitGroup{}
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			logger.Info("concurrent message")
		}()
	}
	wg.Wait()

	got := accepted.Load()
	assert.True(t, got >= 1 && got <= numGoroutines, "accepted=%d", got)
}

func TestAbsoluteSampling_HighTraffic(t *testing.T) {
	var accepted atomic.Int64

	logger := slog.New(
		slogmulti.
			Pipe(AbsoluteSamplingOption{
				Tick: 5 * time.Second,
				Max:  100,
				OnAccepted: func(_ context.Context, _ slog.Record) {
					accepted.Add(1)
				},
			}.NewMiddleware()).
			Handler(slog.NewTextHandler(&bytes.Buffer{}, nil)),
	)

	const total = 10000
	for i := 0; i < total; i++ {
		logger.Info("high traffic message")
	}

	got := accepted.Load()
	// First window is deterministic: p=0, n > Max && p <= Max → drop all above Max
	assert.Equal(t, int64(100), got, "accepted=%d, expected exactly Max(100)", got)
}

func TestAbsoluteSampling_HighTrafficConcurrent(t *testing.T) {
	const numGoroutines = 100
	const logsPerGoroutine = 100

	var accepted, dropped atomic.Int64

	logger := slog.New(
		slogmulti.
			Pipe(AbsoluteSamplingOption{
				Tick: 5 * time.Second,
				Max:  200,
				OnAccepted: func(_ context.Context, _ slog.Record) {
					accepted.Add(1)
				},
				OnDropped: func(_ context.Context, _ slog.Record) {
					dropped.Add(1)
				},
			}.NewMiddleware()).
			Handler(slog.NewTextHandler(&bytes.Buffer{}, nil)),
	)

	wg := sync.WaitGroup{}
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < logsPerGoroutine; j++ {
				logger.Info("concurrent high traffic")
			}
		}()
	}
	wg.Wait()

	total := accepted.Load() + dropped.Load()
	assert.Equal(t, int64(numGoroutines*logsPerGoroutine), total,
		"accepted(%d) + dropped(%d) should equal total(%d)", accepted.Load(), dropped.Load(), numGoroutines*logsPerGoroutine)
}

func TestAbsoluteSampling_ConcurrentDifferentKeys(t *testing.T) {
	const numGoroutines = 50
	const logsPerGoroutine = 50

	var accepted atomic.Int64

	logger := slog.New(
		slogmulti.
			Pipe(AbsoluteSamplingOption{
				Tick: 5 * time.Second,
				Max:  10,
				OnAccepted: func(_ context.Context, _ slog.Record) {
					accepted.Add(1)
				},
			}.NewMiddleware()).
			Handler(slog.NewTextHandler(&bytes.Buffer{}, nil)),
	)

	wg := sync.WaitGroup{}
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		msg := fmt.Sprintf("message-%d", i)
		go func() {
			defer wg.Done()
			for j := 0; j < logsPerGoroutine; j++ {
				logger.Info(msg)
			}
		}()
	}
	wg.Wait()

	got := accepted.Load()
	// Each of 50 keys gets Max=10 in first window → up to 500 accepted
	assert.True(t, got >= int64(numGoroutines) && got <= int64(numGoroutines*logsPerGoroutine),
		"accepted=%d, expected at least %d (one per key)", got, numGoroutines)
}

func TestAbsoluteSampling_PanicOnZeroMax(t *testing.T) {
	assert.Panics(t, func() {
		AbsoluteSamplingOption{
			Tick: time.Second,
			Max:  0,
		}.NewMiddleware()
	})
}
