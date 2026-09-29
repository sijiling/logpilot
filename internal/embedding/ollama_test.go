package embedding

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOllamaEmbed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embed" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		var request struct {
			Model string   `json:"model"`
			Input []string `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request.Model != "nomic-embed-text" || len(request.Input) != 2 {
			t.Fatalf("request = %#v", request)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"embeddings": [][]float32{{1, 2}, {3, 4}}})
	}))
	defer server.Close()

	client, err := NewOllama(&http.Client{Timeout: time.Second}, server.URL, "nomic-embed-text", 2)
	if err != nil {
		t.Fatalf("NewOllama() error: %v", err)
	}
	vectors, err := client.Embed(context.Background(), []string{"one", "two"})
	if err != nil {
		t.Fatalf("Embed() error: %v", err)
	}
	if len(vectors) != 2 || len(vectors[0]) != 2 {
		t.Fatalf("vectors = %#v", vectors)
	}
}

func TestOllamaEmbedRejectsWrongDimension(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"embeddings": [][]float32{{1}}})
	}))
	defer server.Close()
	client, _ := NewOllama(server.Client(), server.URL, "model", 2)
	if _, err := client.Embed(context.Background(), []string{"one"}); err == nil {
		t.Fatal("Embed() error = nil")
	}
}

func TestOllamaEmbedPropagatesCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { time.Sleep(time.Second) }))
	defer server.Close()
	client, _ := NewOllama(server.Client(), server.URL, "model", 2)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.Embed(ctx, []string{"one"}); err == nil {
		t.Fatal("Embed() error = nil")
	}
}
