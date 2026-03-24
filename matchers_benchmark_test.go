package slogsampling

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

func newBenchRecord() slog.Record {
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "benchmark message", 0)
	r.AddAttrs(slog.String("key", "value"), slog.Int("count", 42))
	return r
}

func BenchmarkMatchAll(b *testing.B) {
	matcher := MatchAll()
	ctx := context.Background()
	r := newBenchRecord()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher(ctx, &r)
	}
}

func BenchmarkMatchByLevel(b *testing.B) {
	matcher := MatchByLevel()
	ctx := context.Background()
	r := newBenchRecord()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher(ctx, &r)
	}
}

func BenchmarkMatchByMessage(b *testing.B) {
	matcher := MatchByMessage()
	ctx := context.Background()
	r := newBenchRecord()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher(ctx, &r)
	}
}

func BenchmarkMatchByLevelAndMessage(b *testing.B) {
	matcher := MatchByLevelAndMessage()
	ctx := context.Background()
	r := newBenchRecord()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher(ctx, &r)
	}
}

func BenchmarkMatchBySource(b *testing.B) {
	matcher := MatchBySource()
	ctx := context.Background()
	r := newBenchRecord()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher(ctx, &r)
	}
}

func BenchmarkMatchByAttribute(b *testing.B) {
	matcher := MatchByAttribute(nil, "key")
	ctx := context.Background()
	r := newBenchRecord()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher(ctx, &r)
	}
}
