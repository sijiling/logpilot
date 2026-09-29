package tools

import (
	"context"
	"strings"
	"testing"
)

func TestLogAnalyzerRedactsAndExtractsPanic(t *testing.T) {
	tool := NewLogAnalyzer()
	info, err := tool.Info(context.Background())
	if err != nil {
		t.Fatalf("Info() error: %v", err)
	}
	if info.Name != "analyze_log" {
		t.Fatalf("name = %q", info.Name)
	}

	input := `{"logs":"ERROR panic: runtime error: invalid memory address user=a@example.com ip=10.1.2.3 Authorization: Bearer secret-token"}`
	result, err := tool.InvokableRun(context.Background(), input)
	if err != nil {
		t.Fatalf("InvokableRun() error: %v", err)
	}
	for _, secret := range []string{"a@example.com", "10.1.2.3", "secret-token"} {
		if strings.Contains(result, secret) {
			t.Errorf("result contains secret %q: %s", secret, result)
		}
	}
	if !strings.Contains(result, `"panic":true`) {
		t.Errorf("result does not identify panic: %s", result)
	}
}

func TestLogAnalyzerRejectsMalformedJSON(t *testing.T) {
	if _, err := NewLogAnalyzer().InvokableRun(context.Background(), `{`); err == nil {
		t.Fatal("error = nil")
	}
}
