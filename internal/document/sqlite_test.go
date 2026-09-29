package document

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/sijiling/logpilot/internal/session"
)

func TestSQLiteStoreLifecycle(t *testing.T) {
	repository, err := session.Open(filepath.Join(t.TempDir(), "logpilot.db"))
	if err != nil {
		t.Fatalf("session.Open() error: %v", err)
	}
	defer repository.Close()
	store := NewSQLiteStore(repository.DB())
	now := time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)
	record := Record{ID: "doc-1", Title: "Runbook", ContentHash: "sha256:x", Status: StatusProcessing, CreatedAt: now, UpdatedAt: now}
	if err := store.Create(context.Background(), record); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	found, ok, err := store.FindByHash(context.Background(), record.ContentHash)
	if err != nil || !ok || found.ID != record.ID {
		t.Fatalf("FindByHash() = %#v %v %v", found, ok, err)
	}
	if err := store.MarkReady(context.Background(), record.ID, 3, now.Add(time.Second)); err != nil {
		t.Fatalf("MarkReady() error: %v", err)
	}
	found, _, _ = store.FindByHash(context.Background(), record.ContentHash)
	if found.Status != StatusReady || found.ChunkCount != 3 {
		t.Fatalf("ready record = %#v", found)
	}
}
