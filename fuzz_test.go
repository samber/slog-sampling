package slogsampling

import (
	"bytes"
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	slogmulti "github.com/samber/slog-multi"
	"github.com/stretchr/testify/assert"
)

func FuzzUniformSampling(f *testing.F) {
	f.Add(0.0, 50, "hello world")
	f.Add(0.5, 100, "test message")
	f.Add(1.0, 200, "another message")
	f.Add(0.001, 10, "")
	f.Add(0.999, 500, "fuzz")

	f.Fuzz(func(t *testing.T, rate float64, count int, message string) {
		if rate < 0.0 || rate > 1.0 || count < 1 || count > 1000 {
			t.Skip()
		}

		var accepted, dropped atomic.Int64

		logger := slog.New(
			slogmulti.
				Pipe(UniformSamplingOption{
					Rate: rate,
					OnAccepted: func(_ context.Context, _ slog.Record) {
						accepted.Add(1)
					},
					OnDropped: func(_ context.Context, _ slog.Record) {
						dropped.Add(1)
					},
				}.NewMiddleware()).
				Handler(slog.NewTextHandler(&bytes.Buffer{}, nil)),
		)

		for i := 0; i < count; i++ {
			logger.Info(message)
		}

		total := accepted.Load() + dropped.Load()
		assert.Equal(t, int64(count), total,
			"accepted(%d) + dropped(%d) != total(%d)", accepted.Load(), dropped.Load(), count)
	})
}

func FuzzThresholdSampling(f *testing.F) {
	f.Add(uint64(5), 0.0, 50, "hello world")
	f.Add(uint64(1), 0.5, 100, "test")
	f.Add(uint64(100), 1.0, 200, "message")
	f.Add(uint64(10), 0.3, 500, "fuzz msg")

	f.Fuzz(func(t *testing.T, threshold uint64, rate float64, count int, message string) {
		if rate < 0.0 || rate > 1.0 || count < 1 || count > 1000 || threshold < 1 || threshold > 500 {
			t.Skip()
		}

		var accepted, dropped atomic.Int64

		logger := slog.New(
			slogmulti.
				Pipe(ThresholdSamplingOption{
					Tick:      5 * time.Second,
					Threshold: threshold,
					Rate:      rate,
					OnAccepted: func(_ context.Context, _ slog.Record) {
						accepted.Add(1)
					},
					OnDropped: func(_ context.Context, _ slog.Record) {
						dropped.Add(1)
					},
				}.NewMiddleware()).
				Handler(slog.NewTextHandler(&bytes.Buffer{}, nil)),
		)

		for i := 0; i < count; i++ {
			logger.Info(message)
		}

		total := accepted.Load() + dropped.Load()
		assert.Equal(t, int64(count), total,
			"accepted(%d) + dropped(%d) != total(%d)", accepted.Load(), dropped.Load(), count)

		// At least min(threshold, count) should be accepted
		minExpected := int64(threshold)
		if int64(count) < minExpected {
			minExpected = int64(count)
		}
		assert.True(t, accepted.Load() >= minExpected,
			"accepted=%d < threshold=%d", accepted.Load(), minExpected)
	})
}

func FuzzAbsoluteSampling(f *testing.F) {
	f.Add(uint64(5), 50, "hello")
	f.Add(uint64(10), 100, "test")
	f.Add(uint64(100), 200, "message")

	f.Fuzz(func(t *testing.T, max uint64, count int, message string) {
		if max < 1 || max > 500 || count < 1 || count > 1000 {
			t.Skip()
		}

		var accepted, dropped atomic.Int64

		logger := slog.New(
			slogmulti.
				Pipe(AbsoluteSamplingOption{
					Tick: 5 * time.Second,
					Max:  max,
					OnAccepted: func(_ context.Context, _ slog.Record) {
						accepted.Add(1)
					},
					OnDropped: func(_ context.Context, _ slog.Record) {
						dropped.Add(1)
					},
				}.NewMiddleware()).
				Handler(slog.NewTextHandler(&bytes.Buffer{}, nil)),
		)

		for i := 0; i < count; i++ {
			logger.Info(message)
		}

		total := accepted.Load() + dropped.Load()
		assert.Equal(t, int64(count), total,
			"accepted(%d) + dropped(%d) != total(%d)", accepted.Load(), dropped.Load(), count)
	})
}

func FuzzCustomSampling(f *testing.F) {
	f.Add(0.0, 50, "hello")
	f.Add(0.5, 100, "test")
	f.Add(1.0, 200, "message")

	f.Fuzz(func(t *testing.T, rate float64, count int, message string) {
		if rate < 0.0 || rate > 1.0 || count < 1 || count > 1000 {
			t.Skip()
		}

		var accepted, dropped atomic.Int64

		logger := slog.New(
			slogmulti.
				Pipe(CustomSamplingOption{
					Sampler: func(_ context.Context, _ slog.Record) float64 {
						return rate
					},
					OnAccepted: func(_ context.Context, _ slog.Record) {
						accepted.Add(1)
					},
					OnDropped: func(_ context.Context, _ slog.Record) {
						dropped.Add(1)
					},
				}.NewMiddleware()).
				Handler(slog.NewTextHandler(&bytes.Buffer{}, nil)),
		)

		for i := 0; i < count; i++ {
			logger.Info(message)
		}

		total := accepted.Load() + dropped.Load()
		assert.Equal(t, int64(count), total,
			"accepted(%d) + dropped(%d) != total(%d)", accepted.Load(), dropped.Load(), count)
	})
}

