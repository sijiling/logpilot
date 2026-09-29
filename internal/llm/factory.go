// Package llm constructs model adapters used by LogPilot.
package llm

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"

	"github.com/sijiling/logpilot/internal/config"
)

// NewChatModel creates an Eino OpenAI-compatible tool-calling model.
func NewChatModel(ctx context.Context, cfg config.Config) (model.ToolCallingChatModel, error) {
	if cfg.LLMBaseURL == "" || cfg.LLMModel == "" {
		return nil, fmt.Errorf("llm: base url and model are required")
	}
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: cfg.LLMBaseURL,
		APIKey:  cfg.LLMAPIKey,
		Model:   cfg.LLMModel,
		Timeout: 60 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("llm: create openai chat model: %w", err)
	}
	return chatModel, nil
}
