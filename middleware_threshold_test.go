package slogsampling

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	slogmulti "github.com/samber/slog-multi"
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
