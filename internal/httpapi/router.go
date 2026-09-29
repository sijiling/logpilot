// Package httpapi exposes LogPilot's HTTP API.
package httpapi

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/sijiling/logpilot/internal/diagnosis"
	"github.com/sijiling/logpilot/internal/document"
	"github.com/sijiling/logpilot/internal/session"
)

type DocumentIndexer interface {
	Index(context.Context, string, []byte) (document.IndexResult, error)
}
type Diagnoser interface {
	Diagnose(context.Context, diagnosis.Request) (diagnosis.Response, error)
	Stream(context.Context, diagnosis.Request, func(diagnosis.Event) error) error
}
type SessionReader interface {
	Find(context.Context, string) (session.Session, []session.Message, error)
}

// Dependencies contains HTTP-layer application ports.
type Dependencies struct {
	Documents DocumentIndexer
	Diagnoses Diagnoser
	Sessions  SessionReader
}

// NewRouter creates the versioned HTTP router.
func NewRouter(dependencies Dependencies) http.Handler {
	gin.SetMode(gin.ReleaseMode)
	h := &handler{dependencies: dependencies, diagnoses: make(chan struct{}, 10)}
	router := gin.New()
	router.Use(gin.Recovery())
	v1 := router.Group("/api/v1")
	v1.POST("/documents", h.uploadDocument)
	v1.POST("/diagnoses", h.diagnose)
	v1.POST("/diagnoses/stream", h.streamDiagnosis)
	v1.GET("/sessions/:sessionID", h.getSession)
	return router
}

type handler struct {
	dependencies Dependencies
	diagnoses    chan struct{}
}
