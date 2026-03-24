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

func TestCustomSampling_AlwaysAccept(t *testing.T) {
	var buf bytes.Buffer

	logger := slog.New(
		slogmulti.
			Pipe(CustomSamplingOption{
				Sampler: func(_ context.Context, _ slog.Record) float64 {
					return 1.0
				},
			}.NewMiddleware()).
			Handler(slog.NewTextHandler(&buf, nil)),
	)

	const total = 100
	for i := 0; i < total; i++ {
		logger.Info("test message")
	}

	numLines := bytes.Count(buf.Bytes(), []byte("\n"))
	assert.Equal(t, total, numLines)
}

func TestCustomSampling_AlwaysDrop(t *testing.T) {
	var buf bytes.Buffer

	logger := slog.New(
		slogmulti.
			Pipe(CustomSamplingOption{
				Sampler: func(_ context.Context, _ slog.Record) float64 {
					return 0.0
				},
			}.NewMiddleware()).
			Handler(slog.NewTextHandler(&buf, nil)),
	)

	for i := 0; i < 100; i++ {
		logger.Info("test message")
	}

	numLines := bytes.Count(buf.Bytes(), []byte("\n"))
	assert.Equal(t, 0, numLines)
}

func TestCustomSampling_InvalidRate(t *testing.T) {
	var dropped atomic.Int64

	logger := slog.New(
		slogmulti.
			Pipe(CustomSamplingOption{
				Sampler: func(_ context.Context, _ slog.Record) float64 {
					return -1.0 // invalid
				},
				OnDropped: func(_ context.Context, _ slog.Record) {
					dropped.Add(1)
				},
			}.NewMiddleware()).
			Handler(slog.NewTextHandler(&bytes.Buffer{}, nil)),
	)

	logger.Info("test message")
	assert.Equal(t, int64(1), dropped.Load())

	// Also test rate > 1.0
	var dropped2 atomic.Int64
	logger2 := slog.New(
		slogmulti.
			Pipe(CustomSamplingOption{
				Sampler: func(_ context.Context, _ slog.Record) float64 {
					return 1.5
				},
				OnDropped: func(_ context.Context, _ slog.Record) {
					dropped2.Add(1)
				},
			}.NewMiddleware()).
			Handler(slog.NewTextHandler(&bytes.Buffer{}, nil)),
	)

	logger2.Info("test message")
	assert.Equal(t, int64(1), dropped2.Load())
}

func TestCustomSampling_Probabilistic(t *testing.T) {
	var accepted atomic.Int64

	logger := slog.New(
		slogmulti.
			Pipe(CustomSamplingOption{
				Sampler: func(_ context.Context, _ slog.Record) float64 {
					return 0.5
				},
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
	assert.True(t, got > 3000 && got < 7000,
		"accepted=%d, expected ~5000 (50%% of %d)", got, total)
}

func TestCustomSampling_Hooks(t *testing.T) {
	var accepted, dropped atomic.Int64

	logger := slog.New(
		slogmulti.
			Pipe(CustomSamplingOption{
				Sampler: func(_ context.Context, _ slog.Record) float64 {
					return 0.5
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

	const total = 1000
	for i := 0; i < total; i++ {
		logger.Info("test message")
	}

	a := accepted.Load()
	d := dropped.Load()
	assert.Equal(t, int64(total), a+d,
		"accepted(%d) + dropped(%d) should equal total(%d)", a, d, total)
}

func TestCustomSampling_Race(t *testing.T) {
	const numGoroutines = 500

	var accepted, dropped atomic.Int64

	logger := slog.New(
		slogmulti.
			Pipe(CustomSamplingOption{
				Sampler: func(_ context.Context, _ slog.Record) float64 {
					return 0.5
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
			logger.Info("concurrent message")
		}()
	}
	wg.Wait()

	total := accepted.Load() + dropped.Load()
	assert.Equal(t, int64(numGoroutines), total)
}

func TestCustomSampling_LevelBasedRate(t *testing.T) {
	var accepted atomic.Int64

	logger := slog.New(
		slogmulti.
			Pipe(CustomSamplingOption{
				Sampler: func(_ context.Context, r slog.Record) float64 {
					if r.Level >= slog.LevelError {
						return 1.0 // always accept errors
					}
					return 0.0 // drop everything else
				},
				OnAccepted: func(_ context.Context, _ slog.Record) {
					accepted.Add(1)
				},
			}.NewMiddleware()).
			Handler(slog.NewTextHandler(&bytes.Buffer{}, nil)),
	)

	for i := 0; i < 100; i++ {
		logger.Info("info message")
		logger.Error("error message")
	}

	assert.Equal(t, int64(100), accepted.Load())
}
