// Package tools implements the business tools exposed to the Eino Agent.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

var (
	emailPattern  = regexp.MustCompile(`(?i)\b[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}\b`)
	ipv4Pattern   = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
	bearerPattern = regexp.MustCompile(`(?i)(bearer\s+)[A-Za-z0-9._~+/-]+`)
	statusPattern = regexp.MustCompile(`\b[1-5]\d{2}\b`)
)

// LogAnalyzer extracts deterministic signals from logs after redaction.
type LogAnalyzer struct{}

// NewLogAnalyzer creates the local log-analysis tool.
func NewLogAnalyzer() *LogAnalyzer { return &LogAnalyzer{} }

func (*LogAnalyzer) Info(context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "analyze_log",
		Desc: "Analyze supplied log text for severity, HTTP codes, Go panic signals, and a redacted excerpt.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"logs": {Type: schema.String, Desc: "Raw log text to analyze", Required: true},
		}),
	}, nil
}

func (*LogAnalyzer) InvokableRun(_ context.Context, arguments string, _ ...tool.Option) (string, error) {
	var input struct {
		Logs string `json:"logs"`
	}
	if err := json.Unmarshal([]byte(arguments), &input); err != nil {
		return "", fmt.Errorf("analyze log: decode arguments: %w", err)
	}
	if strings.TrimSpace(input.Logs) == "" {
		return "", fmt.Errorf("analyze log: logs are required")
	}
	redacted := redact(input.Logs)
	level := "unknown"
	upper := strings.ToUpper(redacted)
	for _, candidate := range []string{"FATAL", "ERROR", "WARN", "INFO", "DEBUG"} {
		if strings.Contains(upper, candidate) {
			level = candidate
			break
		}
	}
	result := struct {
		Level       string   `json:"level"`
		Panic       bool     `json:"panic"`
		StatusCodes []string `json:"status_codes"`
		Excerpt     string   `json:"redacted_excerpt"`
	}{Level: level, Panic: strings.Contains(strings.ToLower(redacted), "panic:"), StatusCodes: statusPattern.FindAllString(redacted, -1), Excerpt: truncate(redacted, 4096)}
	encoded, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("analyze log: encode result: %w", err)
	}
	return string(encoded), nil
}

func redact(value string) string {
	value = emailPattern.ReplaceAllString(value, "[REDACTED_EMAIL]")
	value = ipv4Pattern.ReplaceAllString(value, "[REDACTED_IP]")
	return bearerPattern.ReplaceAllString(value, "${1}[REDACTED_TOKEN]")
}

func truncate(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

var _ tool.InvokableTool = (*LogAnalyzer)(nil)
