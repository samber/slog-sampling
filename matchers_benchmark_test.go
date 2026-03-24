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

func BenchmarkMatchByContextValue(b *testing.B) {
	type ctxKey string
	key := ctxKey("trace_id")
	matcher := MatchByContextValue(key)
	ctx := context.WithValue(context.Background(), key, "abc-123-def-456")
	r := newBenchRecord()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher(ctx, &r)
	}
}

func BenchmarkCompactionFNV32a(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CompactionFNV32a("INFO@some log message here")
	}
}

func BenchmarkCompactionFNV64a(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CompactionFNV64a("INFO@some log message here")
	}
}

func BenchmarkCompactionFNV128a(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CompactionFNV128a("INFO@some log message here")
	}
}

func BenchmarkAnyToString(b *testing.B) {
	b.Run("string", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			anyToString("hello world")
		}
	})
	b.Run("int64", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			anyToString(int64(42))
		}
	})
	b.Run("float64", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			anyToString(float64(3.14))
		}
	})
	b.Run("bool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			anyToString(true)
		}
	})
	b.Run("nil", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			anyToString(nil)
		}
	})
}
