package agent

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	diagtools "github.com/sijiling/logpilot/internal/tools"
)

func TestNewAgentAddsSystemPrompt(t *testing.T) {
	fake := &fakeChatModel{}
	agent, err := New(context.Background(), fake, []tool.BaseTool{diagtools.NewLogAnalyzer()}, NewTraceRecorder())
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	message, err := agent.Generate(context.Background(), []*schema.Message{schema.UserMessage("diagnose")})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if message.Content != "final diagnosis" {
		t.Fatalf("content = %q", message.Content)
	}
	if !fake.bound {
		t.Fatal("WithTools was not called")
	}
	if len(fake.input) < 2 || fake.input[0].Role != schema.System {
		t.Fatalf("input does not start with system prompt: %#v", fake.input)
	}
}

type fakeChatModel struct {
	bound bool
	input []*schema.Message
}

func (f *fakeChatModel) WithTools([]*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	f.bound = true
	return f, nil
}
func (f *fakeChatModel) Generate(_ context.Context, input []*schema.Message, _ ...model.Option) (*schema.Message, error) {
	f.input = input
	return schema.AssistantMessage("final diagnosis", nil), nil
}
func (f *fakeChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	message, err := f.Generate(ctx, input, opts...)
	if err != nil {
		return nil, err
	}
	return schema.StreamReaderFromArray([]*schema.Message{message}), nil
}

var _ model.ToolCallingChatModel = (*fakeChatModel)(nil)
