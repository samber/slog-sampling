package slogsampling

import (
	"bytes"
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"

	slogmulti "github.com/samber/slog-multi"
	"github.com/stretchr/testify/assert"
)

// This test previously failed with the race detector
func TestUniformRace(t *testing.T) {
	const numGoroutines = 100

	buf := &bytes.Buffer{}
	textLogHandler := slog.NewTextHandler(buf, nil)
	sampleMiddleware := UniformSamplingOption{Rate: 0.2}.NewMiddleware()
	sampledLogger := slog.New(slogmulti.Pipe(sampleMiddleware).Handler(textLogHandler))

	wg := &sync.WaitGroup{}
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		goroutineIndex := i
		go func() {
			defer wg.Done()
			sampledLogger.Info("mesage from goroutine", "goroutineIndex", goroutineIndex)
		}()
	}
	wg.Wait()

	// numLines should be in exclusive range (0, numGoroutines)
	// this is probabilistic so it might fail but is pretty unlikely
	numLines := bytes.Count(buf.Bytes(), []byte("\n"))
	if 0 >= numLines || numLines >= numGoroutines {
		t.Errorf("numLines=%d; should be in exclusive range (0, %d)", numLines, numGoroutines)
		t.Error("raw output:")
		t.Error(buf.String())
	}
}

func TestUniformSampling_Rate0(t *testing.T) {
	var buf bytes.Buffer

	logger := slog.New(
		slogmulti.
			Pipe(UniformSamplingOption{Rate: 0.0}.NewMiddleware()).
			Handler(slog.NewTextHandler(&buf, nil)),
	)

	for i := 0; i < 100; i++ {
		logger.Info("test message")
	}

	numLines := bytes.Count(buf.Bytes(), []byte("\n"))
	assert.Equal(t, 0, numLines)
}

func TestUniformSampling_Rate1(t *testing.T) {
	var buf bytes.Buffer

	logger := slog.New(
		slogmulti.
			Pipe(UniformSamplingOption{Rate: 1.0}.NewMiddleware()).
			Handler(slog.NewTextHandler(&buf, nil)),
	)

	const total = 100
	for i := 0; i < total; i++ {
		logger.Info("test message")
	}

	numLines := bytes.Count(buf.Bytes(), []byte("\n"))
	assert.Equal(t, total, numLines)
}

func TestUniformSampling_Probabilistic(t *testing.T) {
	var accepted atomic.Int64

	logger := slog.New(
		slogmulti.
			Pipe(UniformSamplingOption{
				Rate: 0.5,
				OnAccepted: func(_ context.Context, _ slog.Record) {
					accepted.Add(1)
				},
			}.NewMiddleware()).
			Handler(slog.NewTextHandler(&bytes.Buffer{}, nil)),
	)

	const total = 10000
	for i := 0; i < total; i++ {
		logger.Info("test message")
	}

	got := accepted.Load()
	assert.True(t, got > 4000 && got < 6000,
		"accepted=%d, expected ~5000 (50%% of %d)", got, total)
}

func TestUniformSampling_Hooks(t *testing.T) {
	var accepted, dropped atomic.Int64

	logger := slog.New(
		slogmulti.
			Pipe(UniformSamplingOption{
				Rate: 0.5,
				OnAccepted: func(_ context.Context, _ slog.Record) {
					accepted.Add(1)
				},
				OnDropped: func(_ context.Context, _ slog.Record) {
					dropped.Add(1)
				},
			}.NewMiddleware()).
			Handler(slog.NewTextHandler(&bytes.Buffer{}, nil)),
	)

	const total = 1000
	for i := 0; i < total; i++ {
		logger.Info("test message")
	}

	a := accepted.Load()
	d := dropped.Load()
	assert.Equal(t, int64(total), a+d,
		"accepted(%d) + dropped(%d) should equal total(%d)", a, d, total)
}

func TestUniformSampling_PanicOnInvalidRate(t *testing.T) {
	assert.Panics(t, func() {
		UniformSamplingOption{Rate: -0.1}.NewMiddleware()
	})
	assert.Panics(t, func() {
		UniformSamplingOption{Rate: 1.1}.NewMiddleware()
	})
}

func TestUniformSampling_HighTrafficConcurrent(t *testing.T) {
	const numGoroutines = 100
	const logsPerGoroutine = 100

	var accepted, dropped atomic.Int64

	logger := slog.New(
		slogmulti.
			Pipe(UniformSamplingOption{
				Rate: 0.3,
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

	// Check rate is roughly 30%
	rate := float64(accepted.Load()) / float64(total)
	assert.True(t, rate > 0.2 && rate < 0.4,
		"rate=%.2f, expected ~0.3", rate)
}
