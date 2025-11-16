package mbox

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/mail"
	"os"
	"strings"
	"time"

	mboxlib "github.com/emersion/go-mbox"

	"github.com/dhcgn/mbox-to-imap/filter"
	"github.com/dhcgn/mbox-to-imap/model"
	"github.com/dhcgn/mbox-to-imap/runner"
)

var (
	ErrMessageIDMissing = errors.New("mbox message missing Message-Id header")
)

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
	path := strings.TrimSpace(opts.Path)
	if path == "" {
		return nil, fmt.Errorf("mbox path is empty")
	}

	filterOpts := filter.Options{
		IncludeHeader: opts.IncludeHeader,
		IncludeBody:   opts.IncludeBody,
		ExcludeHeader: opts.ExcludeHeader,
		ExcludeBody:   opts.ExcludeBody,
	}

	f, err := filter.New(filterOpts)
	if err != nil {
		return nil, err
	}

	reader := &fileReader{
		path:   path,
		logger: logger,
		filter: f,
	}

	return reader, nil
}

type fileReader struct {
	path   string
	logger *slog.Logger
	filter *filter.Filter
}

func (f *fileReader) Stream(ctx context.Context, out chan<- model.Envelope) error {
	var reader *mboxlib.Reader

	if mbox_test_data_using {
		reader = mboxlib.NewReader(bytes.NewReader(mbox_test_data))
	} else {
		file, err := os.Open(f.path)
		if err != nil {
			return fmt.Errorf("open mbox: %w", err)
		}
		defer file.Close()
		reader = mboxlib.NewReader(file)
	}

	for idx := 0; ; idx++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		msgReader, err := reader.NextMessage()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return f.emitError(ctx, out, fmt.Errorf("message %d: %w", idx, err))
		}

		raw, err := io.ReadAll(msgReader)
		if err != nil {
			return f.emitError(ctx, out, fmt.Errorf("message %d read: %w", idx, err))
		}

		header, body := filter.SplitRawMessage(raw)
		if !f.filter.Allows(header, body) {
			continue
		}

		msg, err := parseMail(raw)
		if err != nil {
			if errors.Is(err, ErrMessageIDMissing) {
				err = fmt.Errorf("message %d: %w", idx, err)
			} else {
				err = fmt.Errorf("message %d parse: %w", idx, err)
			}
			return f.emitError(ctx, out, err)
		}

		msg.Size = int64(len(raw))
		msg.Raw = raw

		if err := f.emitEnvelope(ctx, out, model.Envelope{Message: msg}); err != nil {
			return err
		}
	}
}

func (f *fileReader) emitError(ctx context.Context, out chan<- model.Envelope, err error) error {
	if f.logger != nil {
		f.logger.Error("mbox stream error", "path", f.path, "err", err)
	}
	if err := f.emitEnvelope(ctx, out, model.Envelope{Err: err}); err != nil {
		return err
	}
	return nil
}

func (f *fileReader) emitEnvelope(ctx context.Context, out chan<- model.Envelope, env model.Envelope) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case out <- env:
		return nil
	}
}

func parseMail(raw []byte) (model.Message, error) {
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		return model.Message{}, err
	}

	id := strings.TrimSpace(msg.Header.Get("Message-Id"))
	if id == "" {
		id = strings.TrimSpace(msg.Header.Get("Message-ID"))
	}
	id = strings.Trim(id, " <>")
	if id == "" {
		return model.Message{}, ErrMessageIDMissing
	}

	var receivedAt time.Time
	if date := msg.Header.Get("Date"); date != "" {
		if t, err := mail.ParseDate(date); err == nil {
			receivedAt = t
		}
	}

	sum := sha256.Sum256(raw)
	hash := base64.StdEncoding.EncodeToString(sum[:])

	return model.Message{
		ID:         id,
		Hash:       hash,
		ReceivedAt: receivedAt,
	}, nil
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

// MboxMessage represents a single message from an mbox file for stats.
type MboxMessage struct {
	Headers mail.Header
	Body    []byte
}

var (
	mbox_test_data_using = false
	mbox_test_data       []byte
)

// Read opens an mbox file and iterates through its messages,
// calling the provided callback for each message.
func Read(path string, callback func(m *MboxMessage) error) error {
	var reader *mboxlib.Reader

	if mbox_test_data_using {
		reader = mboxlib.NewReader(bytes.NewReader(mbox_test_data))
	} else {
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("open mbox: %w", err)
		}
		defer file.Close()
		reader = mboxlib.NewReader(file)
	}

	for {
		msgReader, err := reader.NextMessage()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		msg, err := mail.ReadMessage(msgReader)
		if err != nil {
			// try to continue
			continue
		}

		body, err := io.ReadAll(msg.Body)
		if err != nil {
			// try to continue
			continue
		}

		mboxMsg := &MboxMessage{
			Headers: msg.Header,
			Body:    body,
		}

		if err := callback(mboxMsg); err != nil {
			return err
		}
	}
}

// CountMessages counts the total number of messages in an mbox file.
// If progressCallback is provided, it will be called with (bytesRead, totalSize) during counting.
func CountMessages(path string, progressCallback func(bytesRead, totalSize int64)) (int, error) {
	var reader *mboxlib.Reader
	var fileSize int64
	var progressReader io.Reader

	if mbox_test_data_using {
		reader = mboxlib.NewReader(bytes.NewReader(mbox_test_data))
	} else {
		file, err := os.Open(path)
		if err != nil {
			return 0, fmt.Errorf("open mbox: %w", err)
		}
		defer file.Close()

		// Get file size for progress tracking
		stat, err := file.Stat()
		if err != nil {
			return 0, fmt.Errorf("stat mbox: %w", err)
		}
		fileSize = stat.Size()

		// Wrap file reader with progress tracking if callback provided
		if progressCallback != nil {
			progressReader = &progressTrackingReader{
				r:           file,
				total:       fileSize,
				callback:    progressCallback,
				reportEvery: 100 * time.Millisecond, // Report every 100ms
				lastReport:  time.Now(),
			}
			reader = mboxlib.NewReader(progressReader)
		} else {
			reader = mboxlib.NewReader(file)
		}
	}

	count := 0
	for {
		msgReader, err := reader.NextMessage()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return count, nil
			}
			return 0, err
		}

		// Just consume the message without parsing
		_, err = io.Copy(io.Discard, msgReader)
		if err != nil {
			// Continue counting even if we can't read this message
			count++
			continue
		}

		count++
	}
}

// progressTrackingReader wraps an io.Reader and reports progress via callback.
type progressTrackingReader struct {
	r           io.Reader
	total       int64
	read        int64
	lastReport  time.Time
	reportEvery time.Duration
	callback    func(read, total int64)
}

func (p *progressTrackingReader) Read(buf []byte) (int, error) {
	n, err := p.r.Read(buf)
	p.read += int64(n)

	// Use time-based throttling instead of count-based
	if p.callback != nil {
		now := time.Now()
		if now.Sub(p.lastReport) >= p.reportEvery {
			p.callback(p.read, p.total)
			p.lastReport = now
		}
	}
	return n, err
}
