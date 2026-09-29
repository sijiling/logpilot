// Package diagnosis coordinates Agent runs and persistence.
package diagnosis

import "time"

// Request is the validated input for one diagnosis.
type Request struct {
	SessionID string `json:"session_id"`
	Question  string `json:"question"`
	Logs      string `json:"logs"`
	Service   string `json:"service"`
}

// Evidence cites a source used in a diagnosis.
type Evidence struct {
	Type       string  `json:"type"`
	Title      string  `json:"title,omitempty"`
	ChunkIndex int     `json:"chunk_index,omitempty"`
	Score      float32 `json:"score,omitempty"`
}

// ToolCall summarizes one business-tool execution.
type ToolCall struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	DurationMS int64  `json:"duration_ms"`
}

// Response is the public diagnosis result.
type Response struct {
	SessionID   string     `json:"session_id"`
	DiagnosisID string     `json:"diagnosis_id"`
	Summary     string     `json:"summary"`
	Causes      []string   `json:"causes"`
	Steps       []string   `json:"steps"`
	Warnings    []string   `json:"warnings"`
	Evidence    []Evidence `json:"evidence"`
	ToolCalls   []ToolCall `json:"tool_calls"`
	Rounds      int        `json:"rounds"`
	CreatedAt   time.Time  `json:"created_at"`
}

// Event is one SSE diagnosis event.
type Event struct {
	Type string
	Data any
}
