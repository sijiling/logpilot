package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthCheckUsesAllowlistAndLimitsBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", 6000)))
	}))
	defer server.Close()
	tool := NewHealthCheck(server.Client(), map[string]string{"video": server.URL})
	result, err := tool.InvokableRun(context.Background(), `{"service":"video"}`)
	if err != nil {
		t.Fatalf("InvokableRun() error: %v", err)
	}
	if len(result) > 5000 {
		t.Fatalf("result length = %d, body limit not applied", len(result))
	}
	if !strings.Contains(result, `"status_code":200`) {
		t.Fatalf("result = %s", result)
	}
}

func TestHealthCheckRejectsUnknownService(t *testing.T) {
	tool := NewHealthCheck(http.DefaultClient, map[string]string{})
	if _, err := tool.InvokableRun(context.Background(), `{"service":"metadata"}`); err == nil {
		t.Fatal("error = nil")
	}
}
