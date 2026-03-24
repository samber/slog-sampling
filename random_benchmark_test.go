package slogsampling

import "testing"

func BenchmarkRandomPercentage(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = randomPercentage(1000) //nolint:typecheck
	}
}
