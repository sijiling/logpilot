package embedding

import (
	"context"
	"math"
	"testing"
)

func TestHashEmbedderProducesDeterministicNormalizedVectors(t *testing.T) {
	embedder, err := NewHash(384)
	if err != nil {
		t.Fatalf("NewHash() error: %v", err)
	}
	vectors, err := embedder.Embed(context.Background(), []string{"Go panic nil pointer", "Go panic nil pointer"})
	if err != nil {
		t.Fatalf("Embed() error: %v", err)
	}
	if len(vectors) != 2 || len(vectors[0]) != 384 {
		t.Fatalf("shape = %d x %d", len(vectors), len(vectors[0]))
	}
	var norm float64
	for index, value := range vectors[0] {
		if value != vectors[1][index] {
			t.Fatalf("vectors differ at %d", index)
		}
		norm += float64(value * value)
	}
	if math.Abs(math.Sqrt(norm)-1) > 0.0001 {
		t.Fatalf("norm = %f, want 1", math.Sqrt(norm))
	}
}

func TestHashEmbedderRejectsInvalidDimension(t *testing.T) {
	if _, err := NewHash(0); err == nil {
		t.Fatal("NewHash() error = nil")
	}
}
