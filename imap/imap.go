package imap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/dhcgn/mbox-to-imap/model"
	"github.com/dhcgn/mbox-to-imap/runner"
	"github.com/dhcgn/mbox-to-imap/state"
	"github.com/dhcgn/mbox-to-imap/stats"
)

var (
	ErrNotImplemented   = errors.New("imap uploader not implemented yet")
	ErrMissingMessageID = errors.New("message id is empty")
)

type Options struct {
	Host               string
	Port               int
	Username           string
	Password           string
	UseTLS             bool
	InsecureSkipVerify bool
	TargetFolder       string
	DryRun             bool
}

type Uploader struct {
	opts    Options
	runner  *runner.Runner
	tracker state.Tracker
	uploads <-chan model.Message
	logger  *slog.Logger
}

func NewUploader(opts Options, r *runner.Runner, logger *slog.Logger) (*Uploader, error) {
	if opts.Host == "" {
		return nil, fmt.Errorf("imap host is empty")
	}
	if opts.Port <= 0 {
		return nil, fmt.Errorf("imap port must be positive")
	}
	tracker := r.Tracker()
	if tracker == nil {
		return nil, fmt.Errorf("tracker must not be nil")
	}
	uploader := &Uploader{
		opts:    opts,
		runner:  r,
		tracker: tracker,
		uploads: r.Uploads(),
		logger:  logger,
	}
	r.AddStage("imap", uploader.run)
	return uploader, nil
}

func (u *Uploader) run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-u.uploads:
			if !ok {
				return nil
			}
			if msg.ID == "" {
				u.runner.EmitEvent(stats.Event{Stage: stats.StageIMAP, Type: stats.EventTypeError, Err: ErrMissingMessageID})
				continue
			}

			if u.opts.DryRun {
				u.tracker.MarkProcessed(msg.ID)
				u.runner.EmitEvent(stats.Event{Stage: stats.StageIMAP, Type: stats.EventTypeDryRunUpload, MessageID: msg.ID})
				if u.logger != nil {
					u.logger.Debug("dry-run upload", "messageID", msg.ID, "target", u.opts.TargetFolder)
				}
				continue
			}

			u.runner.EmitEvent(stats.Event{Stage: stats.StageIMAP, Type: stats.EventTypeError, MessageID: msg.ID, Err: ErrNotImplemented})
			return ErrNotImplemented
		}
	}
}
