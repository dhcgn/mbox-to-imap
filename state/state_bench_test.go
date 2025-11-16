package state

import (
	"fmt"
	"os"
	"testing"
)

// BenchmarkFileTracker_MarkProcessed benchmarks the state tracker write performance
func BenchmarkFileTracker_MarkProcessed(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "state-bench-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tracker, err := NewFileTracker(tmpDir, true)
	if err != nil {
		b.Fatal(err)
	}
	defer tracker.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hash := fmt.Sprintf("hash-%d", i)
		msgID := fmt.Sprintf("msg-%d", i)
		if err := tracker.MarkProcessed(hash, msgID); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()

	if err := tracker.Close(); err != nil {
		b.Fatal(err)
	}
}

// BenchmarkFileTracker_AlreadyProcessed benchmarks lookup performance
func BenchmarkFileTracker_AlreadyProcessed(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "state-bench-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tracker, err := NewFileTracker(tmpDir, true)
	if err != nil {
		b.Fatal(err)
	}
	defer tracker.Close()

	// Pre-populate with 1000 entries
	for i := 0; i < 1000; i++ {
		hash := fmt.Sprintf("hash-%d", i)
		msgID := fmt.Sprintf("msg-%d", i)
		if err := tracker.MarkProcessed(hash, msgID); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hash := fmt.Sprintf("hash-%d", i%1000)
		_ = tracker.AlreadyProcessed(hash)
	}
}

// BenchmarkFileTracker_Load benchmarks the state file loading performance
func BenchmarkFileTracker_Load(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "state-bench-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create initial tracker and populate with 10000 entries
	tracker, err := NewFileTracker(tmpDir, true)
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < 10000; i++ {
		hash := fmt.Sprintf("hash-%d", i)
		msgID := fmt.Sprintf("msg-%d", i)
		if err := tracker.MarkProcessed(hash, msgID); err != nil {
			b.Fatal(err)
		}
	}

	if err := tracker.Close(); err != nil {
		b.Fatal(err)
	}

	// Now benchmark loading
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker, err := NewFileTracker(tmpDir, false)
		if err != nil {
			b.Fatal(err)
		}
		tracker.Close()
	}
}

// BenchmarkFileTracker_WithFlush benchmarks write performance with periodic flushes
func BenchmarkFileTracker_WithFlush(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "state-bench-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tracker, err := NewFileTracker(tmpDir, true)
	if err != nil {
		b.Fatal(err)
	}
	defer tracker.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hash := fmt.Sprintf("hash-%d", i)
		msgID := fmt.Sprintf("msg-%d", i)
		if err := tracker.MarkProcessed(hash, msgID); err != nil {
			b.Fatal(err)
		}

		// Simulate periodic flush every 100 entries
		if i%100 == 0 {
			if err := tracker.Flush(); err != nil {
				b.Fatal(err)
			}
		}
	}
	b.StopTimer()

	if err := tracker.Close(); err != nil {
		b.Fatal(err)
	}
}

// BenchmarkMemoryTracker_MarkProcessed benchmarks in-memory tracker for comparison
func BenchmarkMemoryTracker_MarkProcessed(b *testing.B) {
	tracker := NewMemoryTracker()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hash := fmt.Sprintf("hash-%d", i)
		msgID := fmt.Sprintf("msg-%d", i)
		if err := tracker.MarkProcessed(hash, msgID); err != nil {
			b.Fatal(err)
		}
	}
}
