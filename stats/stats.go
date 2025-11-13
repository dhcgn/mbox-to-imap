package stats

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"
)

type Stage string

const (
	StageMbox Stage = "mbox"
	StageIMAP Stage = "imap"
)

type EventType string

const (
	EventTypeScanned      EventType = "scanned"
	EventTypeEnqueued     EventType = "enqueued"
	EventTypeUploaded     EventType = "uploaded"
	EventTypeDryRunUpload EventType = "dry_run_uploaded"
	EventTypeDuplicate    EventType = "duplicate"
	EventTypeError        EventType = "error"
)

type Event struct {
	Stage     Stage
	Type      EventType
	MessageID string
	Err       error
	Detail    string
}

type Summary struct {
	Scanned        int
	Enqueued       int
	Uploaded       int
	DryRunUploaded int
	Duplicates     int
	Errors         int
	LastError      error
}

func (s Summary) LogAttrs() []any {
	attrs := []any{
		"scanned", s.Scanned,
		"enqueued", s.Enqueued,
		"uploaded", s.Uploaded,
		"dryRunUploaded", s.DryRunUploaded,
		"duplicates", s.Duplicates,
		"errors", s.Errors,
	}
	if s.LastError != nil {
		attrs = append(attrs, "lastError", s.LastError.Error())
	}
	return attrs
}

type Collector struct {
	mu      sync.Mutex
	summary Summary
}

func NewCollector() *Collector {
	return &Collector{}
}

func (c *Collector) Run(ctx context.Context, events <-chan Event) {
	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-events:
			if !ok {
				return
			}
			c.apply(evt)
		}
	}
}

func (c *Collector) Snapshot() Summary {
	c.mu.Lock()
	summary := c.summary
	c.mu.Unlock()
	return summary
}

func (c *Collector) apply(evt Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch evt.Type {
	case EventTypeScanned:
		c.summary.Scanned++
	case EventTypeEnqueued:
		c.summary.Enqueued++
	case EventTypeUploaded:
		c.summary.Uploaded++
	case EventTypeDryRunUpload:
		c.summary.DryRunUploaded++
	case EventTypeDuplicate:
		c.summary.Duplicates++
	case EventTypeError:
		c.summary.Errors++
		if evt.Err != nil {
			c.summary.LastError = evt.Err
		}
	}
}

type EventStream interface {
	SubscribeStats(name string, fn func(context.Context, <-chan Event) error)
}

type Reporter struct {
	collector *Collector
	logger    *slog.Logger
	started   time.Time
}

func NewReporter(stream EventStream, logger *slog.Logger) *Reporter {
	reporter := &Reporter{
		collector: NewCollector(),
		logger:    logger,
		started:   time.Now(),
	}
	stream.SubscribeStats("stats-reporter", reporter.consume)
	return reporter
}

func (r *Reporter) consume(ctx context.Context, events <-chan Event) error {
	r.collector.Run(ctx, events)
	summary := r.collector.Snapshot()
	attrs := append(summary.LogAttrs(), "duration", time.Since(r.started))
	if ctx.Err() != nil {
		if r.logger != nil {
			r.logger.Debug("stats collection stopped", append(attrs, "err", ctx.Err())...)
		}
		return ctx.Err()
	}
	if r.logger != nil {
		r.logger.Info("stats summary", attrs...)
	}
	return nil
}

func (r *Reporter) Summary() Summary {
	return r.collector.Snapshot()
}

// PrettyPrintTop prints the top N most frequent items in a map.
func PrettyPrintTop(m map[string]int, limit int) {
	type pair struct {
		Key   string
		Value int
	}

	var pairs []pair
	for k, v := range m {
		pairs = append(pairs, pair{k, v})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Value > pairs[j].Value
	})

	for i := 0; i < limit && i < len(pairs); i++ {
		fmt.Printf("%d. %s (%d)\n", i+1, pairs[i].Key, pairs[i].Value)
	}
}
