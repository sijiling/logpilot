package session

import "time"

// Role identifies the source of a conversation message.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// DiagnosisStatus is the terminal state of a diagnosis.
type DiagnosisStatus string

const (
	DiagnosisSucceeded DiagnosisStatus = "succeeded"
	DiagnosisFailed    DiagnosisStatus = "failed"
	DiagnosisCancelled DiagnosisStatus = "cancelled"
)

// ToolStatus is the result state of a tool execution.
type ToolStatus string

const (
	ToolSucceeded ToolStatus = "succeeded"
	ToolFailed    ToolStatus = "failed"
)

// Session groups user and assistant messages.
type Session struct {
	ID        string
	Title     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Message is one ordered entry in a session.
type Message struct {
	ID        string
	SessionID string
	Role      Role
	Content   string
	CreatedAt time.Time
}

// Diagnosis stores the final structured result of an Agent run.
type Diagnosis struct {
	ID          string
	SessionID   string
	SummaryJSON string
	Rounds      int
	Status      DiagnosisStatus
	CreatedAt   time.Time
}

// ToolRun stores a redacted record of one Agent tool execution.
type ToolRun struct {
	ID              string
	DiagnosisID     string
	Round           int
	ToolName        string
	ArgumentSummary string
	ResultSummary   string
	Status          ToolStatus
	DurationMS      int64
	CreatedAt       time.Time
}
