package model

import "time"

// Message represents a single email message extracted from an mbox archive.
type Message struct {
	ID         string
	Hash       string
	ReceivedAt time.Time
	Size       int64
	Raw        []byte
}

// Envelope wraps a message alongside an optional error encountered while decoding.
type Envelope struct {
	Message Message
	Err     error
}
