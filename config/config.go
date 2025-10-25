package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// Config captures all command-line options required to run the importer.
type Config struct {
	MboxPath           string
	IMAPHost           string
	IMAPPort           int
	IMAPUser           string
	IMAPPass           string
	UseTLS             bool
	InsecureSkipVerify bool
	TargetFolder       string
	StateDir           string
	DryRun             bool
	LogLevel           string
	IncludeHeader      []string
	IncludeBody        []string
	ExcludeHeader      []string
	ExcludeBody        []string
}

// RegisterFlags attaches all CLI flags to the provided command.
func RegisterFlags(cmd *cobra.Command) error {
	defaultStateDir, err := defaultStateDir()
	if err != nil {
		return err
	}

	flags := cmd.Flags()
	flags.String("mbox", "", "Path to the .mbox file to import")
	flags.String("imap-host", "", "IMAP server hostname")
	flags.Int("imap-port", 993, "IMAP server port")
	flags.String("imap-user", "", "IMAP username")
	flags.String("imap-pass", "", "IMAP password (falls back to IMAP_PASS env var)")
	flags.Bool("use-tls", true, "Use TLS for the IMAP connection")
	flags.Bool("insecure-skip-verify", false, "Skip TLS certificate verification (not recommended)")
	flags.String("target-folder", "INBOX", "Target IMAP folder for imported mail")
	flags.String("state-dir", defaultStateDir, "Directory for incremental sync state files")
	flags.Bool("dry-run", false, "Simulate the sync and emit stats without uploading")
	flags.String("log-level", "info", "Logging level: debug, info, warn, error")
	flags.StringArray("include-header", nil, "Regex allow-list applied to message headers (mutually exclusive with exclude flags)")
	flags.StringArray("include-body", nil, "Regex allow-list applied to message bodies (mutually exclusive with exclude flags)")
	flags.StringArray("exclude-header", nil, "Regex block-list applied to message headers (mutually exclusive with include flags)")
	flags.StringArray("exclude-body", nil, "Regex block-list applied to message bodies (mutually exclusive with include flags)")

	if err := cmd.MarkFlagRequired("mbox"); err != nil {
		return err
	}
	if err := cmd.MarkFlagRequired("imap-host"); err != nil {
		return err
	}
	if err := cmd.MarkFlagRequired("imap-user"); err != nil {
		return err
	}

	return nil
}

// LoadConfig converts the parsed Cobra flags into a Config struct with validation.
func LoadConfig(cmd *cobra.Command) (Config, error) {
	flags := cmd.Flags()

	mboxPath, err := flags.GetString("mbox")
	if err != nil {
		return Config{}, err
	}
	imapHost, err := flags.GetString("imap-host")
	if err != nil {
		return Config{}, err
	}
	imapPort, err := flags.GetInt("imap-port")
	if err != nil {
		return Config{}, err
	}
	imapUser, err := flags.GetString("imap-user")
	if err != nil {
		return Config{}, err
	}
	imapPass, err := flags.GetString("imap-pass")
	if err != nil {
		return Config{}, err
	}
	useTLS, err := flags.GetBool("use-tls")
	if err != nil {
		return Config{}, err
	}
	insecureSkipVerify, err := flags.GetBool("insecure-skip-verify")
	if err != nil {
		return Config{}, err
	}
	targetFolder, err := flags.GetString("target-folder")
	if err != nil {
		return Config{}, err
	}
	stateDir, err := flags.GetString("state-dir")
	if err != nil {
		return Config{}, err
	}
	dryRun, err := flags.GetBool("dry-run")
	if err != nil {
		return Config{}, err
	}
	logLevel, err := flags.GetString("log-level")
	if err != nil {
		return Config{}, err
	}
	includeHeader, err := flags.GetStringArray("include-header")
	if err != nil {
		return Config{}, err
	}
	includeBody, err := flags.GetStringArray("include-body")
	if err != nil {
		return Config{}, err
	}
	excludeHeader, err := flags.GetStringArray("exclude-header")
	if err != nil {
		return Config{}, err
	}
	excludeBody, err := flags.GetStringArray("exclude-body")
	if err != nil {
		return Config{}, err
	}

	if imapPass == "" {
		imapPass = os.Getenv("IMAP_PASS")
	}

	if stateDir == "" {
		stateDir, err = defaultStateDir()
		if err != nil {
			return Config{}, err
		}
	}

	logLevel = strings.ToLower(logLevel)
	if logLevel == "warning" {
		logLevel = "warn"
	}

	cfg := Config{
		MboxPath:           mboxPath,
		IMAPHost:           imapHost,
		IMAPPort:           imapPort,
		IMAPUser:           imapUser,
		IMAPPass:           imapPass,
		UseTLS:             useTLS,
		InsecureSkipVerify: insecureSkipVerify,
		TargetFolder:       targetFolder,
		StateDir:           filepath.Clean(stateDir),
		DryRun:             dryRun,
		LogLevel:           logLevel,
		IncludeHeader:      includeHeader,
		IncludeBody:        includeBody,
		ExcludeHeader:      excludeHeader,
		ExcludeBody:        excludeBody,
	}

	if err := validateConfig(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func validateConfig(cfg Config) error {
	if cfg.MboxPath == "" {
		return fmt.Errorf("--mbox is required")
	}
	if cfg.IMAPHost == "" {
		return fmt.Errorf("--imap-host is required")
	}
	if cfg.IMAPUser == "" {
		return fmt.Errorf("--imap-user is required")
	}
	if cfg.IMAPPass == "" {
		return fmt.Errorf("IMAP password must be provided via --imap-pass or IMAP_PASS env var")
	}
	if cfg.IMAPPort <= 0 || cfg.IMAPPort > 65535 {
		return fmt.Errorf("--imap-port must be between 1 and 65535")
	}
	includeActive := len(cfg.IncludeHeader) > 0 || len(cfg.IncludeBody) > 0
	excludeActive := len(cfg.ExcludeHeader) > 0 || len(cfg.ExcludeBody) > 0
	if includeActive && excludeActive {
		return fmt.Errorf("include and exclude flags are mutually exclusive")
	}

	switch cfg.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("invalid --log-level: %s", cfg.LogLevel)
	}

	return nil
}

func defaultStateDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".mbox-to-imap", "state"), nil
}
