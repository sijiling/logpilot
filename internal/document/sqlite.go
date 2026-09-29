package document

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// SQLiteStore persists document indexing state.
type SQLiteStore struct{ db *sql.DB }

// NewSQLiteStore creates a document store on an initialized LogPilot database.
func NewSQLiteStore(db *sql.DB) *SQLiteStore { return &SQLiteStore{db: db} }

func (s *SQLiteStore) FindByHash(ctx context.Context, hash string) (Record, bool, error) {
	var record Record
	var createdAt, updatedAt string
	err := s.db.QueryRowContext(ctx, `SELECT id, title, content_hash, status, chunk_count, error_message, created_at, updated_at
        FROM documents WHERE content_hash = ?`, hash).Scan(&record.ID, &record.Title, &record.ContentHash, &record.Status, &record.ChunkCount, &record.ErrorMessage, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Record{}, false, nil
	}
	if err != nil {
		return Record{}, false, fmt.Errorf("query document by hash: %w", err)
	}
	record.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return Record{}, false, fmt.Errorf("parse document created time: %w", err)
	}
	record.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil {
		return Record{}, false, fmt.Errorf("parse document updated time: %w", err)
	}
	return record, true, nil
}

func (s *SQLiteStore) Create(ctx context.Context, record Record) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO documents
        (id, title, content_hash, status, chunk_count, error_message, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, record.ID, record.Title, record.ContentHash, record.Status, record.ChunkCount, record.ErrorMessage, record.CreatedAt.UTC().Format(time.RFC3339Nano), record.UpdatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("insert document: %w", err)
	}
	return nil
}

func (s *SQLiteStore) MarkReady(ctx context.Context, id string, chunkCount int, updatedAt time.Time) error {
	_, err := s.db.ExecContext(ctx, "UPDATE documents SET status = ?, chunk_count = ?, error_message = '', updated_at = ? WHERE id = ?", StatusReady, chunkCount, updatedAt.UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return fmt.Errorf("mark document ready: %w", err)
	}
	return nil
}

func (s *SQLiteStore) MarkFailed(ctx context.Context, id, message string, updatedAt time.Time) error {
	_, err := s.db.ExecContext(ctx, "UPDATE documents SET status = ?, error_message = ?, updated_at = ? WHERE id = ?", StatusFailed, message, updatedAt.UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return fmt.Errorf("mark document failed: %w", err)
	}
	return nil
}

var _ Store = (*SQLiteStore)(nil)
