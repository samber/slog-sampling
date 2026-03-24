package slogsampling

import (
	"testing"
	"time"
)

func BenchmarkCounter_Inc(b *testing.B) {
	c := newCounter()
	tick := 5 * time.Second
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Inc(tick)
	}
}

func BenchmarkCounter_Inc_Parallel(b *testing.B) {
	c := newCounter()
	tick := 5 * time.Second
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Inc(tick)
		}
	})
}

func BenchmarkCounterWithMemory_Inc(b *testing.B) {
	c := newCounterWithMemory()
	tick := 5 * time.Second
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Inc(tick)
	}
}

func BenchmarkCounterWithMemory_Inc_Parallel(b *testing.B) {
	c := newCounterWithMemory()
	tick := 5 * time.Second
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Inc(tick)
		}
	})
}

// Adversarial: force window reset on every call

func BenchmarkCounter_Inc_WindowReset(b *testing.B) {
	c := newCounter()
	tick := time.Nanosecond // forces reset every call
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Inc(tick)
	}
}

func BenchmarkCounter_Inc_WindowReset_Parallel(b *testing.B) {
	c := newCounter()
	tick := time.Nanosecond
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Inc(tick)
		}
	})
}

func BenchmarkCounterWithMemory_Inc_WindowReset_Parallel(b *testing.B) {
	c := newCounterWithMemory()
	tick := time.Nanosecond
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Inc(tick)
		}
	})
}
