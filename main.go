package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/dhcgn/mbox-to-imap/config"
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

			return run(cfg)
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

func run(cfg config.Config) error {
	// TODO: wire the configuration into the importer workflow.
	return nil
}
