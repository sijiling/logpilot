package diagnosis

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	einoagent "github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/schema"

	logagent "github.com/sijiling/logpilot/internal/agent"
	"github.com/sijiling/logpilot/internal/session"
)

// Runner is the subset of Eino Agent behavior used by the service.
type Runner interface {
	Generate(ctx context.Context, input []*schema.Message, opts ...einoagent.AgentOption) (*schema.Message, error)
}

type Repository interface {
	Create(context.Context, session.Session) error
	AppendMessage(context.Context, session.Message) error
	SaveDiagnosis(context.Context, session.Diagnosis, []session.ToolRun) error
}

type AgentFactory func(context.Context) (Runner, *logagent.TraceRecorder, error)

// Service executes and persists diagnoses.
type Service struct {
	repository Repository
	factory    AgentFactory
	newID      func() string
	now        func() time.Time
}

// NewService creates the diagnosis application service.
func NewService(repository Repository, factory AgentFactory, newID func() string, now func() time.Time) *Service {
	return &Service{repository: repository, factory: factory, newID: newID, now: now}
}

// Diagnose executes a bounded Eino Agent run.
func (s *Service) Diagnose(ctx context.Context, request Request) (Response, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	sessionID, err := s.prepareSession(ctx, request)
	if err != nil {
		return Response{}, err
	}
	runner, recorder, err := s.factory(ctx)
	if err != nil {
		return Response{}, fmt.Errorf("create agent: %w", err)
	}
	message, err := runner.Generate(ctx, []*schema.Message{schema.UserMessage(buildPrompt(request))})
	if err != nil {
		return Response{}, fmt.Errorf("run agent: %w", err)
	}
	return s.persist(ctx, sessionID, message.Content, recorder.Snapshot())
}

// Stream emits protocol events while executing a diagnosis.
// The Eino HTTP adapter streams model output; this application fallback emits
// the final answer as one delta while preserving the public SSE contract.
func (s *Service) Stream(ctx context.Context, request Request, emit func(Event) error) error {
	response, err := s.Diagnose(ctx, request)
	if err != nil {
		return err
	}
	if err := emit(Event{Type: "session", Data: map[string]string{"session_id": response.SessionID}}); err != nil {
		return err
	}
	for _, call := range response.ToolCalls {
		if err := emit(Event{Type: "tool_result", Data: call}); err != nil {
			return err
		}
	}
	if err := emit(Event{Type: "answer", Data: map[string]string{"delta": response.Summary}}); err != nil {
		return err
	}
	return emit(Event{Type: "done", Data: map[string]any{"diagnosis_id": response.DiagnosisID, "rounds": response.Rounds}})
}

func (s *Service) prepareSession(ctx context.Context, request Request) (string, error) {
	sessionID := request.SessionID
	now := s.now().UTC()
	if sessionID == "" {
		sessionID = s.newID()
		title := truncateTitle(request.Question, 80)
		if err := s.repository.Create(ctx, session.Session{ID: sessionID, Title: title, CreatedAt: now, UpdatedAt: now}); err != nil {
			return "", fmt.Errorf("create session: %w", err)
		}
	}
	if err := s.repository.AppendMessage(ctx, session.Message{ID: s.newID(), SessionID: sessionID, Role: session.RoleUser, Content: buildPrompt(request), CreatedAt: now}); err != nil {
		return "", fmt.Errorf("save user message: %w", err)
	}
	return sessionID, nil
}

func (s *Service) persist(ctx context.Context, sessionID, summary string, traces []logagent.Trace) (Response, error) {
	now := s.now().UTC()
	response := Response{SessionID: sessionID, DiagnosisID: s.newID(), Summary: summary, Causes: []string{}, Steps: []string{}, Warnings: []string{"diagnosis is advisory; no repair was executed"}, Evidence: []Evidence{}, Rounds: max(1, len(traces)+1), CreatedAt: now}
	runs := make([]session.ToolRun, len(traces))
	for index, trace := range traces {
		response.ToolCalls = append(response.ToolCalls, ToolCall{Name: trace.Name, Status: trace.Status, DurationMS: trace.Duration.Milliseconds()})
		runs[index] = session.ToolRun{ID: s.newID(), DiagnosisID: response.DiagnosisID, Round: index + 1, ToolName: trace.Name, ArgumentSummary: trace.ArgumentHash, ResultSummary: trace.ResultSummary, Status: session.ToolStatus(trace.Status), DurationMS: trace.Duration.Milliseconds(), CreatedAt: now}
	}
	encoded, err := json.Marshal(response)
	if err != nil {
		return Response{}, fmt.Errorf("encode diagnosis: %w", err)
	}
	if err := s.repository.AppendMessage(ctx, session.Message{ID: s.newID(), SessionID: sessionID, Role: session.RoleAssistant, Content: summary, CreatedAt: now}); err != nil {
		return Response{}, fmt.Errorf("save assistant message: %w", err)
	}
	if err := s.repository.SaveDiagnosis(ctx, session.Diagnosis{ID: response.DiagnosisID, SessionID: sessionID, SummaryJSON: string(encoded), Rounds: response.Rounds, Status: session.DiagnosisSucceeded, CreatedAt: now}, runs); err != nil {
		return Response{}, fmt.Errorf("save diagnosis: %w", err)
	}
	return response, nil
}

func buildPrompt(request Request) string {
	parts := []string{"Question: " + strings.TrimSpace(request.Question)}
	if request.Logs != "" {
		parts = append(parts, "Logs:\n"+request.Logs)
	}
	if request.Service != "" {
		parts = append(parts, "Service: "+request.Service)
	}
	return strings.Join(parts, "\n\n")
}

func truncateTitle(value string, limit int) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit])
}
