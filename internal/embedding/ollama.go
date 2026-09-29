package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const maxEmbeddingResponse = 8 << 20

// Ollama generates embeddings through Ollama's HTTP API.
type Ollama struct {
	client    *http.Client
	baseURL   string
	model     string
	dimension int
}

// NewOllama validates and creates an Ollama embedding client.
func NewOllama(client *http.Client, baseURL, model string, dimension int) (*Ollama, error) {
	parsed, err := url.ParseRequestURI(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("embedding: invalid ollama base url")
	}
	if client == nil || strings.TrimSpace(model) == "" || dimension <= 0 {
		return nil, fmt.Errorf("embedding: invalid ollama configuration")
	}
	return &Ollama{client: client, baseURL: strings.TrimRight(baseURL, "/"), model: model, dimension: dimension}, nil
}

// Embed returns one vector for each input text.
func (o *Ollama) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	payload, err := json.Marshal(struct {
		Model string   `json:"model"`
		Input []string `json:"input"`
	}{Model: o.model, Input: texts})
	if err != nil {
		return nil, fmt.Errorf("embedding: encode request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/api/embed", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("embedding: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedding: execute request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("embedding: unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var response struct {
		Embeddings [][]float32 `json:"embeddings"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxEmbeddingResponse)).Decode(&response); err != nil {
		return nil, fmt.Errorf("embedding: decode response: %w", err)
	}
	if len(response.Embeddings) != len(texts) {
		return nil, fmt.Errorf("embedding: got %d vectors for %d texts", len(response.Embeddings), len(texts))
	}
	for _, vector := range response.Embeddings {
		if len(vector) != o.dimension {
			return nil, fmt.Errorf("embedding: vector dimension %d, want %d", len(vector), o.dimension)
		}
	}
	return response.Embeddings, nil
}
