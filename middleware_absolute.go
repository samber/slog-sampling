package slogsampling

import (
	"context"
	"log/slog"
	"math/rand"
	"time"

	"github.com/cornelk/hashmap"
	slogmulti "github.com/samber/slog-multi"
)

type AbsoluteSamplingOption struct {
	// This will log all entries with the same hash until max is reached,
	// in a `Tick` interval as-is. Following that, it will reduce log rate depending on previous interval.
	Tick time.Duration
	Max  uint64

	// Group similar logs (default: by level and message)
	Matcher Matcher

	// Optional hooks
	OnAccepted func(context.Context, slog.Record)
	OnDropped  func(context.Context, slog.Record)
}

// NewMiddleware returns a slog-multi middleware.
func (o AbsoluteSamplingOption) NewMiddleware() slogmulti.Middleware {
	if o.Max == 0 {
		panic("unexpected Max: must be greater than 0")
	}

	if o.Matcher == nil {
		o.Matcher = DefaultMatcher
	}

	rand := rand.New(rand.NewSource(time.Now().UnixNano()))
	counters := hashmap.New[string, *counterWithMemory]() // @TODO: implement LRU or LFU draining

	return slogmulti.NewInlineMiddleware(
		func(ctx context.Context, level slog.Level, next func(context.Context, slog.Level) bool) bool {
			return next(ctx, level)
		},
		func(ctx context.Context, record slog.Record, next func(context.Context, slog.Record) error) error {
			key := o.Matcher(ctx, &record)

			c, _ := counters.GetOrInsert(key, newCounterWithMemory())

			n, p := c.Inc(o.Tick)

			// 3 cases:
			//   - current interval is over threshold but not previous -> drop
			//   - previous interval is over threshold -> apply rate limit
			//   - none of current and previous intervals are over threshold -> accept

			if (n > o.Max && p <= o.Max) || (p > o.Max && rand.Float64() >= float64(o.Max)/float64(p)) {
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
