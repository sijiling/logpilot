package agent

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/compose"
)

func TestTraceRecorderMiddlewareRecordsToolRun(t *testing.T) {
	recorder := NewTraceRecorder()
	middleware := recorder.Middleware()
	endpoint := middleware.Invokable(func(context.Context, *compose.ToolInput) (*compose.ToolOutput, error) {
		return &compose.ToolOutput{Result: `{"ok":true}`}, nil
	})
	_, err := endpoint(context.Background(), &compose.ToolInput{Name: "analyze_log", Arguments: `{"logs":"secret"}`, CallID: "call-1"})
	if err != nil {
		t.Fatalf("endpoint error: %v", err)
	}
	runs := recorder.Snapshot()
	if len(runs) != 1 || runs[0].Name != "analyze_log" || runs[0].Status != "succeeded" {
		t.Fatalf("runs = %#v", runs)
	}
	if runs[0].ArgumentHash == "" || runs[0].ArgumentHash == `{"logs":"secret"}` {
		t.Fatal("arguments must be recorded only as a hash")
	}
}
