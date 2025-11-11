package mbox

import (
	"context"
	"errors"
	"log/slog"

	"github.com/dhcgn/mbox-to-imap/model"
	"github.com/dhcgn/mbox-to-imap/runner"
)

var ErrNotImplemented = errors.New("mbox reader not implemented yet")

type Options struct {
	Path          string
	IncludeHeader []string
	IncludeBody   []string
	ExcludeHeader []string
	ExcludeBody   []string
}

type Reader interface {
	Stream(ctx context.Context, out chan<- model.Envelope) error
}

func NewReader(opts Options, logger *slog.Logger) (Reader, error) {
	return &noopReader{opts: opts, logger: logger}, nil
}

type noopReader struct {
	opts   Options
	logger *slog.Logger
}

func (n *noopReader) Stream(ctx context.Context, out chan<- model.Envelope) error {
	if n.logger != nil {
		n.logger.Warn("mbox reader not implemented", "path", n.opts.Path)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case out <- model.Envelope{Err: ErrNotImplemented}:
		return nil
	}
}

type Producer struct {
	reader Reader
	runner *runner.Runner
}

func NewProducer(opts Options, r *runner.Runner, logger *slog.Logger) (*Producer, error) {
	reader, err := NewReader(opts, logger)
	if err != nil {
		return nil, err
	}
	producer := &Producer{reader: reader, runner: r}
	r.AddStage("mbox", producer.run)
	return producer, nil
}

func (p *Producer) run(ctx context.Context) error {
	defer p.runner.CloseMailbox()
	return p.reader.Stream(ctx, p.runner.MailboxWriter())
}
