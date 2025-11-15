package filter

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

// Options captures the filtering configuration.
type Options struct {
	IncludeHeader []string
	IncludeBody   []string
	ExcludeHeader []string
	ExcludeBody   []string
}

// Filter holds compiled regex patterns for filtering messages.
type Filter struct {
	includeMode    bool
	excludeMode    bool
	includeHeader  []*regexp.Regexp
	includeBody    []*regexp.Regexp
	excludeHeader  []*regexp.Regexp
	excludeBody    []*regexp.Regexp
	needHeaderText bool
	needBodyText   bool
	// Tracking
	includeHeaderHits map[string]int
	includeBodyHits   map[string]int
	excludeHeaderHits map[string]int
	excludeBodyHits   map[string]int
}

// New creates a new Filter from the provided options.
func New(opts Options) (*Filter, error) {
	includeHeader, err := compilePatterns(opts.IncludeHeader)
	if err != nil {
		return nil, fmt.Errorf("compile include-header pattern: %w", err)
	}
	includeBody, err := compilePatterns(opts.IncludeBody)
	if err != nil {
		return nil, fmt.Errorf("compile include-body pattern: %w", err)
	}
	excludeHeader, err := compilePatterns(opts.ExcludeHeader)
	if err != nil {
		return nil, fmt.Errorf("compile exclude-header pattern: %w", err)
	}
	excludeBody, err := compilePatterns(opts.ExcludeBody)
	if err != nil {
		return nil, fmt.Errorf("compile exclude-body pattern: %w", err)
	}

	includeActive := len(includeHeader) > 0 || len(includeBody) > 0
	excludeActive := len(excludeHeader) > 0 || len(excludeBody) > 0
	if includeActive && excludeActive {
		return nil, fmt.Errorf("include and exclude filters are mutually exclusive")
	}

	return &Filter{
		includeMode:       includeActive,
		excludeMode:       excludeActive,
		includeHeader:     includeHeader,
		includeBody:       includeBody,
		excludeHeader:     excludeHeader,
		excludeBody:       excludeBody,
		needHeaderText:    len(includeHeader) > 0 || len(excludeHeader) > 0,
		needBodyText:      len(includeBody) > 0 || len(excludeBody) > 0,
		includeHeaderHits: make(map[string]int),
		includeBodyHits:   make(map[string]int),
		excludeHeaderHits: make(map[string]int),
		excludeBodyHits:   make(map[string]int),
	}, nil
}

// Allows returns true if the message passes the filter criteria.
func (f *Filter) Allows(header, body []byte) bool {
	var headerText, bodyText string
	if f.needHeaderText {
		headerText = string(header)
	}
	if f.needBodyText {
		bodyText = string(body)
	}

	if f.includeMode {
		matched := f.matchAnyWithTracking(f.includeHeader, headerText, f.includeHeaderHits) ||
			f.matchAnyWithTracking(f.includeBody, bodyText, f.includeBodyHits)
		return matched
	}

	if f.excludeMode {
		if f.matchAnyWithTracking(f.excludeHeader, headerText, f.excludeHeaderHits) ||
			f.matchAnyWithTracking(f.excludeBody, bodyText, f.excludeBodyHits) {
			return false
		}
	}

	return true
}

// matchAnyWithTracking checks if any pattern matches and tracks which ones hit.
func (f *Filter) matchAnyWithTracking(patterns []*regexp.Regexp, text string, hitCounter map[string]int) bool {
	if len(patterns) == 0 {
		return false
	}
	matched := false
	for _, re := range patterns {
		if re.MatchString(text) {
			hitCounter[re.String()]++
			matched = true
		}
	}
	return matched
}

// GetStats returns the filter hit statistics.
type FilterStats struct {
	IncludeHeaderPatterns []string
	IncludeBodyPatterns   []string
	ExcludeHeaderPatterns []string
	ExcludeBodyPatterns   []string
	IncludeHeaderHits     map[string]int
	IncludeBodyHits       map[string]int
	ExcludeHeaderHits     map[string]int
	ExcludeBodyHits       map[string]int
}

func (f *Filter) GetStats() FilterStats {
	stats := FilterStats{
		IncludeHeaderHits: f.includeHeaderHits,
		IncludeBodyHits:   f.includeBodyHits,
		ExcludeHeaderHits: f.excludeHeaderHits,
		ExcludeBodyHits:   f.excludeBodyHits,
	}

	// Collect all patterns
	for _, re := range f.includeHeader {
		stats.IncludeHeaderPatterns = append(stats.IncludeHeaderPatterns, re.String())
	}
	for _, re := range f.includeBody {
		stats.IncludeBodyPatterns = append(stats.IncludeBodyPatterns, re.String())
	}
	for _, re := range f.excludeHeader {
		stats.ExcludeHeaderPatterns = append(stats.ExcludeHeaderPatterns, re.String())
	}
	for _, re := range f.excludeBody {
		stats.ExcludeBodyPatterns = append(stats.ExcludeBodyPatterns, re.String())
	}

	return stats
}

// SplitRawMessage splits a raw email message into header and body parts.
func SplitRawMessage(raw []byte) (header, body []byte) {
	if len(raw) == 0 {
		return nil, nil
	}

	if idx := bytes.Index(raw, []byte("\r\n\r\n")); idx >= 0 {
		return raw[:idx], raw[idx+4:]
	}
	if idx := bytes.Index(raw, []byte("\n\n")); idx >= 0 {
		return raw[:idx], raw[idx+2:]
	}

	return raw, nil
}

func compilePatterns(patterns []string) ([]*regexp.Regexp, error) {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("compile %q: %w", pattern, err)
		}
		compiled = append(compiled, re)
	}
	return compiled, nil
}

func matchAny(patterns []*regexp.Regexp, text string) bool {
	if len(patterns) == 0 {
		return false
	}
	for _, re := range patterns {
		if re.MatchString(text) {
			return true
		}
	}
	return false
}
