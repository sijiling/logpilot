package session

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestSQLiteRepository_CreateAppendAndFind(t *testing.T) {
	repository := openTestRepository(t)
	ctx := context.Background()
	createdAt := time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)
	wantSession := Session{ID: "session-1", Title: "panic diagnosis", CreatedAt: createdAt, UpdatedAt: createdAt}

	if err := repository.Create(ctx, wantSession); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	messages := []Message{
		{ID: "message-2", SessionID: wantSession.ID, Role: RoleAssistant, Content: "second", CreatedAt: createdAt.Add(2 * time.Second)},
		{ID: "message-1", SessionID: wantSession.ID, Role: RoleUser, Content: "first", CreatedAt: createdAt.Add(time.Second)},
	}
	for _, message := range messages {
		if err := repository.AppendMessage(ctx, message); err != nil {
			t.Fatalf("AppendMessage() error: %v", err)
		}
	}
	wantSession.UpdatedAt = messages[0].CreatedAt

	gotSession, gotMessages, err := repository.Find(ctx, wantSession.ID)
	if err != nil {
		t.Fatalf("Find() error: %v", err)
	}
	if gotSession != wantSession {
		t.Errorf("session = %#v, want %#v", gotSession, wantSession)
	}
	if len(gotMessages) != 2 {
		t.Fatalf("messages length = %d, want 2", len(gotMessages))
	}
	if gotMessages[0].ID != "message-1" || gotMessages[1].ID != "message-2" {
		t.Errorf("message order = [%s %s], want [message-1 message-2]", gotMessages[0].ID, gotMessages[1].ID)
	}
}

func TestSQLiteRepository_FindMissingSession(t *testing.T) {
	repository := openTestRepository(t)

	_, _, err := repository.Find(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Find() error = %v, want ErrNotFound", err)
	}
}

func TestSQLiteRepository_SaveDiagnosisRollsBack(t *testing.T) {
	repository := openTestRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)
	session := Session{ID: "session-1", Title: "rollback", CreatedAt: now, UpdatedAt: now}
	if err := repository.Create(ctx, session); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	diagnosis := Diagnosis{ID: "diagnosis-1", SessionID: session.ID, SummaryJSON: `{"summary":"panic"}`, Rounds: 2, Status: DiagnosisSucceeded, CreatedAt: now}
	runs := []ToolRun{
		{ID: "tool-1", DiagnosisID: diagnosis.ID, Round: 1, ToolName: "analyze_log", Status: ToolSucceeded, CreatedAt: now},
		{ID: "tool-1", DiagnosisID: diagnosis.ID, Round: 2, ToolName: "search_knowledge", Status: ToolSucceeded, CreatedAt: now},
	}

	if err := repository.SaveDiagnosis(ctx, diagnosis, runs); err == nil {
		t.Fatal("SaveDiagnosis() error = nil, want duplicate tool id error")
	}

	var count int
	if err := repository.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM diagnoses WHERE id = ?", diagnosis.ID).Scan(&count); err != nil {
		t.Fatalf("count diagnosis: %v", err)
	}
	if count != 0 {
		t.Errorf("diagnosis rows = %d, want 0 after rollback", count)
	}
}

func openTestRepository(t *testing.T) *SQLiteRepository {
	t.Helper()
	repository, err := Open(filepath.Join(t.TempDir(), "logpilot.db"))
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	t.Cleanup(func() {
		if err := repository.Close(); err != nil {
			t.Errorf("Close() error: %v", err)
		}
	})
	return repository
}
