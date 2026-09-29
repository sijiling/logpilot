package vector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/sijiling/logpilot/internal/document"
)

const maxQdrantResponse = 8 << 20

// Qdrant stores and searches knowledge-document vectors.
type Qdrant struct {
	client     *http.Client
	baseURL    string
	collection string
}

// NewQdrant validates and creates a Qdrant adapter.
func NewQdrant(client *http.Client, baseURL, collection string) (*Qdrant, error) {
	parsed, err := url.ParseRequestURI(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || client == nil || strings.TrimSpace(collection) == "" {
		return nil, fmt.Errorf("vector: invalid qdrant configuration")
	}
	return &Qdrant{client: client, baseURL: strings.TrimRight(baseURL, "/"), collection: collection}, nil
}

// Upsert writes chunks with deterministic point IDs, making retries idempotent.
func (q *Qdrant) Upsert(ctx context.Context, chunks []document.VectorChunk) error {
	type point struct {
		ID      string         `json:"id"`
		Vector  []float32      `json:"vector"`
		Payload map[string]any `json:"payload"`
	}
	points := make([]point, len(chunks))
	for index, chunk := range chunks {
		points[index] = point{
			ID:     uuid.NewSHA1(uuid.NameSpaceURL, []byte(chunk.DocumentID+":"+strconv.Itoa(chunk.Index))).String(),
			Vector: chunk.Vector,
			Payload: map[string]any{
				"document_id": chunk.DocumentID, "title": chunk.Title, "chunk_index": chunk.Index,
				"content": chunk.Content, "content_hash": chunk.ContentHash,
			},
		}
	}
	return q.doJSON(ctx, http.MethodPut, "/collections/"+q.collection+"/points?wait=true", map[string]any{"points": points}, nil)
}

// Search returns the highest-scoring chunks above the requested threshold.
func (q *Qdrant) Search(ctx context.Context, vector []float32, limit int, minScore float32) ([]document.Match, error) {
	request := map[string]any{"vector": vector, "limit": limit, "score_threshold": minScore, "with_payload": true}
	var response struct {
		Result []struct {
			Score   float32 `json:"score"`
			Payload struct {
				DocumentID string `json:"document_id"`
				Title      string `json:"title"`
				ChunkIndex int    `json:"chunk_index"`
				Content    string `json:"content"`
			} `json:"payload"`
		} `json:"result"`
	}
	if err := q.doJSON(ctx, http.MethodPost, "/collections/"+q.collection+"/points/search", request, &response); err != nil {
		return nil, err
	}
	matches := make([]document.Match, len(response.Result))
	for index, result := range response.Result {
		matches[index] = document.Match{DocumentID: result.Payload.DocumentID, Title: result.Payload.Title, Index: result.Payload.ChunkIndex, Content: result.Payload.Content, Score: result.Score}
	}
	return matches, nil
}

// DeleteDocument removes every point belonging to a document.
func (q *Qdrant) DeleteDocument(ctx context.Context, documentID string) error {
	filter := map[string]any{"filter": map[string]any{"must": []any{map[string]any{"key": "document_id", "match": map[string]any{"value": documentID}}}}}
	return q.doJSON(ctx, http.MethodPost, "/collections/"+q.collection+"/points/delete?wait=true", filter, nil)
}

// Ready checks whether the configured collection can be reached.
func (q *Qdrant) Ready(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, q.baseURL+"/collections/"+q.collection, nil)
	if err != nil {
		return fmt.Errorf("vector: create readiness request: %w", err)
	}
	resp, err := q.client.Do(req)
	if err != nil {
		return fmt.Errorf("vector: readiness request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("vector: readiness status %d", resp.StatusCode)
	}
	return nil
}

func (q *Qdrant) doJSON(ctx context.Context, method, path string, input, output any) error {
	payload, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("vector: encode request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, method, q.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("vector: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := q.client.Do(req)
	if err != nil {
		return fmt.Errorf("vector: execute request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("vector: unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if output == nil {
		return nil
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxQdrantResponse)).Decode(output); err != nil {
		return fmt.Errorf("vector: decode response: %w", err)
	}
	return nil
}
