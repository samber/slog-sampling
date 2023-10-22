package slogsampling

import (
	"bytes"
	"log/slog"
	"sync"
	"testing"

	slogmulti "github.com/samber/slog-multi"
)

// This test previously failed with the race detector
func TestUniformRace(t *testing.T) {
	const numGoroutines = 100

	buf := &bytes.Buffer{}
	textLogHandler := slog.NewTextHandler(buf, nil)
	sampleMiddleware := UniformSamplingOption{Rate: 0.2}.NewMiddleware()
	sampledLogger := slog.New(slogmulti.Pipe(sampleMiddleware).Handler(textLogHandler))

	wg := &sync.WaitGroup{}
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		goroutineIndex := i
		go func() {
			defer wg.Done()
			sampledLogger.Info("mesage from goroutine", "goroutineIndex", goroutineIndex)
		}()
	}
	wg.Wait()

	// numLines should be in exclusive range (0, numGoroutines)
	// this is probabilistic so it might fail but is pretty unlikely
	numLines := bytes.Count(buf.Bytes(), []byte("\n"))
	if !(0 < numLines && numLines < numGoroutines) {
		t.Errorf("numLines=%d; should be in exclusive range (0, %d)", numLines, numGoroutines)
		t.Error("raw output:")
		t.Error(buf.String())
	}
}
