package session

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"github.com/sijiling/logpilot/migrations"
)

// SQLiteRepository stores LogPilot records in SQLite.
type SQLiteRepository struct {
	db *sql.DB
}

// Open creates a SQLite repository and applies the embedded schema migration.
func Open(path string) (*SQLiteRepository, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxIdleTime(time.Minute)

	if _, err := db.ExecContext(context.Background(), migrations.Initial); err != nil {
		closeErr := db.Close()
		return nil, errors.Join(fmt.Errorf("apply database migration: %w", err), closeErr)
	}

	return &SQLiteRepository{db: db}, nil
}

// Close releases the SQLite connection pool.
func (r *SQLiteRepository) Close() error {
	return r.db.Close()
}

// DB exposes the shared connection pool to other SQLite-backed repositories.
func (r *SQLiteRepository) DB() *sql.DB { return r.db }

// Create inserts a new session.
func (r *SQLiteRepository) Create(ctx context.Context, session Session) error {
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO sessions (id, title, created_at, updated_at) VALUES (?, ?, ?, ?)",
		session.ID, session.Title, formatTime(session.CreatedAt), formatTime(session.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

// AppendMessage inserts a message and advances the parent session timestamp.
func (r *SQLiteRepository) AppendMessage(ctx context.Context, message Message) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin append message transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		"INSERT INTO messages (id, session_id, role, content, created_at) VALUES (?, ?, ?, ?, ?)",
		message.ID, message.SessionID, message.Role, message.Content, formatTime(message.CreatedAt),
	); err != nil {
		return fmt.Errorf("insert message: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		"UPDATE sessions SET updated_at = CASE WHEN updated_at < ? THEN ? ELSE updated_at END WHERE id = ?",
		formatTime(message.CreatedAt), formatTime(message.CreatedAt), message.SessionID,
	); err != nil {
		return fmt.Errorf("update session timestamp: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit append message transaction: %w", err)
	}
	return nil
}

// SaveDiagnosis atomically stores a diagnosis and all of its tool runs.
func (r *SQLiteRepository) SaveDiagnosis(ctx context.Context, diagnosis Diagnosis, runs []ToolRun) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin diagnosis transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		"INSERT INTO diagnoses (id, session_id, summary_json, rounds, status, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		diagnosis.ID, diagnosis.SessionID, diagnosis.SummaryJSON, diagnosis.Rounds, diagnosis.Status, formatTime(diagnosis.CreatedAt),
	); err != nil {
		return fmt.Errorf("insert diagnosis: %w", err)
	}
	for _, run := range runs {
		if _, err := tx.ExecContext(ctx, `INSERT INTO tool_runs
            (id, diagnosis_id, round, tool_name, argument_summary, result_summary, status, duration_ms, created_at)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			run.ID, run.DiagnosisID, run.Round, run.ToolName, run.ArgumentSummary, run.ResultSummary,
			run.Status, run.DurationMS, formatTime(run.CreatedAt),
		); err != nil {
			return fmt.Errorf("insert tool run: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit diagnosis transaction: %w", err)
	}
	return nil
}

// Find returns a session and its messages ordered by creation time.
func (r *SQLiteRepository) Find(ctx context.Context, id string) (Session, []Message, error) {
	var record Session
	var createdAt string
	var updatedAt string
	err := r.db.QueryRowContext(ctx,
		"SELECT id, title, created_at, updated_at FROM sessions WHERE id = ?", id,
	).Scan(&record.ID, &record.Title, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, nil, ErrNotFound
	}
	if err != nil {
		return Session{}, nil, fmt.Errorf("query session: %w", err)
	}
	if record.CreatedAt, err = parseTime(createdAt); err != nil {
		return Session{}, nil, fmt.Errorf("parse session created time: %w", err)
	}
	if record.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return Session{}, nil, fmt.Errorf("parse session updated time: %w", err)
	}

	rows, err := r.db.QueryContext(ctx,
		"SELECT id, session_id, role, content, created_at FROM messages WHERE session_id = ? ORDER BY created_at, id", id,
	)
	if err != nil {
		return Session{}, nil, fmt.Errorf("query session messages: %w", err)
	}
	defer rows.Close()

	messages := make([]Message, 0)
	for rows.Next() {
		var message Message
		var messageTime string
		if err := rows.Scan(&message.ID, &message.SessionID, &message.Role, &message.Content, &messageTime); err != nil {
			return Session{}, nil, fmt.Errorf("scan session message: %w", err)
		}
		message.CreatedAt, err = parseTime(messageTime)
		if err != nil {
			return Session{}, nil, fmt.Errorf("parse message created time: %w", err)
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return Session{}, nil, fmt.Errorf("iterate session messages: %w", err)
	}
	return record, messages, nil
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func parseTime(value string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, value)
}
