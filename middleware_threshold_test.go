package slogsampling

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	slogmulti "github.com/samber/slog-multi"
	"github.com/stretchr/testify/assert"
)

func TestThresholdSampling_DroppedCountAttribute(t *testing.T) {
	var buf bytes.Buffer

	sampling := ThresholdSamplingOption{
		Tick:                50 * time.Millisecond,
		Threshold:           2,
		Rate:                0,
		IncludeDroppedCount: true,
	}

	logger := slog.New(
		slogmulti.
			Pipe(sampling.NewMiddleware()).
			Handler(slog.NewJSONHandler(&buf, &slog.HandlerOptions{})),
	)

	// Window 1: log 5 records, first 2 accepted, 3 dropped
	for i := 0; i < 5; i++ {
		logger.Info("test message")
	}

	// Wait for window to expire
	time.Sleep(60 * time.Millisecond)

	// Window 2: first record should have dropped_count=3 from previous window
	buf.Reset()
	logger.Info("test message")

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("failed to parse log output: %v\nbuf: %s", err, buf.String())
	}

	dropped, ok := record["slog_sampling.dropped_count"]
	if !ok {
		t.Fatalf("expected slog_sampling.dropped_count attribute, got: %v", record)
	}

	if dropped.(float64) != 3 {
		t.Errorf("expected dropped_count=3, got %v", dropped)
	}
}

func TestThresholdSampling_NoDroppedCountWhenDisabled(t *testing.T) {
	var buf bytes.Buffer

	sampling := ThresholdSamplingOption{
		Tick:                50 * time.Millisecond,
		Threshold:           2,
		Rate:                0,
		IncludeDroppedCount: false, // default
	}

	logger := slog.New(
		slogmulti.
			Pipe(sampling.NewMiddleware()).
			Handler(slog.NewJSONHandler(&buf, &slog.HandlerOptions{})),
	)

	// Window 1: generate drops
	for i := 0; i < 5; i++ {
		logger.Info("test message")
	}

	time.Sleep(60 * time.Millisecond)

	// Window 2: should NOT have dropped_count
	buf.Reset()
	logger.Info("test message")

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("failed to parse log output: %v", err)
	}

	if _, ok := record["slog_sampling.dropped_count"]; ok {
		t.Error("unexpected slog_sampling.dropped_count attribute when IncludeDroppedCount=false")
	}
}

func TestThresholdSampling_NoDroppedCountOnFirstWindow(t *testing.T) {
	var buf bytes.Buffer

	sampling := ThresholdSamplingOption{
		Tick:                50 * time.Millisecond,
		Threshold:           2,
		Rate:                0,
		IncludeDroppedCount: true,
	}

	logger := slog.New(
		slogmulti.
			Pipe(sampling.NewMiddleware()).
			Handler(slog.NewJSONHandler(&buf, &slog.HandlerOptions{})),
	)

	// First window, first record — no previous drops
	logger.Info("test message")

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("failed to parse log output: %v", err)
	}

	if _, ok := record["slog_sampling.dropped_count"]; ok {
		t.Error("unexpected slog_sampling.dropped_count on first-ever record")
	}
}

func TestThresholdSampling_Basic(t *testing.T) {
	var accepted atomic.Int64

	logger := slog.New(
		slogmulti.
			Pipe(ThresholdSamplingOption{
				Tick:      5 * time.Second,
				Threshold: 5,
				Rate:      0,
				OnAccepted: func(_ context.Context, _ slog.Record) {
					accepted.Add(1)
				},
			}.NewMiddleware()).
			Handler(slog.NewTextHandler(&bytes.Buffer{}, nil)),
	)

	for i := 0; i < 20; i++ {
		logger.Info("test message")
	}

	assert.Equal(t, int64(5), accepted.Load())
}

