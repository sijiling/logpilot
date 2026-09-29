package session

import (
	"context"
	"errors"
)

// ErrNotFound indicates that a requested session does not exist.
var ErrNotFound = errors.New("session: not found")

// Repository persists sessions, messages, diagnoses, and tool traces.
type Repository interface {
	Create(ctx context.Context, session Session) error
	AppendMessage(ctx context.Context, message Message) error
	SaveDiagnosis(ctx context.Context, diagnosis Diagnosis, runs []ToolRun) error
	Find(ctx context.Context, id string) (Session, []Message, error)
}
