package document

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestServiceIndexReturnsExistingDocument(t *testing.T) {
	store := &fakeStore{existing: &Record{ID: "existing", Status: StatusReady, ChunkCount: 2}}
	embedder := &fakeEmbedder{}
	vectors := &fakeVectorStore{}
	service := NewService(store, embedder, vectors, func() string { return "new" }, fixedClock)

	result, err := service.Index(context.Background(), "Runbook", []byte("same content"))
	if err != nil {
		t.Fatalf("Index() error: %v", err)
	}
	if !result.Duplicated || result.Document.ID != "existing" {
		t.Fatalf("result = %#v, want existing duplicate", result)
	}
	if embedder.calls != 0 || vectors.upsertCalls != 0 {
		t.Fatalf("external calls = embed %d, upsert %d; want zero", embedder.calls, vectors.upsertCalls)
	}
}

func TestServiceIndexCleansVectorsOnFailure(t *testing.T) {
	store := &fakeStore{}
	embedder := &fakeEmbedder{vectors: [][]float32{{1, 2}}}
	vectors := &fakeVectorStore{upsertErr: errors.New("qdrant unavailable")}
	service := NewService(store, embedder, vectors, func() string { return "document-1" }, fixedClock)

	_, err := service.Index(context.Background(), "Runbook", []byte("panic recovery steps"))
	if err == nil {
		t.Fatal("Index() error = nil, want qdrant error")
	}
	if vectors.deletedID != "document-1" {
		t.Errorf("deleted document = %q, want document-1", vectors.deletedID)
	}
	if store.failedID != "document-1" || store.failedMessage == "" {
		t.Errorf("failed state = (%q, %q), want document-1 and message", store.failedID, store.failedMessage)
	}
}

type fakeStore struct {
	existing      *Record
	created       Record
	readyID       string
	failedID      string
	failedMessage string
}

func (s *fakeStore) FindByHash(context.Context, string) (Record, bool, error) {
	if s.existing == nil {
		return Record{}, false, nil
	}
	return *s.existing, true, nil
}
func (s *fakeStore) Create(_ context.Context, record Record) error { s.created = record; return nil }
func (s *fakeStore) MarkReady(_ context.Context, id string, _ int, _ time.Time) error {
	s.readyID = id
	return nil
}
func (s *fakeStore) MarkFailed(_ context.Context, id, message string, _ time.Time) error {
	s.failedID, s.failedMessage = id, message
	return nil
}

type fakeEmbedder struct {
	calls   int
	vectors [][]float32
}

func (e *fakeEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	e.calls++
	if e.vectors != nil {
		return e.vectors, nil
	}
	result := make([][]float32, len(texts))
	for index := range result {
		result[index] = []float32{1, 2}
	}
	return result, nil
}

type fakeVectorStore struct {
	upsertCalls int
	upsertErr   error
	deletedID   string
}

func (s *fakeVectorStore) Upsert(context.Context, []VectorChunk) error {
	s.upsertCalls++
	return s.upsertErr
}
func (*fakeVectorStore) Search(context.Context, []float32, int, float32) ([]Match, error) {
	return nil, nil
}
func (s *fakeVectorStore) DeleteDocument(_ context.Context, id string) error {
	s.deletedID = id
	return nil
}

func fixedClock() time.Time { return time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC) }
