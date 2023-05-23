package slogsampling

import (
	"context"
	"math/rand"
	"time"

	slogmulti "github.com/samber/slog-multi"
	"golang.org/x/exp/slog"
)

type UniformSamplingOption struct {
	// The sample rate for sampling traces in the range [0.0, 1.0].
	Rate float64

	// Optional hooks
	OnAccepted func(context.Context, slog.Record)
	OnDropped  func(context.Context, slog.Record)
}

// NewMiddleware returns a slog-multi middleware.
func (o UniformSamplingOption) NewMiddleware() slogmulti.Middleware {
	if o.Rate < 0.0 || o.Rate > 1.0 {
		panic("unexpected Rate: must be between 0.0 and 1.0")
	}

	rand := rand.New(rand.NewSource(time.Now().UnixNano()))

	return slogmulti.NewInlineMiddleware(
		func(ctx context.Context, level slog.Level, next func(context.Context, slog.Level) bool) bool {
			return next(ctx, level)
		},
		func(ctx context.Context, record slog.Record, next func(context.Context, slog.Record) error) error {
			if rand.Float64() >= o.Rate {
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
