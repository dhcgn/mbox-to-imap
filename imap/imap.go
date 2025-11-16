package imap

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"

	imapv2 "github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"

	"github.com/dhcgn/mbox-to-imap/model"
	"github.com/dhcgn/mbox-to-imap/runner"
	"github.com/dhcgn/mbox-to-imap/state"
	"github.com/dhcgn/mbox-to-imap/stats"
)

var (
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
	var (
		client  *imapclient.Client
		cleanup func()
	)
	defer func() {
		if cleanup != nil {
			cleanup()
		}
	}()

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
			if msg.Hash == "" {
				err := fmt.Errorf("message %s missing hash", msg.ID)
				u.runner.EmitEvent(stats.Event{Stage: stats.StageIMAP, Type: stats.EventTypeError, MessageID: msg.ID, Err: err})
				return err
			}

			if u.opts.DryRun {
				if err := u.tracker.MarkProcessed(msg.Hash, msg.ID); err != nil {
					u.runner.EmitEvent(stats.Event{Stage: stats.StageIMAP, Type: stats.EventTypeError, MessageID: msg.ID, Err: err})
					return err
				}
				u.runner.EmitEvent(stats.Event{Stage: stats.StageIMAP, Type: stats.EventTypeDryRunUpload, MessageID: msg.ID})
				if u.logger != nil {
					u.logger.Debug("dry-run upload", "messageID", msg.ID, "target", u.targetFolder(), "hash", msg.Hash)
				}
				continue
			}

			if client == nil {
				var err error
				client, cleanup, err = u.dial(ctx)
				if err != nil {
					u.runner.EmitEvent(stats.Event{Stage: stats.StageIMAP, Type: stats.EventTypeError, MessageID: msg.ID, Err: err})
					return err
				}
			}

			if err := u.appendMessage(client, msg); err != nil {
				err = fmt.Errorf("upload message %s: %w", msg.ID, err)
				u.runner.EmitEvent(stats.Event{Stage: stats.StageIMAP, Type: stats.EventTypeError, MessageID: msg.ID, Err: err})
				return err
			}

			if err := u.tracker.MarkProcessed(msg.Hash, msg.ID); err != nil {
				u.runner.EmitEvent(stats.Event{Stage: stats.StageIMAP, Type: stats.EventTypeError, MessageID: msg.ID, Err: err})
				return err
			}

			u.runner.EmitEvent(stats.Event{Stage: stats.StageIMAP, Type: stats.EventTypeUploaded, MessageID: msg.ID})
			if u.logger != nil {
				u.logger.Debug("uploaded message", "messageID", msg.ID, "target", u.targetFolder(), "hash", msg.Hash)
			}
		}
	}
}

func (u *Uploader) dial(ctx context.Context) (*imapclient.Client, func(), error) {
	address := net.JoinHostPort(u.opts.Host, strconv.Itoa(u.opts.Port))
	options := &imapclient.Options{}

	if u.opts.UseTLS {
		options.TLSConfig = &tls.Config{
			ServerName:         u.opts.Host,
			InsecureSkipVerify: u.opts.InsecureSkipVerify,
		}
	}

	var (
		client *imapclient.Client
		err    error
	)

	if u.opts.UseTLS {
		client, err = imapclient.DialTLS(address, options)
	} else {
		client, err = imapclient.DialInsecure(address, options)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("dial imap %s: %w", address, err)
	}

	if err := client.Login(u.opts.Username, u.opts.Password).Wait(); err != nil {
		_ = client.Close()
		return nil, nil, fmt.Errorf("imap login failed: %w", err)
	}

	if err := u.ensureMailbox(client); err != nil {
		_ = client.Close()
		return nil, nil, err
	}

	// TODO get all uids in the target mailbox and add them to the u *Uploader so that dulicate are not uploaded again
	// add count of messages in the target mailbox to the logger

	if u.logger != nil {
		u.logger.Debug("imap connection established", "address", address, "user", u.opts.Username, "target", u.targetFolder(), "tls", u.opts.UseTLS)
	}

	stopClose := context.AfterFunc(ctx, func() {
		_ = client.Close()
	})

	cleanup := func() {
		stopClose()
		if ctx.Err() == nil {
			if err := client.Logout().Wait(); err != nil {
				if u.logger != nil {
					u.logger.Warn("imap logout failed", "err", err)
				}
			}
		}
		if err := client.Close(); err != nil && u.logger != nil {
			u.logger.Debug("imap connection closed", "err", err)
		}
	}

	return client, cleanup, nil
}

func (u *Uploader) appendMessage(client *imapclient.Client, msg model.Message) error {
	target := u.targetFolder()
	size := int64(len(msg.Raw))

	var opts *imapv2.AppendOptions
	if !msg.ReceivedAt.IsZero() {
		opts = &imapv2.AppendOptions{Time: msg.ReceivedAt}
	}

	cmd := client.Append(target, size, opts)

	remaining := msg.Raw
	for len(remaining) > 0 {
		n, err := cmd.Write(remaining)
		if err != nil {
			_ = cmd.Close()
			return fmt.Errorf("append write: %w", err)
		}
		if n == 0 {
			_ = cmd.Close()
			return fmt.Errorf("append write: wrote 0 bytes")
		}
		remaining = remaining[n:]
	}

	if err := cmd.Close(); err != nil {
		return fmt.Errorf("append close: %w", err)
	}

	if _, err := cmd.Wait(); err != nil {
		return fmt.Errorf("append wait: %w", err)
	}

	return nil
}

func (u *Uploader) targetFolder() string {
	if u.opts.TargetFolder == "" {
		return "INBOX"
	}
	return u.opts.TargetFolder
}

func (u *Uploader) ensureMailbox(client *imapclient.Client) error {
	target := u.targetFolder()
	cmd := client.Create(target, nil)
	if err := cmd.Wait(); err != nil {
		var respErr *imapv2.Error
		if errors.As(err, &respErr) {
			if respErr.Code == imapv2.ResponseCodeAlreadyExists {
				if u.logger != nil {
					u.logger.Debug("imap mailbox already exists", "mailbox", target)
				}
				return nil
			}
		}
		return fmt.Errorf("ensure mailbox %s: %w", target, err)
	}

	if u.logger != nil {
		u.logger.Info("imap mailbox created", "mailbox", target)
	}

	return nil
}
