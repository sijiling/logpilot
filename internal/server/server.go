// Package server owns the HTTP server lifecycle.
package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/sijiling/logpilot/internal/config"
)

// Server is the LogPilot HTTP server.
type Server struct {
	logger          *slog.Logger
	httpServer      *http.Server
	handler         http.Handler
	shutdownTimeout time.Duration
}

type settings struct {
	handler   http.Handler
	metrics   http.Handler
	readiness []func(context.Context) error
}

// Option customizes server routing and dependency checks.
type Option func(*settings)

// WithHandler mounts the application handler after operational endpoints.
func WithHandler(handler http.Handler) Option { return func(s *settings) { s.handler = handler } }

// WithMetrics mounts a Prometheus handler at /metrics.
func WithMetrics(handler http.Handler) Option { return func(s *settings) { s.metrics = handler } }

// WithReadiness adds a dependency readiness check.
func WithReadiness(check func(context.Context) error) Option {
	return func(s *settings) { s.readiness = append(s.readiness, check) }
}

// New creates an HTTP server with bounded timeouts and core middleware.
func New(cfg config.Config, logger *slog.Logger, options ...Option) *Server {
	settings := &settings{}
	for _, option := range options {
		option(settings)
	}
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery(), requestID())
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.GET("/readyz", func(c *gin.Context) {
		for _, check := range settings.readiness {
			if err := check(c.Request.Context()); err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready"})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
	if settings.metrics != nil {
		router.GET("/metrics", gin.WrapH(settings.metrics))
	}
	if settings.handler != nil {
		router.NoRoute(gin.WrapH(settings.handler))
	}

	shutdownTimeout := cfg.ShutdownTimeout
	if shutdownTimeout <= 0 {
		shutdownTimeout = 15 * time.Second
	}

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      65 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return &Server{
		logger:          logger,
		httpServer:      httpServer,
		handler:         router,
		shutdownTimeout: shutdownTimeout,
	}
}

// Handler returns the configured HTTP handler for tests and embedding.
func (s *Server) Handler() http.Handler {
	return s.handler
}

// HTTPServer exposes the configured standard library server.
func (s *Server) HTTPServer() *http.Server {
	return s.httpServer
}

// Run serves requests until the context is cancelled or the server fails.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.httpServer.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.shutdownTimeout)
		defer cancel()
		s.logger.Info("shutting down HTTP server")
		return s.httpServer.Shutdown(shutdownCtx)
	}
}

func requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.NewString()
		}
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}
