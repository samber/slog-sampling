package main

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	slogmulti "github.com/samber/slog-multi"
	slogsampling "github.com/samber/slog-sampling"
	"golang.org/x/exp/slog"
)

func main() {
	var accepted atomic.Int64
	var dropped atomic.Int64

	option := slogsampling.Option{
		Tick:       5 * time.Second,
		First:      10,
		Thereafter: 10,
		OnAccepted: func(context.Context, slog.Record) {
			accepted.Add(1)
		},
		OnDropped: func(context.Context, slog.Record) {
			dropped.Add(1)
		},
	}

	logger := slog.New(
		slogmulti.
			Pipe(option.NewSamplingMiddleware()).
			Handler(slog.NewJSONHandler(os.Stdout)),
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