func TestThresholdSampling_RateAboveThreshold(t *testing.T) {
	var accepted atomic.Int64

	logger := slog.New(
		slogmulti.
			Pipe(ThresholdSamplingOption{
				Tick:      5 * time.Second,
				Threshold: 2,
				Rate:      0.5,
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
	// First 2 always accepted + ~50% of remaining 9998
	expectedMin := int64(4000)
	expectedMax := int64(6000)
	assert.True(t, got > expectedMin && got < expectedMax,
		"accepted=%d, expected in (%d, %d)", got, expectedMin, expectedMax)
}

func TestThresholdSampling_Rate1(t *testing.T) {
	var accepted atomic.Int64

	logger := slog.New(
		slogmulti.
			Pipe(ThresholdSamplingOption{
				Tick:      5 * time.Second,
				Threshold: 2,
				Rate:      1.0,
				OnAccepted: func(_ context.Context, _ slog.Record) {
					accepted.Add(1)
				},
			}.NewMiddleware()).
			Handler(slog.NewTextHandler(&bytes.Buffer{}, nil)),
	)

	const total = 100
	for i := 0; i < total; i++ {
		logger.Info("test message")
	}

	assert.Equal(t, int64(total), accepted.Load())
}

func TestThresholdSampling_Rate0(t *testing.T) {
	var accepted atomic.Int64

	logger := slog.New(
		slogmulti.
			Pipe(ThresholdSamplingOption{
				Tick:      5 * time.Second,
				Threshold: 5,
				Rate:      0,
				OnAccepted: func(_ context.Context, _ slog.Record) {
					accepted.Add(1)
				},
			}.NewMiddleware()).
			Handler(slog.NewTextHandler(&bytes.Buffer{}, nil)),
	)

	for i := 0; i < 100; i++ {
		logger.Info("test message")
	}

	assert.Equal(t, int64(5), accepted.Load())
}

func TestThresholdSampling_MultipleKeys(t *testing.T) {
	var accepted atomic.Int64

	logger := slog.New(
		slogmulti.
			Pipe(ThresholdSamplingOption{
				Tick:      5 * time.Second,
				Threshold: 3,
				Rate:      0,
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

	// Each key gets Threshold=3, so 3×3=9 accepted
	assert.Equal(t, int64(9), accepted.Load())
}

func TestThresholdSampling_WindowReset(t *testing.T) {
	var accepted atomic.Int64

	logger := slog.New(
		slogmulti.
			Pipe(ThresholdSamplingOption{
				Tick:      50 * time.Millisecond,
				Threshold: 3,
				Rate:      0,
				OnAccepted: func(_ context.Context, _ slog.Record) {
					accepted.Add(1)
				},
			}.NewMiddleware()).
			Handler(slog.NewTextHandler(&bytes.Buffer{}, nil)),
	)

	// Window 1
	for i := 0; i < 10; i++ {
		logger.Info("test message")
	}
	assert.Equal(t, int64(3), accepted.Load())

	time.Sleep(60 * time.Millisecond)

	// Window 2: threshold resets
	for i := 0; i < 10; i++ {
		logger.Info("test message")
	}
	assert.Equal(t, int64(6), accepted.Load())
}

func TestThresholdSampling_Hooks(t *testing.T) {
	var accepted, dropped atomic.Int64

	logger := slog.New(
		slogmulti.
			Pipe(ThresholdSamplingOption{
				Tick:      5 * time.Second,
				Threshold: 5,
				Rate:      0,
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
	assert.Equal(t, int64(5), a)
	assert.Equal(t, int64(15), d)
	assert.Equal(t, int64(total), a+d)
}

func TestThresholdSampling_PanicOnInvalidRate(t *testing.T) {
	assert.Panics(t, func() {
		ThresholdSamplingOption{
			Tick:      time.Second,
			Threshold: 1,
			Rate:      -0.1,
		}.NewMiddleware()
	})
	assert.Panics(t, func() {
		ThresholdSamplingOption{
			Tick:      time.Second,
			Threshold: 1,
			Rate:      1.1,
		}.NewMiddleware()
	})
}

func TestThresholdSampling_Race(t *testing.T) {
	const numGoroutines = 500

	var accepted, dropped atomic.Int64

	logger := slog.New(
		slogmulti.
			Pipe(ThresholdSamplingOption{
				Tick:      5 * time.Second,
				Threshold: 10,
				Rate:      0.5,
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
	// At least Threshold should be accepted
	assert.True(t, accepted.Load() >= 10, "accepted=%d, expected at least Threshold(10)", accepted.Load())
}

func TestThresholdSampling_HighTraffic(t *testing.T) {
	var accepted atomic.Int64

	logger := slog.New(
		slogmulti.
			Pipe(ThresholdSamplingOption{
				Tick:      5 * time.Second,
				Threshold: 100,
				Rate:      0,
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

	assert.Equal(t, int64(100), accepted.Load())
}

func TestThresholdSampling_ConcurrentDifferentKeys(t *testing.T) {
	const numGoroutines = 100
	const logsPerGoroutine = 50

	var accepted, dropped atomic.Int64

	logger := slog.New(
		slogmulti.
			Pipe(ThresholdSamplingOption{
				Tick:      5 * time.Second,
				Threshold: 5,
				Rate:      0,
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
		msg := fmt.Sprintf("message-%d", i)
		go func() {
			defer wg.Done()
			for j := 0; j < logsPerGoroutine; j++ {
				logger.Info(msg)
			}
		}()
	}
	wg.Wait()

	total := accepted.Load() + dropped.Load()
	assert.Equal(t, int64(numGoroutines*logsPerGoroutine), total)
	// Each key gets Threshold=5, so at least 100*5=500 accepted
	assert.True(t, accepted.Load() >= int64(numGoroutines*5),
		"accepted=%d, expected at least %d", accepted.Load(), numGoroutines*5)
}

func TestThresholdSampling_HighTrafficConcurrent(t *testing.T) {
	const numGoroutines = 100
	const logsPerGoroutine = 100

	var accepted, dropped atomic.Int64

	logger := slog.New(
		slogmulti.
			Pipe(ThresholdSamplingOption{
				Tick:      5 * time.Second,
				Threshold: 50,
				Rate:      0.2,
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
				logger.Info("high traffic concurrent")
			}
		}()
	}
	wg.Wait()

	total := accepted.Load() + dropped.Load()
	assert.Equal(t, int64(numGoroutines*logsPerGoroutine), total,
		"accepted(%d) + dropped(%d) should equal total(%d)", accepted.Load(), dropped.Load(), numGoroutines*logsPerGoroutine)
}
