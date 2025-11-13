package cmd

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/dhcgn/mbox-to-imap/mbox"
	"github.com/dhcgn/mbox-to-imap/stats"
	"github.com/spf13/cobra"
)

var (
	reportDir string
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

			if messageCount%250 == 0 {
				printStats()
			}

			return nil
		})

		if err != nil {
			return fmt.Errorf("error reading mbox file: %w", err)
		}

		// Final print
		printStats()

		// Save CSV reports
		if err := saveCSVReports(counter, headersToTrack, reportDir, 1000); err != nil {
			return fmt.Errorf("error saving CSV reports: %w", err)
		}

		fmt.Printf("\nReports saved to directory: %s\n", reportDir)

		return nil
	},
}

func init() {
	mboxStatsCmd.Flags().StringVarP(&reportDir, "output", "o", ".", "Output directory for CSV reports")
	rootCmd.AddCommand(mboxStatsCmd)
}

func saveCSVReports(counter map[string]map[string]int, headers []string, dir string, limit int) error {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// Write data for each header category to a separate file
	for _, header := range headers {
		counts := counter[header]

		// Generate filename
		filename := fmt.Sprintf("report_%s.csv", normalizeHeaderName(header))
		filePath := filepath.Join(dir, filename)

		file, err := os.Create(filePath)
		if err != nil {
			return err
		}

		writer := csv.NewWriter(file)

		// Write header
		if err := writer.Write([]string{"Value", "Count"}); err != nil {
			file.Close()
			return err
		}

		// Sort by count descending
		type pair struct {
			Key   string
			Value int
		}
		var pairs []pair
		for k, v := range counts {
			pairs = append(pairs, pair{k, v})
		}
		sort.Slice(pairs, func(i, j int) bool {
			return pairs[i].Value > pairs[j].Value
		})

		// Write top N entries
		for i := 0; i < limit && i < len(pairs); i++ {
			record := []string{
				pairs[i].Key,
				strconv.Itoa(pairs[i].Value),
			}
			if err := writer.Write(record); err != nil {
				file.Close()
				return err
			}
		}

		writer.Flush()
		file.Close()

		if err := writer.Error(); err != nil {
			return err
		}
	}

	return nil
}

func normalizeHeaderName(header string) string {
	// Convert to lowercase and replace invalid filename chars
	name := strings.ToLower(header)
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, " ", "_")
	return name
}
