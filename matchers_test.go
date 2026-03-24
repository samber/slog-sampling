package slogsampling

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMatchAll(t *testing.T) {
	matcher := MatchAll()
	ctx := context.Background()
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "hello", 0)

	assert.Equal(t, "", matcher(ctx, &r))
}

func TestMatchByLevel(t *testing.T) {
	matcher := MatchByLevel()
	ctx := context.Background()

	for _, level := range []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError} {
		r := slog.NewRecord(time.Now(), level, "msg", 0)
		assert.Equal(t, level.String(), matcher(ctx, &r))
	}
}

func TestMatchByMessage(t *testing.T) {
	matcher := MatchByMessage()
	ctx := context.Background()

	for _, msg := range []string{"hello", "world", "", "with spaces", "special@chars#!"} {
		r := slog.NewRecord(time.Now(), slog.LevelInfo, msg, 0)
		assert.Equal(t, msg, matcher(ctx, &r))
	}
}

func TestMatchByLevelAndMessage(t *testing.T) {
	matcher := MatchByLevelAndMessage()
	ctx := context.Background()

	r := slog.NewRecord(time.Now(), slog.LevelError, "something failed", 0)
	assert.Equal(t, "ERROR@something failed", matcher(ctx, &r))

	r2 := slog.NewRecord(time.Now(), slog.LevelInfo, "something failed", 0)
	assert.Equal(t, "INFO@something failed", matcher(ctx, &r2))

	// Different messages with same level produce different keys
	assert.NotEqual(t, matcher(ctx, &r), matcher(ctx, &r2))
}

func TestMatchBySource(t *testing.T) {
	matcher := MatchBySource()
	ctx := context.Background()

	// Record with PC=0 won't have source info but should not panic
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	result := matcher(ctx, &r)
	assert.NotEmpty(t, result) // format: "file@line@function"
}

func TestMatchByAttribute(t *testing.T) {
	matcher := MatchByAttribute(nil, "request_id")
	ctx := context.Background()

	// Attribute found
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	r.AddAttrs(slog.String("request_id", "abc-123"))
	assert.Equal(t, "abc-123", matcher(ctx, &r))

	// Attribute not found
	r2 := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	r2.AddAttrs(slog.String("other_key", "value"))
	assert.Equal(t, "", matcher(ctx, &r2))

	// No attributes at all
	r3 := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	assert.Equal(t, "", matcher(ctx, &r3))

	// Integer attribute
	matcherInt := MatchByAttribute(nil, "count")
	r4 := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	r4.AddAttrs(slog.Int64("count", 42))
	assert.Equal(t, "42", matcherInt(ctx, &r4))
}

func TestMatchByContextValue(t *testing.T) {
	type ctxKey string
	key := ctxKey("trace_id")
	matcher := MatchByContextValue(key)

	// Value present
	ctx := context.WithValue(context.Background(), key, "trace-abc")
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	assert.Equal(t, "trace-abc", matcher(ctx, &r))

	// Value absent
	ctx2 := context.Background()
	assert.Equal(t, "", matcher(ctx2, &r))
}

func TestAnyToString(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"nil", nil, ""},
		{"string", "hello", "hello"},
		{"bytes", []byte("world"), "world"},
		{"int64", int64(42), "42"},
		{"int64_negative", int64(-1), "-1"},
		{"uint64", uint64(100), "100"},
		{"float64", float64(3.14), "3.14"},
		{"bool_true", true, "true"},
		{"bool_false", false, "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, anyToString(tt.input))
		})
	}
}

func TestCompactionFNV32a(t *testing.T) {
	// Deterministic
	assert.Equal(t, CompactionFNV32a("hello"), CompactionFNV32a("hello"))

	// Different inputs → different outputs
	assert.NotEqual(t, CompactionFNV32a("hello"), CompactionFNV32a("world"))

	// Non-empty
	assert.NotEmpty(t, CompactionFNV32a(""))
}

func TestCompactionFNV64a(t *testing.T) {
	assert.Equal(t, CompactionFNV64a("hello"), CompactionFNV64a("hello"))
	assert.NotEqual(t, CompactionFNV64a("hello"), CompactionFNV64a("world"))
	assert.NotEmpty(t, CompactionFNV64a(""))
}

func TestCompactionFNV128a(t *testing.T) {
	assert.Equal(t, CompactionFNV128a("hello"), CompactionFNV128a("hello"))
	assert.NotEqual(t, CompactionFNV128a("hello"), CompactionFNV128a("world"))
	assert.NotEmpty(t, CompactionFNV128a(""))
}
