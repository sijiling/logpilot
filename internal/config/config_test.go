package config

import (
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name        string
		env         map[string]string
		wantErr     string
		wantAddr    string
		wantDBPath  string
		wantTargets map[string]string
	}{
		{
			name: "loads defaults",
			env: map[string]string{
				"LOGPILOT_LLM_BASE_URL":    "http://model.example/v1",
				"LOGPILOT_LLM_MODEL":       "test-model",
				"LOGPILOT_EMBED_BASE_URL":  "http://embed.example",
				"LOGPILOT_EMBED_MODEL":     "test-embed",
				"LOGPILOT_EMBED_DIMENSION": "384",
			},
			wantAddr:    ":8080",
			wantDBPath:  "./data/logpilot.db",
			wantTargets: map[string]string{},
		},
		{
			name: "parses service targets",
			env: map[string]string{
				"LOGPILOT_LLM_BASE_URL":    "http://model.example/v1",
				"LOGPILOT_LLM_MODEL":       "test-model",
				"LOGPILOT_EMBED_BASE_URL":  "http://embed.example",
				"LOGPILOT_EMBED_MODEL":     "test-embed",
				"LOGPILOT_EMBED_DIMENSION": "768",
				"LOGPILOT_SERVICE_TARGETS": "video=http://video:8080/healthz,user=http://user:8080/healthz",
			},
			wantAddr:   ":8080",
			wantDBPath: "./data/logpilot.db",
			wantTargets: map[string]string{
				"video": "http://video:8080/healthz",
				"user":  "http://user:8080/healthz",
			},
		},
		{
			name: "rejects missing model",
			env: map[string]string{
				"LOGPILOT_LLM_BASE_URL":    "http://model.example/v1",
				"LOGPILOT_EMBED_BASE_URL":  "http://embed.example",
				"LOGPILOT_EMBED_MODEL":     "test-embed",
				"LOGPILOT_EMBED_DIMENSION": "384",
			},
			wantErr: "LOGPILOT_LLM_MODEL is required",
		},
		{
			name: "rejects invalid embedding dimension",
			env: map[string]string{
				"LOGPILOT_LLM_BASE_URL":    "http://model.example/v1",
				"LOGPILOT_LLM_MODEL":       "test-model",
				"LOGPILOT_EMBED_BASE_URL":  "http://embed.example",
				"LOGPILOT_EMBED_MODEL":     "test-embed",
				"LOGPILOT_EMBED_DIMENSION": "zero",
			},
			wantErr: "LOGPILOT_EMBED_DIMENSION must be a positive integer",
		},
		{
			name: "rejects malformed service target",
			env: map[string]string{
				"LOGPILOT_LLM_BASE_URL":    "http://model.example/v1",
				"LOGPILOT_LLM_MODEL":       "test-model",
				"LOGPILOT_EMBED_BASE_URL":  "http://embed.example",
				"LOGPILOT_EMBED_MODEL":     "test-embed",
				"LOGPILOT_EMBED_DIMENSION": "384",
				"LOGPILOT_SERVICE_TARGETS": "missing-url",
			},
			wantErr: "invalid LOGPILOT_SERVICE_TARGETS entry",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clearEnvironment(t)
			for key, value := range test.env {
				t.Setenv(key, value)
			}

			cfg, err := Load()
			if test.wantErr != "" {
				if err == nil || err.Error() != test.wantErr {
					t.Fatalf("Load() error = %v, want %q", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() unexpected error: %v", err)
			}
			if cfg.HTTPAddr != test.wantAddr {
				t.Errorf("HTTPAddr = %q, want %q", cfg.HTTPAddr, test.wantAddr)
			}
			if cfg.DatabasePath != test.wantDBPath {
				t.Errorf("DatabasePath = %q, want %q", cfg.DatabasePath, test.wantDBPath)
			}
			if cfg.ShutdownTimeout != 15*time.Second {
				t.Errorf("ShutdownTimeout = %s, want 15s", cfg.ShutdownTimeout)
			}
			if len(cfg.ServiceTargets) != len(test.wantTargets) {
				t.Fatalf("ServiceTargets length = %d, want %d", len(cfg.ServiceTargets), len(test.wantTargets))
			}
			for name, target := range test.wantTargets {
				if cfg.ServiceTargets[name] != target {
					t.Errorf("ServiceTargets[%q] = %q, want %q", name, cfg.ServiceTargets[name], target)
				}
			}
		})
	}
}

func clearEnvironment(t *testing.T) {
	t.Helper()
	keys := []string{
		"LOGPILOT_HTTP_ADDR",
		"LOGPILOT_DATABASE_PATH",
		"LOGPILOT_LLM_BASE_URL",
		"LOGPILOT_LLM_API_KEY",
		"LOGPILOT_LLM_MODEL",
		"LOGPILOT_EMBED_BASE_URL",
		"LOGPILOT_EMBED_MODEL",
		"LOGPILOT_EMBED_DIMENSION",
		"LOGPILOT_QDRANT_URL",
		"LOGPILOT_SERVICE_TARGETS",
	}
	for _, key := range keys {
		t.Setenv(key, "")
	}
}
