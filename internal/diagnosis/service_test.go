package diagnosis

import (
	"context"
	"testing"
	"time"

	einoagent "github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/schema"

	logagent "github.com/sijiling/logpilot/internal/agent"
	"github.com/sijiling/logpilot/internal/session"
)

func TestServiceDiagnosePersistsConversation(t *testing.T) {
	repository := &fakeRepository{}
	ids := []string{"session-1", "message-user", "diagnosis-1", "message-assistant"}
	index := 0
	service := NewService(repository, func(context.Context) (Runner, *logagent.TraceRecorder, error) {
		return fakeRunner{}, logagent.NewTraceRecorder(), nil
	}, func() string { id := ids[index]; index++; return id }, func() time.Time { return time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC) })

	response, err := service.Diagnose(context.Background(), Request{Question: "why panic?", Logs: "panic: nil"})
	if err != nil {
		t.Fatalf("Diagnose() error: %v", err)
	}
	if response.SessionID != "session-1" || response.Summary != "check nil pointer" {
		t.Fatalf("response = %#v", response)
	}
	if len(repository.messages) != 2 || repository.diagnosis.ID != "diagnosis-1" {
		t.Fatalf("persisted = %#v %#v", repository.messages, repository.diagnosis)
	}
}

type fakeRunner struct{}

func (fakeRunner) Generate(context.Context, []*schema.Message, ...einoagent.AgentOption) (*schema.Message, error) {
	return schema.AssistantMessage("check nil pointer", nil), nil
}

type fakeRepository struct {
	created   session.Session
	messages  []session.Message
	diagnosis session.Diagnosis
}

func (r *fakeRepository) Create(_ context.Context, value session.Session) error {
	r.created = value
	return nil
}
func (r *fakeRepository) AppendMessage(_ context.Context, value session.Message) error {
	r.messages = append(r.messages, value)
	return nil
}
func (r *fakeRepository) SaveDiagnosis(_ context.Context, value session.Diagnosis, _ []session.ToolRun) error {
	r.diagnosis = value
	return nil
}