// FuzzThresholdSamplingConcurrent tests threshold sampling under concurrent fuzzed load.
func FuzzThresholdSamplingConcurrent(f *testing.F) {
	f.Add(uint64(5), 0.3, 10, 20, "msg")
	f.Add(uint64(10), 0.5, 50, 10, "test")
	f.Add(uint64(1), 1.0, 100, 5, "hello")

	f.Fuzz(func(t *testing.T, threshold uint64, rate float64, numGoroutines int, logsPerGoroutine int, message string) {
		if rate < 0.0 || rate > 1.0 || threshold < 1 || threshold > 100 ||
			numGoroutines < 1 || numGoroutines > 100 || logsPerGoroutine < 1 || logsPerGoroutine > 100 {
			t.Skip()
		}

		var accepted, dropped atomic.Int64

		logger := slog.New(
			slogmulti.
				Pipe(ThresholdSamplingOption{
					Tick:      5 * time.Second,
					Threshold: threshold,
					Rate:      rate,
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
					logger.Info(message)
				}
			}()
		}
		wg.Wait()

		total := accepted.Load() + dropped.Load()
		assert.Equal(t, int64(numGoroutines*logsPerGoroutine), total)
	})
}

func FuzzAbsoluteSamplingConcurrent(f *testing.F) {
	f.Add(uint64(5), 10, 20, "msg")
	f.Add(uint64(50), 50, 10, "test")
	f.Add(uint64(1), 100, 5, "hello")

	f.Fuzz(func(t *testing.T, max uint64, numGoroutines int, logsPerGoroutine int, message string) {
		if max < 1 || max > 100 ||
			numGoroutines < 1 || numGoroutines > 100 || logsPerGoroutine < 1 || logsPerGoroutine > 100 {
			t.Skip()
		}

		var accepted, dropped atomic.Int64

		logger := slog.New(
			slogmulti.
				Pipe(AbsoluteSamplingOption{
					Tick: 5 * time.Second,
					Max:  max,
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
					logger.Info(message)
				}
			}()
		}
		wg.Wait()

		total := accepted.Load() + dropped.Load()
		assert.Equal(t, int64(numGoroutines*logsPerGoroutine), total)
	})
}

func FuzzUniformSamplingConcurrent(f *testing.F) {
	f.Add(0.3, 10, 20, "msg")
	f.Add(0.5, 50, 10, "test")
	f.Add(1.0, 100, 5, "hello")

	f.Fuzz(func(t *testing.T, rate float64, numGoroutines int, logsPerGoroutine int, message string) {
		if rate < 0.0 || rate > 1.0 ||
			numGoroutines < 1 || numGoroutines > 100 || logsPerGoroutine < 1 || logsPerGoroutine > 100 {
			t.Skip()
		}

		var accepted, dropped atomic.Int64

		logger := slog.New(
			slogmulti.
				Pipe(UniformSamplingOption{
					Rate: rate,
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
					logger.Info(message)
				}
			}()
		}
		wg.Wait()

		total := accepted.Load() + dropped.Load()
		assert.Equal(t, int64(numGoroutines*logsPerGoroutine), total)
	})
}

func FuzzCustomSamplingConcurrent(f *testing.F) {
	f.Add(0.3, 10, 20, "msg")
	f.Add(0.5, 50, 10, "test")
	f.Add(1.0, 100, 5, "hello")

	f.Fuzz(func(t *testing.T, rate float64, numGoroutines int, logsPerGoroutine int, message string) {
		if rate < 0.0 || rate > 1.0 ||
			numGoroutines < 1 || numGoroutines > 100 || logsPerGoroutine < 1 || logsPerGoroutine > 100 {
			t.Skip()
		}

		var accepted, dropped atomic.Int64

		logger := slog.New(
			slogmulti.
				Pipe(CustomSamplingOption{
					Sampler: func(_ context.Context, _ slog.Record) float64 {
						return rate
					},
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
					logger.Info(message)
				}
			}()
		}
		wg.Wait()

		total := accepted.Load() + dropped.Load()
		assert.Equal(t, int64(numGoroutines*logsPerGoroutine), total)
	})
}

// FuzzCounterWindowBoundary tests counter CAS races with adversarial tick durations.
func FuzzCounterWindowBoundary(f *testing.F) {
	f.Add(int64(1000), 10, 50)      // 1µs tick
	f.Add(int64(1000000), 50, 20)   // 1ms tick
	f.Add(int64(100000000), 100, 5) // 100ms tick

	f.Fuzz(func(t *testing.T, tickNanos int64, numGoroutines int, incsPerGoroutine int) {
		if tickNanos < 1 || tickNanos > 1000000000 ||
			numGoroutines < 1 || numGoroutines > 100 || incsPerGoroutine < 1 || incsPerGoroutine > 100 {
			t.Skip()
		}

		tick := time.Duration(tickNanos) * time.Nanosecond
		c := newCounter()

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
	})
}
