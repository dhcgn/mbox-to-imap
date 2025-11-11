package state

import "sync"

type Tracker interface {
	AlreadyProcessed(id string) bool
	MarkProcessed(id string)
	Snapshot() Snapshot
}

type Snapshot struct {
	Processed int
}

type MemoryTracker struct {
	mu        sync.RWMutex
	processed map[string]struct{}
}

func NewMemoryTracker() *MemoryTracker {
	return &MemoryTracker{processed: make(map[string]struct{})}
}

func (m *MemoryTracker) AlreadyProcessed(id string) bool {
	if id == "" {
		return false
	}

	m.mu.RLock()
	_, ok := m.processed[id]
	m.mu.RUnlock()
	return ok
}

func (m *MemoryTracker) MarkProcessed(id string) {
	if id == "" {
		return
	}

	m.mu.Lock()
	m.processed[id] = struct{}{}
	m.mu.Unlock()
}

func (m *MemoryTracker) Snapshot() Snapshot {
	m.mu.RLock()
	count := len(m.processed)
	m.mu.RUnlock()
	return Snapshot{Processed: count}
}
