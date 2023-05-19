package slogsampling

import (
	"context"
	"time"

	slogmulti "github.com/samber/slog-multi"
	"golang.org/x/exp/slog"
)

type Option struct {
	// This will log the first `First` log entries with the same level and message
	// in a `Tick` interval as-is. Following that, it will allow through
	// every `Thereafter`th log entry with the same level and message in that interval.
	Tick       time.Duration
	First      uint64
	Thereafter uint64

	// Optional hooks
	OnAccepted func(context.Context, slog.Record)
	OnDropped  func(context.Context, slog.Record)
}

// NewSamplingMiddleware returns a slog-multi middleware.
func (o Option) NewSamplingMiddleware() slogmulti.Middleware {
	counters := newCounters()

	return slogmulti.NewInlineMiddleware(
		func(ctx context.Context, level slog.Level, next func(context.Context, slog.Level) bool) bool {
			return next(ctx, level)
		},
		func(ctx context.Context, record slog.Record, next func(context.Context, slog.Record) error) error {
			counter := counters.get(record.Level, record)

			n := counter.Inc(record.Time, o.Tick)
			if n > o.First && (o.Thereafter == 0 || (n-o.First)%o.Thereafter != 0) {
				hook(o.OnDropped, ctx, record)
				return nil
			}

			hook(o.OnAccepted, ctx, record)
			return next(ctx, record)
		},
		func(attrs []slog.Attr, next func([]slog.Attr) slog.Handler) slog.Handler {
			return next(attrs)
		},
		func(name string, next func(string) slog.Handler) slog.Handler {
			return next(name)
		},
	)
}

func hook(hook func(context.Context, slog.Record), ctx context.Context, record slog.Record) {
	if hook != nil {
		hook(ctx, record)
	}
}
