package llm

import (
	"context"
	"testing"

	"github.com/sijiling/logpilot/internal/config"
)

func TestNewChatModel(t *testing.T) {
	chatModel, err := NewChatModel(context.Background(), config.Config{LLMBaseURL: "http://localhost:11434/v1", LLMAPIKey: "local", LLMModel: "qwen3"})
	if err != nil {
		t.Fatalf("NewChatModel() error: %v", err)
	}
	if chatModel == nil {
		t.Fatal("chatModel = nil")
	}
}
