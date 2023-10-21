package main

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"log/slog"

	slogmulti "github.com/samber/slog-multi"
	slogsampling "github.com/samber/slog-sampling"
)

func main() {
	var accepted atomic.Int64
	var dropped atomic.Int64

	option := slogsampling.ThresholdSamplingOption{
		Tick:      5 * time.Second,
		Threshold: 10,
		Rate:      0.1,

		Matcher: func(ctx context.Context, record *slog.Record) string {
			return record.Level.String()
		},

		OnAccepted: func(context.Context, slog.Record) {
			accepted.Add(1)
		},
		OnDropped: func(context.Context, slog.Record) {
			dropped.Add(1)
		},
	}

	// option := slogsampling.CustomSamplingOption{
	// 	Sampler: func(ctx context.Context, record slog.Record) float64 {
	// 		switch record.Level {
	// 		case slog.LevelError:
	// 			return 0.5
	// 		case slog.LevelWarn:
	// 			return 0.2
	// 		default:
	// 			return 0.01
	// 		}
	// 	},
	// 	OnAccepted: func(context.Context, slog.Record) {
	// 		accepted.Add(1)
	// 	},
	// 	OnDropped: func(context.Context, slog.Record) {
	// 		dropped.Add(1)
	// 	},
	// }

	// option := slogsampling.UniformSamplingOption{
	// 	Rate: 0.33,
	// 	OnAccepted: func(context.Context, slog.Record) {
	// 		accepted.Add(1)
	// 	},
	// 	OnDropped: func(context.Context, slog.Record) {
	// 		dropped.Add(1)
	// 	},
	// }

	// option := slogsampling.AbsoluteSamplingOption{
	// 	Tick: 5 * time.Second,
	// 	Max:  10,

	// 	Matcher: func(ctx context.Context, record *slog.Record) string {
	// 		return record.Level.String()
	// 	},

	// 	OnAccepted: func(context.Context, slog.Record) {
	// 		accepted.Add(1)
	// 	},
	// 	OnDropped: func(context.Context, slog.Record) {
	// 		dropped.Add(1)
	// 	},
	// }

	logger := slog.New(
		slogmulti.
			Pipe(option.NewMiddleware()).
			Handler(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{})),
	)

	l := logger.
		With("email", "samuel@acme.org").
		With("environment", "dev").
		With("hello", "world")

	for i := 0; i < 100; i++ {
		l.Error("Message 1")
		l.Error("Message 2")
		l.Info("Message 1")
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Printf("\n\nResults:\n")
	fmt.Printf("Accepted: %d\n", accepted.Load())
	fmt.Printf("Dropped: %d\n", dropped.Load())
}
