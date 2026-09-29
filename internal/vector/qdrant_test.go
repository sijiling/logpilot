package vector

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sijiling/logpilot/internal/document"
)

func TestQdrantUpsertAndSearch(t *testing.T) {
	var upserted bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/collections/logpilot_documents/points":
			upserted = true
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode upsert: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
		case "/collections/logpilot_documents/points/search":
			response := map[string]any{"result": []any{map[string]any{
				"score": 0.91, "payload": map[string]any{"document_id": "doc-1", "title": "Runbook", "chunk_index": 0, "content": "panic steps"},
			}}}
			_ = json.NewEncoder(w).Encode(response)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	store, err := NewQdrant(server.Client(), server.URL, "logpilot_documents")
	if err != nil {
		t.Fatalf("NewQdrant() error: %v", err)
	}
	err = store.Upsert(context.Background(), []document.VectorChunk{{DocumentID: "doc-1", Title: "Runbook", Index: 0, Content: "panic steps", ContentHash: "sha256:x", Vector: []float32{1, 2}}})
	if err != nil {
		t.Fatalf("Upsert() error: %v", err)
	}
	if !upserted {
		t.Fatal("upsert endpoint was not called")
	}
	matches, err := store.Search(context.Background(), []float32{1, 2}, 3, 0.55)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}
	if len(matches) != 1 || matches[0].DocumentID != "doc-1" || matches[0].Score < 0.9 {
		t.Fatalf("matches = %#v", matches)
	}
}

func TestQdrantReturnsStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "down", http.StatusServiceUnavailable) }))
	defer server.Close()
	store, _ := NewQdrant(server.Client(), server.URL, "documents")
	if _, err := store.Search(context.Background(), []float32{1}, 3, 0.5); err == nil {
		t.Fatal("Search() error = nil")
	}
}
