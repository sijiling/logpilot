package embedding

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"strings"
	"unicode"
)

// Hash is a deterministic feature-hashing embedder for small local runbooks.
type Hash struct{ dimension int }

// NewHash creates a feature-hashing embedder with the requested vector size.
func NewHash(dimension int) (*Hash, error) {
	if dimension <= 0 {
		return nil, fmt.Errorf("embedding: dimension must be positive")
	}
	return &Hash{dimension: dimension}, nil
}

// Embed converts text tokens and rune bigrams into normalized vectors.
func (h *Hash) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	vectors := make([][]float32, len(texts))
	for index, text := range texts {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		vector := make([]float32, h.dimension)
		for _, feature := range hashFeatures(text) {
			hasher := fnv.New32a()
			_, _ = hasher.Write([]byte(feature))
			vector[int(hasher.Sum32()%uint32(h.dimension))]++
		}
		normalize(vector)
		vectors[index] = vector
	}
	return vectors, nil
}

func hashFeatures(text string) []string {
	lower := strings.ToLower(text)
	result := strings.FieldsFunc(lower, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	compact := []rune(strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || unicode.IsPunct(r) {
			return -1
		}
		return unicode.ToLower(r)
	}, text))
	for index := range compact {
		result = append(result, string(compact[index]))
		if index+1 < len(compact) {
			result = append(result, string(compact[index:index+2]))
		}
	}
	return result
}

func normalize(vector []float32) {
	var sum float64
	for _, value := range vector {
		sum += float64(value * value)
	}
	if sum == 0 {
		return
	}
	norm := float32(math.Sqrt(sum))
	for index := range vector {
		vector[index] /= norm
	}
}
