package agent

import (
	"context"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

// New assembles the Eino ReAct Agent with bounded steps and trace middleware.
func New(ctx context.Context, chatModel model.ToolCallingChatModel, tools []tool.BaseTool, recorder *TraceRecorder) (*react.Agent, error) {
	middlewares := make([]compose.ToolMiddleware, 0, 1)
	if recorder != nil {
		middlewares = append(middlewares, recorder.Middleware())
	}
	return react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: chatModel,
		ToolsConfig:      compose.ToolsNodeConfig{Tools: tools, ToolCallMiddlewares: middlewares},
		MaxStep:          6,
		MessageModifier: func(_ context.Context, input []*schema.Message) []*schema.Message {
			messages := make([]*schema.Message, 0, len(input)+1)
			messages = append(messages, schema.SystemMessage(systemPrompt))
			return append(messages, input...)
		},
	})
}
