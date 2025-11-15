package filter

import (
	"testing"
)

func TestFilter_Allows_IncludeMode(t *testing.T) {
	opts := Options{
		IncludeHeader: []string{"Subject: Test"},
	}
	f, err := New(opts)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	header := []byte("Subject: Test Message\nFrom: sender@example.com\n")
	body := []byte("This is the message body")

	if !f.Allows(header, body) {
		t.Error("Expected message to be allowed (header matches)")
	}

	headerNoMatch := []byte("Subject: Other\nFrom: sender@example.com\n")
	if f.Allows(headerNoMatch, body) {
		t.Error("Expected message to be filtered out (header doesn't match)")
	}
}

func TestFilter_Allows_ExcludeMode(t *testing.T) {
	opts := Options{
		ExcludeHeader: []string{"spam"},
	}
	f, err := New(opts)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	header := []byte("Subject: Normal Message\nFrom: sender@example.com\n")
	body := []byte("This is the message body")

	if !f.Allows(header, body) {
		t.Error("Expected message to be allowed (no spam)")
	}

	headerSpam := []byte("Subject: This is spam\nFrom: spammer@example.com\n")
	if f.Allows(headerSpam, body) {
		t.Error("Expected message to be filtered out (contains spam)")
	}
}

func TestFilter_MutuallyExclusive(t *testing.T) {
	opts := Options{
		IncludeHeader: []string{"test"},
		ExcludeHeader: []string{"spam"},
	}
	_, err := New(opts)
	if err == nil {
		t.Error("Expected error when both include and exclude are specified")
	}
}

func TestFilter_NoFilters(t *testing.T) {
	opts := Options{}
	f, err := New(opts)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	header := []byte("Subject: Any Message\n")
	body := []byte("Any body content")

	if !f.Allows(header, body) {
		t.Error("Expected message to be allowed when no filters are active")
	}
}

func TestFilter_BodyFiltering(t *testing.T) {
	opts := Options{
		IncludeBody: []string{"important"},
	}
	f, err := New(opts)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	header := []byte("Subject: Message\n")
	bodyMatch := []byte("This is an important message")
	bodyNoMatch := []byte("This is a regular message")

	if !f.Allows(header, bodyMatch) {
		t.Error("Expected message to be allowed (body matches)")
	}

	if f.Allows(header, bodyNoMatch) {
		t.Error("Expected message to be filtered out (body doesn't match)")
	}
}

func TestSplitRawMessage(t *testing.T) {
	tests := []struct {
		name       string
		raw        []byte
		wantHeader []byte
		wantBody   []byte
	}{
		{
			name:       "CRLF separator",
			raw:        []byte("Header: value\r\n\r\nBody content"),
			wantHeader: []byte("Header: value"),
			wantBody:   []byte("Body content"),
		},
		{
			name:       "LF separator",
			raw:        []byte("Header: value\n\nBody content"),
			wantHeader: []byte("Header: value"),
			wantBody:   []byte("Body content"),
		},
		{
			name:       "No separator",
			raw:        []byte("All header content"),
			wantHeader: []byte("All header content"),
			wantBody:   nil,
		},
		{
			name:       "Empty message",
			raw:        []byte{},
			wantHeader: nil,
			wantBody:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotHeader, gotBody := SplitRawMessage(tt.raw)
			if string(gotHeader) != string(tt.wantHeader) {
				t.Errorf("SplitRawMessage() header = %q, want %q", gotHeader, tt.wantHeader)
			}
			if string(gotBody) != string(tt.wantBody) {
				t.Errorf("SplitRawMessage() body = %q, want %q", gotBody, tt.wantBody)
			}
		})
	}
}
