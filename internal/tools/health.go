package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const maxHealthBody = 4096

// HealthCheck queries administrator-configured service health endpoints.
type HealthCheck struct {
	client  *http.Client
	targets map[string]string
}

// NewHealthCheck creates an allowlisted health-check tool.
func NewHealthCheck(client *http.Client, targets map[string]string) *HealthCheck {
	copyClient := *client
	copyClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &HealthCheck{client: &copyClient, targets: targets}
}

func (h *HealthCheck) Info(context.Context) (*schema.ToolInfo, error) {
	services := make([]string, 0, len(h.targets))
	for service := range h.targets {
		services = append(services, service)
	}
	return &schema.ToolInfo{Name: "check_health", Desc: "Check one configured service health endpoint. Only allowlisted service names are accepted.", ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
		"service": {Type: schema.String, Desc: "Configured service name", Required: true, Enum: services},
	})}, nil
}

func (h *HealthCheck) InvokableRun(ctx context.Context, arguments string, _ ...tool.Option) (string, error) {
	var input struct {
		Service string `json:"service"`
	}
	if err := json.Unmarshal([]byte(arguments), &input); err != nil {
		return "", fmt.Errorf("check health: decode arguments: %w", err)
	}
	target, ok := h.targets[input.Service]
	if !ok {
		return "", fmt.Errorf("check health: service is not allowlisted")
	}
	requestCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, target, nil)
	if err != nil {
		return "", fmt.Errorf("check health: create request: %w", err)
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("check health: execute request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxHealthBody))
	if err != nil {
		return "", fmt.Errorf("check health: read response: %w", err)
	}
	result := struct {
		Service    string `json:"service"`
		StatusCode int    `json:"status_code"`
		Healthy    bool   `json:"healthy"`
		Body       string `json:"body"`
	}{
		Service: input.Service, StatusCode: resp.StatusCode, Healthy: resp.StatusCode >= 200 && resp.StatusCode < 300, Body: strings.TrimSpace(string(body)),
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("check health: encode result: %w", err)
	}
	return string(encoded), nil
}

var _ tool.InvokableTool = (*HealthCheck)(nil)
