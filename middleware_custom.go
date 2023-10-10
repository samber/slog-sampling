package slogsampling

import (
	"context"

	"log/slog"

	slogmulti "github.com/samber/slog-multi"
)

type CustomSamplingOption struct {
	// The sample rate for sampling traces in the range [0.0, 1.0].
	Sampler func(context.Context, slog.Record) float64

	// Optional hooks
	OnAccepted func(context.Context, slog.Record)
	OnDropped  func(context.Context, slog.Record)
}

// NewMiddleware returns a slog-multi middleware.
func (o CustomSamplingOption) NewMiddleware() slogmulti.Middleware {
	rand := newLockedRNG()

	return slogmulti.NewInlineMiddleware(
		func(ctx context.Context, level slog.Level, next func(context.Context, slog.Level) bool) bool {
			return next(ctx, level)
		},
		func(ctx context.Context, record slog.Record, next func(context.Context, slog.Record) error) error {
			if rand.Float64() >= o.Sampler(ctx, record) {
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
