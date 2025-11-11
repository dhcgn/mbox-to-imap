package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/dhcgn/mbox-to-imap/config"
	"github.com/dhcgn/mbox-to-imap/imap"
	"github.com/dhcgn/mbox-to-imap/mbox"
	"github.com/dhcgn/mbox-to-imap/runner"
	"github.com/dhcgn/mbox-to-imap/stats"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "mbox-to-imap",
		Short: "Import messages from mbox archives into an IMAP mailbox",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadConfig(cmd)
			if err != nil {
				return err
			}

			logger, cleanup, err := setupLogger(cfg)
			if err != nil {
				return err
			}
			defer func() {
				_ = cleanup()
			}()

			slog.SetDefault(logger)
			logger.Info("starting mbox-to-imap", "mbox", cfg.MboxPath, "target", cfg.TargetFolder, "dryRun", cfg.DryRun)

			return run(cfg, logger)
		},
	}

	if err := config.RegisterFlags(rootCmd); err != nil {
		fmt.Fprintf(os.Stderr, "failed to register CLI flags: %v\n", err)
		os.Exit(1)
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(cfg config.Config, logger *slog.Logger) error {
	r, err := runner.New(cfg, logger)
	if err != nil {
		return fmt.Errorf("runner.New: %w", err)
	}
	stats.NewReporter(r, logger)

	readerOpts := mbox.Options{
		Path:          cfg.MboxPath,
		IncludeHeader: cfg.IncludeHeader,
		IncludeBody:   cfg.IncludeBody,
		ExcludeHeader: cfg.ExcludeHeader,
		ExcludeBody:   cfg.ExcludeBody,
	}

	if _, err := mbox.NewProducer(readerOpts, r, logger); err != nil {
		return fmt.Errorf("mbox.NewProducer: %w", err)
	}

	uploaderOpts := imap.Options{
		Host:               cfg.IMAPHost,
		Port:               cfg.IMAPPort,
		Username:           cfg.IMAPUser,
		Password:           cfg.IMAPPass,
		UseTLS:             cfg.UseTLS,
		InsecureSkipVerify: cfg.InsecureSkipVerify,
		TargetFolder:       cfg.TargetFolder,
		DryRun:             cfg.DryRun,
	}

	if _, err := imap.NewUploader(uploaderOpts, r, logger); err != nil {
		return fmt.Errorf("imap.NewUploader: %w", err)
	}

	return r.Start()
}

func setupLogger(cfg config.Config) (*slog.Logger, func() error, error) {
	level := new(slog.LevelVar)
	level.Set(slog.LevelInfo)

	switch cfg.LogLevel {
	case "debug":
		level.Set(slog.LevelDebug)
	case "info":
		level.Set(slog.LevelInfo)
	case "warn":
		level.Set(slog.LevelWarn)
	case "error":
		level.Set(slog.LevelError)
	}

	opts := &slog.HandlerOptions{Level: level}
	cleanup := func() error { return nil }

	if cfg.LogDir != "" {
		if err := os.MkdirAll(cfg.LogDir, 0o755); err != nil {
			return nil, cleanup, err
		}

		logFilePath := filepath.Join(cfg.LogDir, fmt.Sprintf("mbox-to-imap-%s.log", time.Now().Format("20060102T150405")))
		file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return nil, cleanup, err
		}

		handler := slog.NewTextHandler(io.MultiWriter(os.Stdout, file), opts)
		cleanup = func() error {
			return file.Close()
		}
		return slog.New(handler), cleanup, nil
	}

	handler := slog.NewTextHandler(os.Stdout, opts)
	return slog.New(handler), cleanup, nil
}
