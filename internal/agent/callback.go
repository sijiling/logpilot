package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	"github.com/cloudwego/eino/compose"
)

// Trace records a redacted tool execution for persistence and metrics.
type Trace struct {
	Name          string
	ArgumentHash  string
	ResultSummary string
	Status        string
	Duration      time.Duration
}

// TraceRecorder safely collects Eino tool middleware events.
type TraceRecorder struct {
	mu   sync.Mutex
	runs []Trace
}

// NewTraceRecorder creates an empty recorder.
func NewTraceRecorder() *TraceRecorder { return &TraceRecorder{} }

// Middleware returns Eino middleware that records non-streaming tools.
func (r *TraceRecorder) Middleware() compose.ToolMiddleware {
	return compose.ToolMiddleware{Invokable: func(next compose.InvokableToolEndpoint) compose.InvokableToolEndpoint {
		return func(ctx context.Context, input *compose.ToolInput) (*compose.ToolOutput, error) {
			started := time.Now()
			output, err := next(ctx, input)
			status := "succeeded"
			result := ""
			if err != nil {
				status = "failed"
			} else if output != nil {
				result = truncateTrace(output.Result, 4096)
			}
			hash := sha256.Sum256([]byte(input.Arguments))
			r.mu.Lock()
			r.runs = append(r.runs, Trace{Name: input.Name, ArgumentHash: "sha256:" + hex.EncodeToString(hash[:]), ResultSummary: result, Status: status, Duration: time.Since(started)})
			r.mu.Unlock()
			return output, err
		}
	}}
}

// Snapshot returns an isolated copy of all recorded traces.
func (r *TraceRecorder) Snapshot() []Trace {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Trace(nil), r.runs...)
}

func truncateTrace(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}
