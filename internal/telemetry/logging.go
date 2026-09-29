package telemetry

import (
	"log/slog"
	"net/http"
	"time"
)

// Logging emits one structured record for each HTTP request.
func Logging(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		wrapped := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(wrapped, r)
		logger.InfoContext(r.Context(), "http request", "method", r.Method, "path", normalizedRoute(r.URL.Path), "status", wrapped.status, "duration_ms", time.Since(started).Milliseconds())
	})
}
