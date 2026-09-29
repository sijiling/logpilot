// Package document defines knowledge-document indexing models.
package document

import "time"

// Status is the indexing lifecycle state of a document.
type Status string

const (
	StatusProcessing Status = "processing"
	StatusReady      Status = "ready"
	StatusFailed     Status = "failed"
)

// Record describes a knowledge document stored in SQLite.
type Record struct {
	ID           string
	Title        string
	ContentHash  string
	Status       Status
	ChunkCount   int
	ErrorMessage string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
