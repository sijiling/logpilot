package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"github.com/sijiling/logpilot/internal/document"
)

// KnowledgeSearch retrieves internal runbook evidence.
type KnowledgeSearch struct {
	embedder document.Embedder
	vectors  document.VectorStore
}

// NewKnowledgeSearch creates the RAG retrieval tool.
func NewKnowledgeSearch(embedder document.Embedder, vectors document.VectorStore) *KnowledgeSearch {
	return &KnowledgeSearch{embedder: embedder, vectors: vectors}
}

func (*KnowledgeSearch) Info(context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: "search_knowledge", Desc: "Search internal operational runbooks for evidence relevant to a diagnosis.", ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
		"question": {Type: schema.String, Desc: "Question to search for", Required: true},
	})}, nil
}

func (k *KnowledgeSearch) InvokableRun(ctx context.Context, arguments string, _ ...tool.Option) (string, error) {
	var input struct {
		Question string `json:"question"`
	}
	if err := json.Unmarshal([]byte(arguments), &input); err != nil {
		return "", fmt.Errorf("search knowledge: decode arguments: %w", err)
	}
	if strings.TrimSpace(input.Question) == "" {
		return "", fmt.Errorf("search knowledge: question is required")
	}
	vectors, err := k.embedder.Embed(ctx, []string{input.Question})
	if err != nil {
		return "", fmt.Errorf("search knowledge: embed question: %w", err)
	}
	if len(vectors) != 1 {
		return "", fmt.Errorf("search knowledge: embedder returned %d vectors", len(vectors))
	}
	matches, err := k.vectors.Search(ctx, vectors[0], 3, 0.55)
	if err != nil {
		return "", fmt.Errorf("search knowledge: query vectors: %w", err)
	}
	encoded, err := json.Marshal(matches)
	if err != nil {
		return "", fmt.Errorf("search knowledge: encode result: %w", err)
	}
	return string(encoded), nil
}

var _ tool.InvokableTool = (*KnowledgeSearch)(nil)
