package cmd

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/dhcgn/mbox-to-imap/filter"
	"github.com/dhcgn/mbox-to-imap/mbox"
	"github.com/dhcgn/mbox-to-imap/stats"
	"github.com/spf13/cobra"
)

var (
	reportDir     string
	topN          int
	includeHeader []string
	includeBody   []string
	excludeHeader []string
	excludeBody   []string
)

var mboxStatsCmd = &cobra.Command{
	Use:   "mbox-stats [mbox file]",
	Short: "Analyse the mbox file and show statistics",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mboxPath := args[0]

		fmt.Println("Analyzing mbox file:", mboxPath)

		// Validate filter flags
		includeActive := len(includeHeader) > 0 || len(includeBody) > 0
		excludeActive := len(excludeHeader) > 0 || len(excludeBody) > 0
		if includeActive && excludeActive {
			return fmt.Errorf("include and exclude flags are mutually exclusive")
		}

		// Create filter
		filterOpts := filter.Options{
			IncludeHeader: includeHeader,
			IncludeBody:   includeBody,
			ExcludeHeader: excludeHeader,
			ExcludeBody:   excludeBody,
		}
		f, err := filter.New(filterOpts)
		if err != nil {
			return fmt.Errorf("create filter: %w", err)
		}

		counter := make(map[string]map[string]int)
		headersToTrack := []string{"Delivered-To", "Subject", "From", "To"}
		for _, h := range headersToTrack {
			counter[h] = make(map[string]int)
		}

		messageCount := 0
		skippedCount := 0
		printStats := func() {
			// ANSI escape code to clear screen and move cursor to top-left
			fmt.Print("\033[H\033[2J")
			totalMessages := messageCount + skippedCount
			var filterPercent float64
			if totalMessages > 0 {
				filterPercent = float64(skippedCount) / float64(totalMessages) * 100
			}
			fmt.Printf("Processed %d messages (skipped %d by filters, %.2f%%)...\n\n", messageCount, skippedCount, filterPercent)

			// Print filter statistics
			filterStats := f.GetStats()
			hasFilterStats := false

			if len(filterStats.IncludeHeaderPatterns) > 0 {
				hasFilterStats = true
				fmt.Println("Include Header Filters:")
				printFilterHits(filterStats.IncludeHeaderPatterns, filterStats.IncludeHeaderHits)
				fmt.Println()
			}

			if len(filterStats.IncludeBodyPatterns) > 0 {
				hasFilterStats = true
				fmt.Println("Include Body Filters:")
				printFilterHits(filterStats.IncludeBodyPatterns, filterStats.IncludeBodyHits)
				fmt.Println()
			}

			if len(filterStats.ExcludeHeaderPatterns) > 0 {
				hasFilterStats = true
				fmt.Println("Exclude Header Filters:")
				printFilterHits(filterStats.ExcludeHeaderPatterns, filterStats.ExcludeHeaderHits)
				fmt.Println()
			}

			if len(filterStats.ExcludeBodyPatterns) > 0 {
				hasFilterStats = true
				fmt.Println("Exclude Body Filters:")
				printFilterHits(filterStats.ExcludeBodyPatterns, filterStats.ExcludeBodyHits)
				fmt.Println()
			}

			if hasFilterStats {
				fmt.Println("---")
				fmt.Println()
			}

			for _, header := range headersToTrack {
				fmt.Printf("Top %d %s:\n", topN, header)
				stats.PrettyPrintTop(counter[header], topN)
				fmt.Println()
			}
		}

		err = mbox.Read(mboxPath, func(m *mbox.MboxMessage) error {
			// Apply filter
			headerBytes, readErr := io.ReadAll(strings.NewReader(formatHeaders(m.Headers)))
			if readErr != nil {
				return readErr
			}
			if !f.Allows(headerBytes, m.Body) {
				skippedCount++
				return nil
			}

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
	mboxStatsCmd.Flags().IntVarP(&topN, "top", "t", 10, "Number of top items to display in statistics")
	mboxStatsCmd.Flags().StringArrayVar(&includeHeader, "include-header", nil, "Regex allow-list applied to message headers (mutually exclusive with exclude flags)")
	mboxStatsCmd.Flags().StringArrayVar(&includeBody, "include-body", nil, "Regex allow-list applied to message bodies (mutually exclusive with exclude flags)")
	mboxStatsCmd.Flags().StringArrayVar(&excludeHeader, "exclude-header", nil, "Regex block-list applied to message headers (mutually exclusive with include flags)")
	mboxStatsCmd.Flags().StringArrayVar(&excludeBody, "exclude-body", nil, "Regex block-list applied to message bodies (mutually exclusive with include flags)")
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

func formatHeaders(headers map[string][]string) string {
	var sb strings.Builder
	for key, values := range headers {
		for _, value := range values {
			sb.WriteString(key)
			sb.WriteString(": ")
			sb.WriteString(value)
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func printFilterHits(patterns []string, hits map[string]int) {
	// Sort by hit count descending
	type pair struct {
		Pattern string
		Count   int
		HasHits bool
	}
	var pairs []pair

	// Add all patterns with their hit counts
	for _, pattern := range patterns {
		count := hits[pattern]
		pairs = append(pairs, pair{pattern, count, count > 0})
	}

	sort.Slice(pairs, func(i, j int) bool {
		// Sort by hit count descending, then by pattern
		if pairs[i].Count != pairs[j].Count {
			return pairs[i].Count > pairs[j].Count
		}
		return pairs[i].Pattern < pairs[j].Pattern
	})

	for _, p := range pairs {
		if p.HasHits {
			fmt.Printf("  ✓ %s: %d hits\n", p.Pattern, p.Count)
		} else {
			fmt.Printf("  ✗ %s: 0 hits\n", p.Pattern)
		}
	}
}
