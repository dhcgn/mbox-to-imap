package state

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Tracker interface {
	AlreadyProcessed(hash string) bool
	MarkProcessed(hash, messageID string) error
	Snapshot() Snapshot
}

type Snapshot struct {
	Processed int
}

type MemoryTracker struct {
	mu        sync.RWMutex
	processed map[string]string
}

func NewMemoryTracker() *MemoryTracker {
	return &MemoryTracker{processed: make(map[string]string)}
}

func (m *MemoryTracker) AlreadyProcessed(hash string) bool {
	if hash == "" {
		return false
	}

	m.mu.RLock()
	_, ok := m.processed[hash]
	m.mu.RUnlock()
	return ok
}

func (m *MemoryTracker) MarkProcessed(hash, messageID string) error {
	if hash == "" {
		return nil
	}

	m.mu.Lock()
	m.processed[hash] = messageID
	m.mu.Unlock()
	return nil
}

func (m *MemoryTracker) Snapshot() Snapshot {
	m.mu.RLock()
	count := len(m.processed)
	m.mu.RUnlock()
	return Snapshot{Processed: count}
}

// FileTracker persists processed message hashes so future runs can skip them.
type FileTracker struct {
	*MemoryTracker
	path    string
	persist bool
}

type fileRecord struct {
	Hash      string `json:"hash"`
	MessageID string `json:"message_id"`
}

func NewFileTracker(stateDir string, persist bool) (*FileTracker, error) {
	if strings.TrimSpace(stateDir) == "" {
		return nil, fmt.Errorf("state directory is empty")
	}

	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return nil, fmt.Errorf("create state directory: %w", err)
	}

	tracker := &FileTracker{
		MemoryTracker: NewMemoryTracker(),
		path:          filepath.Join(stateDir, "processed.jsonl"),
		persist:       persist,
	}

	if err := tracker.load(); err != nil {
		return nil, err
	}

	return tracker, nil
}

func (f *FileTracker) load() error {
	file, err := os.Open(f.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("open state file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for line := 1; scanner.Scan(); line++ {
		text := scanner.Bytes()
		if len(text) == 0 {
			continue
		}

		var record fileRecord
		if err := json.Unmarshal(text, &record); err != nil {
			return fmt.Errorf("parse state line %d: %w", line, err)
		}
		if record.Hash == "" {
			continue
		}

		f.mu.Lock()
		f.processed[record.Hash] = record.MessageID
		f.mu.Unlock()
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read state file: %w", err)
	}

	return nil
}

func (f *FileTracker) MarkProcessed(hash, messageID string) error {
	if hash == "" {
		return nil
	}

	f.mu.Lock()
	if _, exists := f.processed[hash]; exists {
		f.mu.Unlock()
		return nil
	}
	f.processed[hash] = messageID
	f.mu.Unlock()

	if !f.persist {
		return nil
	}

	file, err := os.OpenFile(f.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open state file for append: %w", err)
	}
	defer file.Close()

	record := fileRecord{Hash: hash, MessageID: messageID}
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("encode state record: %w", err)
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write state record: %w", err)
	}

	return nil
}
