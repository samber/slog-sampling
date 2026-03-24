package slogsampling

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	slogmulti "github.com/samber/slog-multi"
)

func BenchmarkBaseline(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", "key", "value")
	}
}

func BenchmarkUniformSampling(b *testing.B) {
	logger := slog.New(
		slogmulti.
			Pipe(UniformSamplingOption{Rate: 0.5}.NewMiddleware()).
			Handler(slog.NewTextHandler(io.Discard, nil)),
	)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", "key", "value")
	}
}

func BenchmarkThresholdSampling(b *testing.B) {
	logger := slog.New(
		slogmulti.
			Pipe(ThresholdSamplingOption{
				Tick:      5 * time.Second,
				Threshold: 10,
				Rate:      0.5,
			}.NewMiddleware()).
			Handler(slog.NewTextHandler(io.Discard, nil)),
	)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", "key", "value")
	}
}

func BenchmarkThresholdSampling_Rate0(b *testing.B) {
	logger := slog.New(
		slogmulti.
			Pipe(ThresholdSamplingOption{
				Tick:      5 * time.Second,
				Threshold: 10,
				Rate:      0,
			}.NewMiddleware()).
			Handler(slog.NewTextHandler(io.Discard, nil)),
	)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", "key", "value")
	}
}

func BenchmarkThresholdSampling_Rate1(b *testing.B) {
	logger := slog.New(
		slogmulti.
			Pipe(ThresholdSamplingOption{
				Tick:      5 * time.Second,
				Threshold: 10,
				Rate:      1.0,
			}.NewMiddleware()).
			Handler(slog.NewTextHandler(io.Discard, nil)),
	)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", "key", "value")
	}
}

func BenchmarkAbsoluteSampling(b *testing.B) {
	logger := slog.New(
		slogmulti.
			Pipe(AbsoluteSamplingOption{
				Tick: 5 * time.Second,
				Max:  100,
			}.NewMiddleware()).
			Handler(slog.NewTextHandler(io.Discard, nil)),
	)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", "key", "value")
	}
}

func BenchmarkCustomSampling(b *testing.B) {
	logger := slog.New(
		slogmulti.
			Pipe(CustomSamplingOption{
				Sampler: func(_ context.Context, _ slog.Record) float64 {
					return 0.5
				},
			}.NewMiddleware()).
			Handler(slog.NewTextHandler(io.Discard, nil)),
	)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", "key", "value")
	}
}

// Parallel benchmarks — measure concurrent throughput

func BenchmarkUniformSampling_Parallel(b *testing.B) {
	logger := slog.New(
		slogmulti.
			Pipe(UniformSamplingOption{Rate: 0.5}.NewMiddleware()).
			Handler(slog.NewTextHandler(io.Discard, nil)),
	)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("benchmark message", "key", "value")
		}
	})
}

func BenchmarkThresholdSampling_Parallel(b *testing.B) {
	logger := slog.New(
		slogmulti.
			Pipe(ThresholdSamplingOption{
				Tick:      5 * time.Second,
				Threshold: 10,
				Rate:      0.5,
			}.NewMiddleware()).
			Handler(slog.NewTextHandler(io.Discard, nil)),
	)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("benchmark message", "key", "value")
		}
	})
}

func BenchmarkAbsoluteSampling_Parallel(b *testing.B) {
	logger := slog.New(
		slogmulti.
			Pipe(AbsoluteSamplingOption{
				Tick: 5 * time.Second,
				Max:  100,
			}.NewMiddleware()).
			Handler(slog.NewTextHandler(io.Discard, nil)),
	)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("benchmark message", "key", "value")
		}
	})
}

func BenchmarkCustomSampling_Parallel(b *testing.B) {
	logger := slog.New(
		slogmulti.
			Pipe(CustomSamplingOption{
				Sampler: func(_ context.Context, _ slog.Record) float64 {
					return 0.5
				},
			}.NewMiddleware()).
			Handler(slog.NewTextHandler(io.Discard, nil)),
	)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("benchmark message", "key", "value")
		}
	})
}

// Adversarial benchmarks — stress buffer, window boundaries, and hooks

func BenchmarkThresholdSampling_ManyKeys(b *testing.B) {
	logger := slog.New(
		slogmulti.
			Pipe(ThresholdSamplingOption{
				Tick:      5 * time.Second,
				Threshold: 10,
				Rate:      0.5,
			}.NewMiddleware()).
			Handler(slog.NewTextHandler(io.Discard, nil)),
	)
	messages := make([]string, 1000)
	for i := range messages {
		messages[i] = fmt.Sprintf("msg-%d", i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info(messages[i%len(messages)], "key", "value")
	}
}

func BenchmarkThresholdSampling_WindowBoundary_Parallel(b *testing.B) {
	logger := slog.New(
		slogmulti.
			Pipe(ThresholdSamplingOption{
				Tick:      time.Microsecond, // forces constant window resets
				Threshold: 1,
				Rate:      0.5,
			}.NewMiddleware()).
			Handler(slog.NewTextHandler(io.Discard, nil)),
	)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("window boundary", "key", "value")
		}
	})
}

func BenchmarkAbsoluteSampling_ManyKeys(b *testing.B) {
	logger := slog.New(
		slogmulti.
			Pipe(AbsoluteSamplingOption{
				Tick: 5 * time.Second,
				Max:  100,
			}.NewMiddleware()).
			Handler(slog.NewTextHandler(io.Discard, nil)),
	)
	messages := make([]string, 1000)
	for i := range messages {
		messages[i] = fmt.Sprintf("msg-%d", i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info(messages[i%len(messages)], "key", "value")
	}
}

func BenchmarkAbsoluteSampling_WindowBoundary_Parallel(b *testing.B) {
	logger := slog.New(
		slogmulti.
			Pipe(AbsoluteSamplingOption{
				Tick: time.Microsecond,
				Max:  1,
			}.NewMiddleware()).
			Handler(slog.NewTextHandler(io.Discard, nil)),
	)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("window boundary", "key", "value")
		}
	})
}

func BenchmarkThresholdSampling_WithHooks(b *testing.B) {
	logger := slog.New(
		slogmulti.
			Pipe(ThresholdSamplingOption{
				Tick:      5 * time.Second,
				Threshold: 10,
				Rate:      0.5,
				OnAccepted: func(_ context.Context, _ slog.Record) {},
				OnDropped:  func(_ context.Context, _ slog.Record) {},
			}.NewMiddleware()).
			Handler(slog.NewTextHandler(io.Discard, nil)),
	)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", "key", "value")
	}
}

func BenchmarkThresholdSampling_WithDroppedCount(b *testing.B) {
	logger := slog.New(
		slogmulti.
			Pipe(ThresholdSamplingOption{
				Tick:                5 * time.Second,
				Threshold:           10,
				Rate:                0.5,
				IncludeDroppedCount: true,
			}.NewMiddleware()).
			Handler(slog.NewTextHandler(io.Discard, nil)),
	)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", "key", "value")
	}
}
