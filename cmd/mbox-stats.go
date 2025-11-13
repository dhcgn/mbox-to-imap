package cmd

import (
	"fmt"

	"github.com/dhcgn/mbox-to-imap/mbox"
	"github.com/dhcgn/mbox-to-imap/stats"
	"github.com/spf13/cobra"
)

var mboxStatsCmd = &cobra.Command{
	Use:   "mbox-stats [mbox file]",
	Short: "Analyse the mbox file and show statistics",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mboxPath := args[0]

		fmt.Println("Analyzing mbox file:", mboxPath)

		counter := make(map[string]map[string]int)
		headersToTrack := []string{"Delivered-To", "Subject", "From", "To"}
		for _, h := range headersToTrack {
			counter[h] = make(map[string]int)
		}

		messageCount := 0
		printStats := func() {
			// ANSI escape code to clear screen and move cursor to top-left
			fmt.Print("\033[H\033[2J")
			fmt.Printf("Processed %d messages...\n\n", messageCount)
			for _, header := range headersToTrack {
				fmt.Printf("Top 10 %s:\n", header)
				stats.PrettyPrintTop(counter[header], 10)
				fmt.Println()
			}
		}

		err := mbox.Read(mboxPath, func(m *mbox.MboxMessage) error {
			messageCount++
			for _, headerName := range headersToTrack {
				if value := m.Headers.Get(headerName); value != "" {
					counter[headerName][value]++
				}
			}

			if messageCount%100 == 0 {
				printStats()
			}

			return nil
		})

		if err != nil {
			return fmt.Errorf("error reading mbox file: %w", err)
		}

		// Final print
		printStats()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(mboxStatsCmd)
}
