package slogsampling

import (
	"context"

	"log/slog"
)

func hook(hook func(context.Context, slog.Record), ctx context.Context, record slog.Record) {
	if hook != nil {
		hook(ctx, record)
	}
}
