package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sijiling/logpilot/internal/diagnosis"
	"github.com/sijiling/logpilot/internal/document"
	"github.com/sijiling/logpilot/internal/session"
)

func TestRouterDiagnosisAndSession(t *testing.T) {
	now := time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)
	diagnoser := &fakeDiagnoser{response: diagnosis.Response{SessionID: "session-1", DiagnosisID: "diagnosis-1", Summary: "nil pointer", Rounds: 2, CreatedAt: now}}
	sessions := &fakeSessions{record: session.Session{ID: "session-1", CreatedAt: now, UpdatedAt: now}, messages: []session.Message{{ID: "m1", SessionID: "session-1", Role: session.RoleUser, Content: "panic", CreatedAt: now}}}
	router := NewRouter(Dependencies{Documents: &fakeDocuments{}, Diagnoses: diagnoser, Sessions: sessions})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/diagnoses", strings.NewReader(`{"question":"why panic?"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("diagnosis status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"summary":"nil pointer"`) {
		t.Fatalf("body = %s", recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/api/v1/sessions/session-1", nil)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"role":"user"`) {
		t.Fatalf("session response = %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestRouterRejectsEmptyQuestion(t *testing.T) {
	router := NewRouter(Dependencies{Documents: &fakeDocuments{}, Diagnoses: &fakeDiagnoser{}, Sessions: &fakeSessions{}})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/diagnoses", strings.NewReader(`{"question":""}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), "INVALID_ARGUMENT") {
		t.Fatalf("response = %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestRouterStreamsSSEInOrder(t *testing.T) {
	diagnoser := &fakeDiagnoser{events: []diagnosis.Event{{Type: "session", Data: map[string]string{"session_id": "s1"}}, {Type: "answer", Data: map[string]string{"delta": "result"}}, {Type: "done", Data: map[string]string{"diagnosis_id": "d1"}}}}
	router := NewRouter(Dependencies{Documents: &fakeDocuments{}, Diagnoses: diagnoser, Sessions: &fakeSessions{}})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/diagnoses/stream", strings.NewReader(`{"question":"panic"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	body := recorder.Body.String()
	indexes := []int{strings.Index(body, "event: session"), strings.Index(body, "event: answer"), strings.Index(body, "event: done")}
	if indexes[0] < 0 || indexes[0] >= indexes[1] || indexes[1] >= indexes[2] {
		t.Fatalf("event order invalid: %s", body)
	}
}

func TestRouterUploadsMarkdown(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, _ := writer.CreateFormFile("file", "panic.md")
	_, _ = part.Write([]byte("# Panic"))
	_ = writer.WriteField("title", "Panic Runbook")
	_ = writer.Close()
	documents := &fakeDocuments{result: document.IndexResult{Document: document.Record{ID: "doc-1", Title: "Panic Runbook", Status: document.StatusReady, ChunkCount: 1}}}
	router := NewRouter(Dependencies{Documents: documents, Diagnoses: &fakeDiagnoser{}, Sessions: &fakeSessions{}})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/documents", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated || !strings.Contains(recorder.Body.String(), `"document_id":"doc-1"`) {
		t.Fatalf("response = %d %s", recorder.Code, recorder.Body.String())
	}
}

type fakeDocuments struct{ result document.IndexResult }

func (f *fakeDocuments) Index(context.Context, string, []byte) (document.IndexResult, error) {
	return f.result, nil
}

type fakeDiagnoser struct {
	response diagnosis.Response
	events   []diagnosis.Event
}

func (f *fakeDiagnoser) Diagnose(context.Context, diagnosis.Request) (diagnosis.Response, error) {
	return f.response, nil
}
func (f *fakeDiagnoser) Stream(_ context.Context, _ diagnosis.Request, emit func(diagnosis.Event) error) error {
	for _, event := range f.events {
		if err := emit(event); err != nil {
			return err
		}
	}
	return nil
}

type fakeSessions struct {
	record   session.Session
	messages []session.Message
	err      error
}

func (f *fakeSessions) Find(context.Context, string) (session.Session, []session.Message, error) {
	return f.record, f.messages, f.err
}

func decodeJSON(t *testing.T, body []byte, value any) {
	t.Helper()
	if err := json.Unmarshal(body, value); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
}
