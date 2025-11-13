package progress

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/pterm/pterm"

	"github.com/dhcgn/mbox-to-imap/stats"
)

// Bar manages a progress bar for tracking message processing.
type Bar struct {
	pb             *pterm.ProgressbarPrinter
	total          int
	alreadyDone    int
	currentScanned int
	mu             sync.Mutex
	enabled        bool
}

// New creates a new progress bar if logLevel is "info".
func New(total int, alreadyDone int, logLevel string) *Bar {
	enabled := logLevel == "info"

	bar := &Bar{
		total:       total,
		alreadyDone: alreadyDone,
		enabled:     enabled,
	}

	if enabled {
		// Create progress bar with total steps
		pb, _ := pterm.DefaultProgressbar.
			WithTotal(total).
			WithTitle("Processing messages").
			Start()

		bar.pb = pb

		// Show initial status
		pterm.Info.Printf("Total messages in mbox: %d\n", total)
		pterm.Info.Printf("Already processed: %d\n", alreadyDone)
		pterm.Info.Printf("Remaining to process: %d\n", total-alreadyDone)
		pterm.Println()

		// Set initial progress to already done count
		pb.Current = alreadyDone
	}

	return bar
}

// Update increments the progress bar based on the event type.
func (b *Bar) Update(evt stats.Event) {
	if !b.enabled || b.pb == nil {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	switch evt.Type {
	case stats.EventTypeScanned:
		b.currentScanned++
		// Update progress for each scanned message
		b.pb.Increment()

		// Update title with current message ID (truncated)
		if evt.MessageID != "" {
			displayID := evt.MessageID
			if len(displayID) > 40 {
				displayID = displayID[:37] + "..."
			}
			b.pb.UpdateTitle("Processing: " + displayID)
		}
	case stats.EventTypeUploaded, stats.EventTypeDryRunUpload:
		// Don't print individual success messages - let progress bar handle it
		// This keeps the output clean
	case stats.EventTypeDuplicate:
		// Don't print individual duplicate messages - let progress bar handle it
		// The final stats will show total duplicates
	case stats.EventTypeError:
		// Show error messages above the progress bar
		if evt.Err != nil {
			pterm.Error.Printf("Error: %v\n", evt.Err)
		}
	}
}

// Stop finalizes the progress bar.
func (b *Bar) Stop() {
	if !b.enabled || b.pb == nil {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// Ensure we reach 100%
	if b.pb.Current < b.total {
		b.pb.Current = b.total
	}

	b.pb.Stop()
	pterm.Success.Println("Processing complete!")
}

// Subscriber creates a stats subscriber function that updates the progress bar.
func (b *Bar) Subscriber(ctx context.Context, events <-chan stats.Event) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case evt, ok := <-events:
			if !ok {
				return nil
			}
			b.Update(evt)
		}
	}
}

// ProgressReporter wraps the stats Reporter with progress bar functionality.
type ProgressReporter struct {
	bar       *Bar
	collector *stats.Collector
	logger    *slog.Logger
	started   time.Time
}

// NewProgressReporter creates a new progress reporter with optional progress bar.
func NewProgressReporter(stream stats.EventStream, bar *Bar, logger *slog.Logger) *ProgressReporter {
	reporter := &ProgressReporter{
		bar:       bar,
		collector: stats.NewCollector(),
		logger:    logger,
		started:   time.Now(),
	}

	if bar != nil && bar.enabled {
		// Subscribe both the progress bar and the stats collector
		stream.SubscribeStats("progress-bar", bar.Subscriber)
		stream.SubscribeStats("progress-stats", reporter.collectStats)
	}

	return reporter
}

// collectStats collects statistics and prints final summary.
func (pr *ProgressReporter) collectStats(ctx context.Context, events <-chan stats.Event) error {
	pr.collector.Run(ctx, events)

	// Print final summary after progress bar stops
	summary := pr.collector.Snapshot()
	duration := time.Since(pr.started)

	if pr.logger != nil {
		// Print summary using pterm for nice formatting
		pterm.Println()
		pterm.DefaultSection.Println("Summary Statistics")
		pterm.Info.Printf("Duration: %v\n", duration)
		pterm.Info.Printf("Scanned: %d\n", summary.Scanned)
		pterm.Info.Printf("Enqueued: %d\n", summary.Enqueued)
		pterm.Info.Printf("Uploaded: %d\n", summary.Uploaded)
		pterm.Info.Printf("Dry-run uploaded: %d\n", summary.DryRunUploaded)
		pterm.Info.Printf("Duplicates (skipped): %d\n", summary.Duplicates)
		pterm.Info.Printf("Errors: %d\n", summary.Errors)
		if summary.LastError != nil {
			pterm.Error.Printf("Last error: %v\n", summary.LastError)
		}
	}

	return nil
}

// CountProgress manages a progress bar for counting messages in mbox file.
type CountProgress struct {
	pb      *pterm.ProgressbarPrinter
	mu      sync.Mutex
	started time.Time
}

// NewCountProgress creates a progress bar for message counting using file size.
func NewCountProgress() *CountProgress {
	cp := &CountProgress{
		started: time.Now(),
	}

	pb, _ := pterm.DefaultProgressbar.
		WithTotal(100).
		WithTitle("Counting messages in mbox file").
		Start()

	cp.pb = pb
	return cp
}

// Update is called with bytes read and total size during counting.
func (cp *CountProgress) Update(bytesRead, totalSize int64) {
	if cp.pb == nil {
		return
	}

	cp.mu.Lock()
	defer cp.mu.Unlock()

	// Calculate percentage
	if totalSize > 0 {
		percentage := int((bytesRead * 100) / totalSize)
		if percentage > 100 {
			percentage = 100
		}
		cp.pb.Current = percentage

		// Update title with MB read and percentage
		mbRead := float64(bytesRead) / 1024 / 1024
		mbTotal := float64(totalSize) / 1024 / 1024
		cp.pb.UpdateTitle(fmt.Sprintf("Counting messages: %.1f/%.1f MB (%d%%)", mbRead, mbTotal, percentage))
	}
}

// Stop finalizes the counting progress bar.
func (cp *CountProgress) Stop() {
	if cp.pb == nil {
		return
	}

	cp.mu.Lock()
	defer cp.mu.Unlock()

	duration := time.Since(cp.started)
	cp.pb.Current = 100
	cp.pb.Stop()
	pterm.Success.Printf("Message counting complete in %v\n", duration)
}
