package document

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

const (
	defaultChunkTarget  = 800
	defaultChunkOverlap = 120
	embedBatchSize      = 16
)

// Store persists document indexing state.
type Store interface {
	FindByHash(ctx context.Context, hash string) (Record, bool, error)
	Create(ctx context.Context, record Record) error
	MarkReady(ctx context.Context, id string, chunkCount int, updatedAt time.Time) error
	MarkFailed(ctx context.Context, id, message string, updatedAt time.Time) error
}

// Embedder converts text into vectors.
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

// VectorStore persists and searches document vectors.
type VectorStore interface {
	Upsert(ctx context.Context, chunks []VectorChunk) error
	Search(ctx context.Context, vector []float32, limit int, minScore float32) ([]Match, error)
	DeleteDocument(ctx context.Context, documentID string) error
}

// VectorChunk is a document chunk paired with its embedding.
type VectorChunk struct {
	DocumentID  string
	Title       string
	Index       int
	Content     string
	ContentHash string
	Vector      []float32
}

// Match is a scored vector-search result.
type Match struct {
	DocumentID string
	Title      string
	Index      int
	Content    string
	Score      float32
}

// IndexResult describes an indexing operation.
type IndexResult struct {
	Document   Record
	Duplicated bool
}

// Service coordinates document persistence, embedding, and vector indexing.
type Service struct {
	store    Store
	embedder Embedder
	vectors  VectorStore
	newID    func() string
	now      func() time.Time
}

// NewService creates a document indexing service.
func NewService(store Store, embedder Embedder, vectors VectorStore, newID func() string, now func() time.Time) *Service {
	return &Service{store: store, embedder: embedder, vectors: vectors, newID: newID, now: now}
}

// Index creates a searchable document or returns the existing identical one.
func (s *Service) Index(ctx context.Context, title string, content []byte) (IndexResult, error) {
	hashBytes := sha256.Sum256(content)
	hash := "sha256:" + hex.EncodeToString(hashBytes[:])
	existing, found, err := s.store.FindByHash(ctx, hash)
	if err != nil {
		return IndexResult{}, fmt.Errorf("find duplicate document: %w", err)
	}
	if found {
		return IndexResult{Document: existing, Duplicated: true}, nil
	}

	now := s.now().UTC()
	record := Record{
		ID: s.newID(), Title: title, ContentHash: hash, Status: StatusProcessing,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.store.Create(ctx, record); err != nil {
		return IndexResult{}, fmt.Errorf("create document: %w", err)
	}

	chunks := SplitMarkdown(string(content), defaultChunkTarget, defaultChunkOverlap)
	vectorChunks, err := s.embed(ctx, record, chunks)
	if err != nil {
		return IndexResult{}, s.fail(ctx, record.ID, err)
	}
	if err := s.vectors.Upsert(ctx, vectorChunks); err != nil {
		return IndexResult{}, s.fail(ctx, record.ID, fmt.Errorf("upsert document vectors: %w", err))
	}
	if err := s.store.MarkReady(ctx, record.ID, len(chunks), s.now().UTC()); err != nil {
		return IndexResult{}, s.fail(ctx, record.ID, fmt.Errorf("mark document ready: %w", err))
	}
	record.Status = StatusReady
	record.ChunkCount = len(chunks)
	record.UpdatedAt = s.now().UTC()
	return IndexResult{Document: record}, nil
}

func (s *Service) embed(ctx context.Context, record Record, chunks []Chunk) ([]VectorChunk, error) {
	result := make([]VectorChunk, 0, len(chunks))
	for start := 0; start < len(chunks); start += embedBatchSize {
		end := min(start+embedBatchSize, len(chunks))
		texts := make([]string, end-start)
		for index := start; index < end; index++ {
			texts[index-start] = chunks[index].Content
		}
		vectors, err := s.embedder.Embed(ctx, texts)
		if err != nil {
			return nil, fmt.Errorf("embed document chunks: %w", err)
		}
		if len(vectors) != len(texts) {
			return nil, fmt.Errorf("embed document chunks: got %d vectors for %d texts", len(vectors), len(texts))
		}
		for index, vector := range vectors {
			chunk := chunks[start+index]
			result = append(result, VectorChunk{
				DocumentID: record.ID, Title: record.Title, Index: chunk.Index,
				Content: chunk.Content, ContentHash: record.ContentHash, Vector: vector,
			})
		}
	}
	return result, nil
}

func (s *Service) fail(ctx context.Context, id string, cause error) error {
	deleteErr := s.vectors.DeleteDocument(ctx, id)
	markErr := s.store.MarkFailed(ctx, id, cause.Error(), s.now().UTC())
	return errors.Join(cause, deleteErr, markErr)
}
