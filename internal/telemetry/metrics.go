// Package telemetry provides bounded-cardinality service instrumentation.
package telemetry

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics owns LogPilot Prometheus collectors.
type Metrics struct {
	registry *prometheus.Registry
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

// NewMetrics registers service collectors in the supplied registry.
func NewMetrics(registry *prometheus.Registry) *Metrics {
	metrics := &Metrics{
		registry: registry,
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "logpilot_http_requests_total", Help: "Total HTTP requests."}, []string{"method", "route", "status"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "logpilot_http_request_duration_seconds", Help: "HTTP request latency.", Buckets: prometheus.DefBuckets}, []string{"method", "route"}),
	}
	registry.MustRegister(metrics.requests, metrics.duration)
	return metrics
}

// Handler exposes the registry in Prometheus text format.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

// Instrument records request count, status, and latency.
func (m *Metrics) Instrument(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		wrapped := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(wrapped, r)
		route := normalizedRoute(r.URL.Path)
		m.requests.WithLabelValues(r.Method, route, strconv.Itoa(wrapped.status)).Inc()
		m.duration.WithLabelValues(r.Method, route).Observe(time.Since(started).Seconds())
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func normalizedRoute(path string) string {
	if strings.HasPrefix(path, "/api/v1/sessions/") {
		return "/api/v1/sessions/:id"
	}
	return path
}
