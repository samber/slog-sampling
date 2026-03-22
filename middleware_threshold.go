package slogsampling

import (
	"context"
	"time"

	"log/slog"

	slogmulti "github.com/samber/slog-multi"
	"github.com/samber/slog-sampling/buffer"
)

type ThresholdSamplingOption struct {
	// This will log the first `Threshold` log entries with the same hash,
	// in a `Tick` interval as-is. Following that, it will allow `Rate` in the range [0.0, 1.0].
	Tick      time.Duration
	Threshold uint64
	Rate      float64

	// Group similar logs (default: by level and message)
	Matcher Matcher
	Buffer  func(generator func(string) any) buffer.Buffer[string]
	buffer  buffer.Buffer[string]

	// Optional hooks
	OnAccepted func(context.Context, slog.Record)
	OnDropped  func(context.Context, slog.Record)

	// When true, the first accepted record after a suppression window includes
	// a "slog_sampling.dropped_count" attribute with the number of records that
	// were dropped in the previous window. This gives operators visibility into
	// suppression volume without a separate summary goroutine.
	IncludeDroppedCount bool
}

// NewMiddleware returns a slog-multi middleware.
func (o ThresholdSamplingOption) NewMiddleware() slogmulti.Middleware {
	if o.Rate < 0.0 || o.Rate > 1.0 {
		panic("unexpected Rate: must be between 0.0 and 1.0")
	}

	if o.Matcher == nil {
		o.Matcher = DefaultMatcher
	}

	if o.Buffer == nil {
		o.Buffer = buffer.NewUnlimitedBuffer[string]()
	}

	o.buffer = o.Buffer(func(k string) any {
		return newCounter()
	})

	return slogmulti.NewInlineMiddleware(
		func(ctx context.Context, level slog.Level, next func(context.Context, slog.Level) bool) bool {
			return next(ctx, level)
		},
		func(ctx context.Context, record slog.Record, next func(context.Context, slog.Record) error) error {
			key := o.Matcher(ctx, &record)
			v, _ := o.buffer.GetOrInsert(key)
			cnt := v.(*counter)
			n := cnt.Inc(o.Tick)

			if n > o.Threshold {
				// Fast path: skip expensive crypto/rand when Rate is 0 (drop all)
				// or 1 (accept all). Only compute random when probabilistic sampling.
				if o.Rate == 0 {
					cnt.IncDropped()
					hook(o.OnDropped, ctx, record)
					return nil
				}
				if o.Rate < 1.0 {
					random, err := randomPercentage(1000) // 0.001 precision
					if err != nil {
						return err
					}
					if random >= o.Rate {
						cnt.IncDropped()
						hook(o.OnDropped, ctx, record)
						return nil
					}
				}
			}

			// Attach dropped count from previous window to the first accepted
			// record in a new window, so operators see suppression volume inline.
			if o.IncludeDroppedCount && n == 1 {
				if dropped := cnt.PrevDropped(); dropped > 0 {
					record.AddAttrs(slog.Uint64("slog_sampling.dropped_count", dropped))
				}
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
