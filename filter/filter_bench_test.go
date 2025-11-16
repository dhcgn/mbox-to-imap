package filter

import (
	"testing"
)

// BenchmarkFilter_Allows_NoFilters benchmarks the filter when no filters are active
func BenchmarkFilter_Allows_NoFilters(b *testing.B) {
	f, err := New(Options{})
	if err != nil {
		b.Fatal(err)
	}

	header := []byte("From: test@example.com\nTo: user@example.com\nSubject: Test\n")
	body := []byte("This is a test message body with some content.")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Allows(header, body)
	}
}

// BenchmarkFilter_Allows_WithIncludeFilter benchmarks the filter with include patterns
func BenchmarkFilter_Allows_WithIncludeFilter(b *testing.B) {
	f, err := New(Options{
		IncludeHeader: []string{"From:.*@example\\.com"},
	})
	if err != nil {
		b.Fatal(err)
	}

	header := []byte("From: test@example.com\nTo: user@example.com\nSubject: Test\n")
	body := []byte("This is a test message body with some content.")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Allows(header, body)
	}
}

// BenchmarkFilter_Allows_WithExcludeFilter benchmarks the filter with exclude patterns
func BenchmarkFilter_Allows_WithExcludeFilter(b *testing.B) {
	f, err := New(Options{
		ExcludeHeader: []string{"From:.*@spam\\.com"},
	})
	if err != nil {
		b.Fatal(err)
	}

	header := []byte("From: test@example.com\nTo: user@example.com\nSubject: Test\n")
	body := []byte("This is a test message body with some content.")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Allows(header, body)
	}
}

// BenchmarkFilter_Allows_MultiplePatterns benchmarks with multiple regex patterns
func BenchmarkFilter_Allows_MultiplePatterns(b *testing.B) {
	f, err := New(Options{
		IncludeHeader: []string{
			"From:.*@example\\.com",
			"Subject:.*Test.*",
			"To:.*user.*",
		},
	})
	if err != nil {
		b.Fatal(err)
	}

	header := []byte("From: test@example.com\nTo: user@example.com\nSubject: Test\n")
	body := []byte("This is a test message body with some content.")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Allows(header, body)
	}
}

// BenchmarkFilter_Allows_BodyFilter benchmarks body filtering
func BenchmarkFilter_Allows_BodyFilter(b *testing.B) {
	f, err := New(Options{
		IncludeBody: []string{"important.*content"},
	})
	if err != nil {
		b.Fatal(err)
	}

	header := []byte("From: test@example.com\nTo: user@example.com\nSubject: Test\n")
	body := []byte("This message contains important content that should match the filter.")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Allows(header, body)
	}
}

// BenchmarkSplitRawMessage benchmarks the raw message splitting function
func BenchmarkSplitRawMessage(b *testing.B) {
	raw := []byte("From: test@example.com\nTo: user@example.com\nSubject: Test\n\r\n\r\nThis is the body of the message.")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SplitRawMessage(raw)
	}
}
