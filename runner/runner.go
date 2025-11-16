package runner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/dhcgn/mbox-to-imap/config"
	"github.com/dhcgn/mbox-to-imap/model"
	"github.com/dhcgn/mbox-to-imap/state"
	"github.com/dhcgn/mbox-to-imap/stats"
)

var ErrMessageIDMissing = errors.New("mbox message missing id")

type StageFunc func(context.Context) error

type Runner struct {
	cfg    config.Config
	logger *slog.Logger

	ctx    context.Context
	cancel context.CancelFunc

	messages chan model.Envelope
	uploads  chan model.Message
	events   chan stats.Event

	tracker state.Tracker

	workWG  sync.WaitGroup
	statsWG sync.WaitGroup

	errMu sync.Mutex
	err   error

	closeMailboxOnce sync.Once
	closeUploadsOnce sync.Once
	closeEventsOnce  sync.Once
	since            time.Time
}

func New(cfg config.Config, logger *slog.Logger) (*Runner, error) {
	ctx, cancel := context.WithCancel(context.Background())

	tracker, err := state.NewFileTracker(cfg.StateDir, !cfg.DryRun)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("state tracker: %w", err)
	}

	r := &Runner{
		cfg:      cfg,
		logger:   logger,
		ctx:      ctx,
		cancel:   cancel,
		messages: make(chan model.Envelope, 32),
		uploads:  make(chan model.Message, 32),
		events:   make(chan stats.Event, 128),
		tracker:  tracker,
	}

	r.AddStage("bridge", r.bridge)
	return r, nil
}

func (r *Runner) Config() config.Config {
	return r.cfg
}

func (r *Runner) Logger() *slog.Logger {
	return r.logger
}

func (r *Runner) Context() context.Context {
	return r.ctx
}

func (r *Runner) Tracker() state.Tracker {
	return r.tracker
}

func (r *Runner) MailboxWriter() chan<- model.Envelope {
	return r.messages
}

func (r *Runner) CloseMailbox() {
	r.closeMailboxOnce.Do(func() {
		close(r.messages)
	})
}

func (r *Runner) Uploads() <-chan model.Message {
	return r.uploads
}

func (r *Runner) EmitEvent(evt stats.Event) {
	select {
	case <-r.ctx.Done():
	case r.events <- evt:
	}
}

func (r *Runner) SubscribeStats(name string, fn func(context.Context, <-chan stats.Event) error) {
	r.statsWG.Add(1)
	go func() {
		defer r.statsWG.Done()
		if err := fn(r.ctx, r.events); err != nil && !errors.Is(err, context.Canceled) {
			r.fail(fmt.Errorf("%s stats: %w", name, err))
		}
	}()
}

func (r *Runner) AddStage(name string, fn StageFunc) {
	r.workWG.Add(1)
	go func() {
		defer r.workWG.Done()
		if err := fn(r.ctx); err != nil && !errors.Is(err, context.Canceled) {
			r.fail(fmt.Errorf("%s stage: %w", name, err))
		}
	}()
}

func (r *Runner) Start() error {
	r.since = time.Now()

	r.workWG.Wait()
	r.closeEvents()
	r.statsWG.Wait()

	r.cancel()

	err := r.err
	duration := time.Since(r.since)
	if err != nil {
		r.logger.Error("pipeline failed", "duration", duration, "err", err)
		return err
	}

	r.logger.Info("pipeline completed", "duration", duration)
	return nil
}

func (r *Runner) bridge(ctx context.Context) error {
	defer r.closeUploads()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case envelope, ok := <-r.messages:
			if !ok {
				return nil
			}

			if envelope.Err != nil {
				r.EmitEvent(stats.Event{Stage: stats.StageMbox, Type: stats.EventTypeError, Err: envelope.Err})
				r.fail(fmt.Errorf("mbox envelope: %w", envelope.Err))
				continue
			}

			msg := envelope.Message
			r.EmitEvent(stats.Event{Stage: stats.StageMbox, Type: stats.EventTypeScanned, MessageID: msg.ID})

			if msg.ID == "" {
				r.EmitEvent(stats.Event{Stage: stats.StageMbox, Type: stats.EventTypeError, Err: ErrMessageIDMissing})
				r.fail(ErrMessageIDMissing)
				continue
			}

			if msg.Hash != "" && r.tracker.AlreadyProcessed(msg.Hash) {
				r.EmitEvent(stats.Event{Stage: stats.StageMbox, Type: stats.EventTypeDuplicate, MessageID: msg.ID})
				continue
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case r.uploads <- msg:
				r.EmitEvent(stats.Event{Stage: stats.StageMbox, Type: stats.EventTypeEnqueued, MessageID: msg.ID})
			}
		}
	}
}

func (r *Runner) closeUploads() {
	r.closeUploadsOnce.Do(func() {
		close(r.uploads)
	})
}

func (r *Runner) closeEvents() {
	r.closeEventsOnce.Do(func() {
		close(r.events)
	})
}

func (r *Runner) fail(err error) {
	if err == nil {
		return
	}
	r.errMu.Lock()
	if r.err == nil {
		r.err = err
		r.cancel()
	}
	r.errMu.Unlock()
}
