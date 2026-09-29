package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"

	logagent "github.com/sijiling/logpilot/internal/agent"
	"github.com/sijiling/logpilot/internal/config"
	"github.com/sijiling/logpilot/internal/diagnosis"
	"github.com/sijiling/logpilot/internal/document"
	"github.com/sijiling/logpilot/internal/embedding"
	"github.com/sijiling/logpilot/internal/httpapi"
	"github.com/sijiling/logpilot/internal/llm"
	"github.com/sijiling/logpilot/internal/server"
	"github.com/sijiling/logpilot/internal/session"
	"github.com/sijiling/logpilot/internal/telemetry"
	diagtools "github.com/sijiling/logpilot/internal/tools"
	"github.com/sijiling/logpilot/internal/vector"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("load configuration", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	repository, err := session.Open(cfg.DatabasePath)
	if err != nil {
		logger.Error("open database", "error", err)
		os.Exit(1)
	}
	defer repository.Close()
	embedder, err := embedding.NewOllama(&http.Client{Timeout: 20 * time.Second}, cfg.EmbedBaseURL, cfg.EmbedModel, cfg.EmbedDimension)
	if err != nil {
		logger.Error("create embedder", "error", err)
		os.Exit(1)
	}
	vectorStore, err := vector.NewQdrant(&http.Client{Timeout: 5 * time.Second}, cfg.QdrantURL, "logpilot_documents")
	if err != nil {
		logger.Error("create vector store", "error", err)
		os.Exit(1)
	}
	chatModel, err := llm.NewChatModel(ctx, cfg)
	if err != nil {
		logger.Error("create chat model", "error", err)
		os.Exit(1)
	}
	toolSet := []tool.BaseTool{diagtools.NewLogAnalyzer(), diagtools.NewKnowledgeSearch(embedder, vectorStore), diagtools.NewHealthCheck(&http.Client{Timeout: 3 * time.Second}, cfg.ServiceTargets)}
	agentFactory := func(factoryCtx context.Context) (diagnosis.Runner, *logagent.TraceRecorder, error) {
		recorder := logagent.NewTraceRecorder()
		runner, createErr := logagent.New(factoryCtx, chatModel, toolSet, recorder)
		return runner, recorder, createErr
	}
	documentService := document.NewService(document.NewSQLiteStore(repository.DB()), embedder, vectorStore, uuid.NewString, time.Now)
	diagnosisService := diagnosis.NewService(repository, agentFactory, uuid.NewString, time.Now)
	api := httpapi.NewRouter(httpapi.Dependencies{Documents: documentService, Diagnoses: diagnosisService, Sessions: repository})
	metrics := telemetry.NewMetrics(prometheus.NewRegistry())
	application := telemetry.Logging(logger, metrics.Instrument(api))
	httpServer := server.New(cfg, logger, server.WithHandler(application), server.WithMetrics(metrics.Handler()), server.WithReadiness(repository.DB().PingContext), server.WithReadiness(vectorStore.Ready))
	if err := httpServer.Run(ctx); err != nil {
		logger.Error("run server", "error", err)
		os.Exit(1)
	}
}
