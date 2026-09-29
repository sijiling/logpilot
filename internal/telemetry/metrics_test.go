package telemetry

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestMetricsInstrumentHTTP(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)
	handler := metrics.Instrument(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusCreated) }))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/diagnoses", nil))

	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("Gather() error: %v", err)
	}
	names := make([]string, 0, len(families))
	for _, family := range families {
		names = append(names, family.GetName())
	}
	joined := strings.Join(names, ",")
	if !strings.Contains(joined, "logpilot_http_requests_total") || !strings.Contains(joined, "logpilot_http_request_duration_seconds") {
		t.Fatalf("metrics = %s", joined)
	}
}
