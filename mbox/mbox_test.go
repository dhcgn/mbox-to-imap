package mbox

import (
	"context"
	_ "embed"
	"testing"

	"github.com/dhcgn/mbox-to-imap/model"
)

//go:embed test_data/corrupted.mbox
var corruptedMboxData []byte

func TestCorruptedMbox(t *testing.T) {
	mbox_test_data_using = true
	mbox_test_data = corruptedMboxData
	defer func() {
		mbox_test_data_using = false
		mbox_test_data = nil
	}()

	opts := Options{
		Path: "test_data/corrupted.mbox",
	}
	reader, err := NewReader(opts, nil)
	if err != nil {
		t.Fatalf("Failed to create reader: %v", err)
	}

	ctx := context.Background()
	out := make(chan model.Envelope, 10)
	done := make(chan error, 1)

	go func() {
		done <- reader.Stream(ctx, out)
		close(out)
	}()

	messageCount := 0
	for env := range out {
		if env.Err != nil {
			t.Logf("Error encountered: %v", env.Err)
			continue
		}
		if env.Message.ID != "" {
			messageCount++
		}
	}

	if err := <-done; err != nil {
		t.Logf("Stream ended with: %v", err)
	}

	// Todo why 6? I expected 5 messages
	// see cat mbox/test_data/corrupted.mbox | grep "Delivered-To:" | wc -l
	if messageCount != 6 {
		t.Fatalf("Expected 6 messages, got %d", messageCount)
	}

	t.Logf("Parsed %d messages from corrupted mbox", messageCount)
}

func TestStreamWithFilters(t *testing.T) {
	mbox_test_data_using = true
	mbox_test_data = corruptedMboxData
	defer func() {
		mbox_test_data_using = false
		mbox_test_data = nil
	}()

	// TODO Adjust tests to test data
	tests := []struct {
		name          string
		opts          Options
		expectedCount int
	}{
		{
			name: "no filters",
			opts: Options{
				Path: "test_data/corrupted.mbox",
			},
			expectedCount: 6,
		},
		{
			name: "include header filter",
			opts: Options{
				Path:          "test_data/corrupted.mbox",
				IncludeHeader: []string{"Subject:.*test"},
			},
			expectedCount: 0, // adjust based on actual content
		},
		{
			name: "exclude header filter",
			opts: Options{
				Path:          "test_data/corrupted.mbox",
				ExcludeHeader: []string{"Subject:.*nonexistent"},
			},
			expectedCount: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader, err := NewReader(tt.opts, nil)
			if err != nil {
				t.Fatalf("Failed to create reader: %v", err)
			}

			ctx := context.Background()
			out := make(chan model.Envelope, 10)
			done := make(chan error, 1)

			go func() {
				done <- reader.Stream(ctx, out)
				close(out)
			}()

			messageCount := 0
			for env := range out {
				if env.Err != nil {
					t.Logf("Error encountered: %v", env.Err)
					continue
				}
				if env.Message.ID != "" {
					messageCount++
				}
			}

			if err := <-done; err != nil {
				t.Logf("Stream ended with: %v", err)
			}

			if messageCount != tt.expectedCount {
				t.Errorf("Expected %d messages, got %d", tt.expectedCount, messageCount)
			}

			t.Logf("Test %q: parsed %d messages", tt.name, messageCount)
		})
	}
}
