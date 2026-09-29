package server

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sijiling/logpilot/internal/config"
)

func TestServerHealth(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := New(config.Config{HTTPAddr: ":0"}, logger)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	var response map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response["status"] != "ok" {
		t.Errorf("status body = %q, want ok", response["status"])
	}
	if recorder.Header().Get("X-Request-ID") == "" {
		t.Error("X-Request-ID header is empty")
	}
}

func TestServerTimeouts(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := New(config.Config{HTTPAddr: ":0"}, logger)
	httpServer := server.HTTPServer()

	if httpServer.ReadHeaderTimeout != 5*time.Second {
		t.Errorf("ReadHeaderTimeout = %s, want 5s", httpServer.ReadHeaderTimeout)
	}
	if httpServer.ReadTimeout != 10*time.Second {
		t.Errorf("ReadTimeout = %s, want 10s", httpServer.ReadTimeout)
	}
	if httpServer.WriteTimeout != 65*time.Second {
		t.Errorf("WriteTimeout = %s, want 65s", httpServer.WriteTimeout)
	}
	if httpServer.IdleTimeout != 60*time.Second {
		t.Errorf("IdleTimeout = %s, want 60s", httpServer.IdleTimeout)
	}
}
