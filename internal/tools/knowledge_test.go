package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/sijiling/logpilot/internal/document"
)

func TestKnowledgeSearchReturnsMatches(t *testing.T) {
	embedder := toolEmbedder{vector: []float32{1, 2}}
	store := &toolVectorStore{matches: []document.Match{{DocumentID: "doc-1", Title: "Panic", Index: 2, Content: "check nil", Score: 0.8}}}
	tool := NewKnowledgeSearch(embedder, store)
	result, err := tool.InvokableRun(context.Background(), `{"question":"how to fix panic"}`)
	if err != nil {
		t.Fatalf("InvokableRun() error: %v", err)
	}
	if !strings.Contains(result, "doc-1") || store.limit != 3 || store.minScore != 0.55 {
		t.Fatalf("result=%s limit=%d score=%v", result, store.limit, store.minScore)
	}
}

type toolEmbedder struct{ vector []float32 }

func (e toolEmbedder) Embed(context.Context, []string) ([][]float32, error) {
	return [][]float32{e.vector}, nil
}

type toolVectorStore struct {
	matches  []document.Match
	limit    int
	minScore float32
}

func (*toolVectorStore) Upsert(context.Context, []document.VectorChunk) error { return nil }
func (s *toolVectorStore) Search(_ context.Context, _ []float32, limit int, minScore float32) ([]document.Match, error) {
	s.limit, s.minScore = limit, minScore
	return s.matches, nil
}
func (*toolVectorStore) DeleteDocument(context.Context, string) error { return nil }
