# LogPilot MVP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a small Go service that diagnoses logs through an Eino ReAct Agent using knowledge search, log analysis, and health-check tools.

**Architecture:** Use a modular monolith with manual constructor injection at `cmd/server/main.go`. Define small interfaces at consumer boundaries; store business metadata in SQLite and rebuildable vectors in Qdrant.

**Tech Stack:** Go 1.23+, Gin, SQLite, Qdrant, OpenAI-compatible HTTP APIs, Ollama, Prometheus, Docker Compose.

## Global Constraints

- Module path is `github.com/sijiling/logpilot`; change it once, before the first remote push, if the actual GitHub username differs.
- Each numbered task must end in one Git commit after its task-specific verification passes; the completed repository must be pushed to GitHub after full verification.
- The MVP has no shell tool, user authentication, frontend, multi-agent orchestration, or automatic remediation.
- Agent runs are limited to 6 rounds, 60 seconds, and 10 concurrent diagnoses.
- Use manual constructor injection; do not introduce a DI container.
- Use CloudWeGo Eino and Eino-Ext for ChatModel, Tool, ReAct, streaming, and callbacks; do not implement a private Agent framework.
- Every external operation accepts `context.Context` and has an explicit timeout.
- Follow the contracts in `docs/logpilot/01-软件需求规格说明书.md` and `docs/logpilot/openapi.yaml`.

---

### Task 1: Bootstrap the service and quality gates

**Files:**
- Create: `logpilot/go.mod`
- Create: `logpilot/cmd/server/main.go`
- Create: `logpilot/internal/config/config.go`
- Create: `logpilot/internal/config/config_test.go`
- Create: `logpilot/internal/server/server.go`
- Create: `logpilot/internal/server/server_test.go`
- Create: `logpilot/.golangci.yml`
- Create: `logpilot/Makefile`

**Interfaces:**
- Produces: `config.Load() (config.Config, error)` and `server.New(config.Config, *slog.Logger) *server.Server`.

- [ ] Write table-driven tests for defaults, missing model settings, invalid embedding dimension, and malformed service targets.
- [ ] Run `go test ./internal/config -v`; expect failure because `config.Load` does not exist.
- [ ] Implement environment parsing with defaults from `docs/logpilot/06-项目计划与风险.md` and return errors for missing required values.
- [ ] Implement Gin server with `/healthz`, timeouts, request ID middleware, signal handling, and 15-second graceful shutdown.
- [ ] Run `go test ./internal/config ./internal/server -v`; expect PASS.
- [ ] Add `fmt`, `vet`, `lint`, `test`, and `race` Make targets and run `go vet ./...`.
- [ ] Commit with `git commit -m "feat: bootstrap logpilot service"`.

### Task 2: Implement SQLite persistence

**Files:**
- Create: `logpilot/migrations/001_init.sql`
- Create: `logpilot/internal/session/model.go`
- Create: `logpilot/internal/session/repository.go`
- Create: `logpilot/internal/session/sqlite.go`
- Create: `logpilot/internal/session/sqlite_test.go`
- Create: `logpilot/internal/document/model.go`

**Interfaces:**
- Produces: `session.Repository` with `Create`, `AppendMessage`, `SaveDiagnosis`, and `Find` methods.
- Produces: `document.Record` and document status constants.

- [ ] Write repository tests using a temporary SQLite database for create/find, ordered messages, transaction rollback, and missing sessions.
- [ ] Run `go test ./internal/session -v`; expect failure because repository types do not exist.
- [ ] Add the five tables and indexes defined in `docs/logpilot/04-数据设计说明书.md`.
- [ ] Implement repository methods with parameterized statements, transactions, UTC timestamps, and wrapped errors.
- [ ] Run `go test ./internal/session -v`; expect PASS.
- [ ] Commit with `git commit -m "feat: add sqlite session persistence"`.

### Task 3: Implement document chunking and indexing

**Files:**
- Create: `logpilot/internal/document/chunker.go`
- Create: `logpilot/internal/document/chunker_test.go`
- Create: `logpilot/internal/document/service.go`
- Create: `logpilot/internal/document/service_test.go`
- Create: `logpilot/internal/embedding/embedder.go`
- Create: `logpilot/internal/vector/store.go`

**Interfaces:**
- Consumes: document records from Task 2.
- Produces: `document.Chunk(text string, target, overlap int) []Chunk`.
- Produces: `embedding.Embedder.Embed(context.Context, []string) ([][]float32, error)`.
- Produces: `vector.Store.Upsert`, `Search`, and `DeleteDocument`.

- [ ] Write chunker tests for headings, paragraph boundaries, oversized paragraphs, stable indexes, and 120-character overlap.
- [ ] Write service tests proving duplicate SHA-256 content skips embedding and failed vector writes set status to `failed` and trigger cleanup.
- [ ] Run `go test ./internal/document -v`; expect failure because chunker and service do not exist.
- [ ] Implement deterministic chunking and the processing→ready/failed state transition.
- [ ] Run `go test ./internal/document -v`; expect PASS.
- [ ] Commit with `git commit -m "feat: add document indexing workflow"`.

### Task 4: Add Ollama and Qdrant adapters

**Files:**
- Create: `logpilot/internal/embedding/ollama.go`
- Create: `logpilot/internal/embedding/ollama_test.go`
- Create: `logpilot/internal/vector/qdrant.go`
- Create: `logpilot/internal/vector/qdrant_test.go`

**Interfaces:**
- Implements: `embedding.Embedder` and `vector.Store` from Task 3.

- [ ] Use `httptest.Server` to test request payloads, response decoding, non-2xx responses, timeouts, response-size limits, and context cancellation.
- [ ] Run `go test ./internal/embedding ./internal/vector -v`; expect failure before adapters exist.
- [ ] Implement HTTP adapters with injected `*http.Client`, no global clients, and deterministic Qdrant point IDs.
- [ ] Run `go test ./internal/embedding ./internal/vector -v`; expect PASS.
- [ ] Commit with `git commit -m "feat: integrate ollama and qdrant"`.

### Task 5: Implement business tools with Eino

**Files:**
- Create: `logpilot/internal/tools/loganalyzer.go`
- Create: `logpilot/internal/tools/loganalyzer_test.go`
- Create: `logpilot/internal/tools/knowledge.go`
- Create: `logpilot/internal/tools/knowledge_test.go`
- Create: `logpilot/internal/tools/health.go`
- Create: `logpilot/internal/tools/health_test.go`

**Interfaces:**
- Produces: three implementations of Eino `tool.InvokableTool`.
- Consumes: `embedding.Embedder` and `vector.Store`.

- [ ] Write tests for tool metadata, malformed JSON, email/IP/token redaction, Go panic extraction, score filtering, target allowlisting, disabled redirects, 3-second timeout, and 4 KiB health-response limits.
- [ ] Run `go test ./internal/tools -v`; expect failure before implementations exist.
- [ ] Implement the three Eino tools without filesystem or shell access and verify their JSON Schema matches accepted arguments.
- [ ] Run `go test ./internal/tools -v`; expect PASS.
- [ ] Commit with `git commit -m "feat: add diagnosis tools"`.

### Task 6: Integrate the Eino ReAct Agent

**Files:**
- Create: `logpilot/internal/llm/factory.go`
- Create: `logpilot/internal/llm/factory_test.go`
- Create: `logpilot/internal/agent/prompt.go`
- Create: `logpilot/internal/agent/factory.go`
- Create: `logpilot/internal/agent/factory_test.go`
- Create: `logpilot/internal/agent/callback.go`
- Create: `logpilot/internal/agent/callback_test.go`

**Interfaces:**
- Produces: `llm.NewChatModel(config.Config) (model.ToolCallingChatModel, error)`.
- Produces: `agent.New(ctx context.Context, model model.ToolCallingChatModel, tools []tool.BaseTool, recorder *TraceRecorder) (*react.Agent, error)`.
- Consumes: Eino tools from Task 5.

- [ ] Create a scripted Fake ChatModel implementing Eino's model interface and returning a known sequence of tool calls and final answers.
- [ ] Test successful multi-tool execution, tool failure recovery, MaxStep=6 termination, overall timeout, Eino stream chunks, Callback trace collection, and context cancellation.
- [ ] Run `go test ./internal/agent ./internal/llm -v`; expect failure before the Eino factories and callback recorder exist.
- [ ] Construct the Eino-Ext OpenAI ChatModel and Eino `react.Agent` with `MaxStep: 6`; implement only business prompts and callbacks, and never expose hidden chain-of-thought.
- [ ] Run `go test ./internal/agent ./internal/llm -v`; expect PASS.
- [ ] Commit with `git commit -m "feat: integrate eino react agent"`.

### Task 7: Expose document, diagnosis, session, and SSE APIs

**Files:**
- Create: `logpilot/internal/diagnosis/model.go`
- Create: `logpilot/internal/diagnosis/service.go`
- Create: `logpilot/internal/diagnosis/service_test.go`
- Create: `logpilot/internal/httpapi/router.go`
- Create: `logpilot/internal/httpapi/document.go`
- Create: `logpilot/internal/httpapi/diagnosis.go`
- Create: `logpilot/internal/httpapi/session.go`
- Create: `logpilot/internal/httpapi/sse.go`
- Create: `logpilot/internal/httpapi/handler_test.go`
- Create: `logpilot/api/openapi.yaml`

**Interfaces:**
- Consumes: document service, Eino ReAct Agent, and session repository.
- Produces: HTTP contract identical to `docs/logpilot/openapi.yaml`.

- [ ] Write `httptest` cases for validation, duplicate upload, new/existing sessions, 404, 429, structured errors, SSE ordering, disconnect cancellation, and exactly one `done` event.
- [ ] Run `go test ./internal/diagnosis ./internal/httpapi -v`; expect failure before handlers exist.
- [ ] Implement services and handlers with a 10-slot diagnosis semaphore and request body limits.
- [ ] Copy the reviewed contract to `logpilot/api/openapi.yaml` and keep handler DTO names aligned with schema fields.
- [ ] Run `go test ./internal/diagnosis ./internal/httpapi -v`; expect PASS.
- [ ] Commit with `git commit -m "feat: expose diagnosis APIs"`.

### Task 8: Add telemetry and dependency readiness

**Files:**
- Create: `logpilot/internal/telemetry/logging.go`
- Create: `logpilot/internal/telemetry/metrics.go`
- Create: `logpilot/internal/telemetry/metrics_test.go`
- Modify: `logpilot/internal/server/server.go`
- Modify: `logpilot/cmd/server/main.go`

**Interfaces:**
- Produces: request, Agent, tool, and document Prometheus metrics listed in the architecture document.

- [ ] Write tests that gather metrics after one request and one tool call and assert stable metric names and labels.
- [ ] Implement JSON logging with redaction and `/metrics`.
- [ ] Add `/readyz` checks for SQLite and Qdrant without making paid model calls.
- [ ] Run `go test ./internal/telemetry ./internal/server -v`; expect PASS.
- [ ] Commit with `git commit -m "feat: add telemetry and readiness"`.

### Task 9: Containerize and run end-to-end acceptance

**Files:**
- Create: `logpilot/Dockerfile`
- Create: `logpilot/docker-compose.yml`
- Create: `logpilot/.dockerignore`
- Create: `logpilot/testdata/runbooks/go-panic.md`
- Create: `logpilot/testdata/eval/cases.jsonl`
- Create: `logpilot/scripts/smoke.ps1`

**Interfaces:**
- Produces: a three-service local environment and repeatable smoke command.

- [ ] Add a multi-stage, non-root Docker image and health checks.
- [ ] Configure persistent volumes for SQLite, Qdrant, and Ollama; do not bake secrets into images.
- [ ] Write a PowerShell smoke script that waits for readiness, uploads the runbook, performs a diagnosis, queries the session, and fails on non-2xx responses.
- [ ] Run `docker compose up --build -d`; expect all containers healthy.
- [ ] Run `powershell -ExecutionPolicy Bypass -File scripts/smoke.ps1`; expect `SMOKE TEST PASSED`.
- [ ] Run `docker compose down`; expect clean shutdown without orphan containers.
- [ ] Commit with `git commit -m "build: add local container stack"`.

### Task 10: Finish documentation, evaluation, and release checks

**Files:**
- Create: `logpilot/README.md`
- Create: `logpilot/CONTRIBUTING.md`
- Create: `logpilot/CHANGELOG.md`
- Create: `logpilot/LICENSE`
- Create: `logpilot/llms.txt`
- Create: `logpilot/cmd/eval/main.go`
- Create: `logpilot/internal/eval/eval.go`
- Create: `logpilot/internal/eval/eval_test.go`

**Interfaces:**
- Produces: deterministic retrieval metrics and documented setup/usage.

- [ ] Implement JSONL evaluation loading and calculation of Hit@3, tool-selection accuracy, and required-keyword recall; test exact metric values with fixtures.
- [ ] Write README sections in this order: title/badges, summary, demo, getting started, features, architecture, configuration, API, testing, contributing, license.
- [ ] Document every environment variable and include real terminal output captured from the smoke test.
- [ ] Run `gofmt -w .`, `go vet ./...`, `golangci-lint run ./...`, `go test ./...`, and `go test -race ./...`; expect all commands to pass.
- [ ] Run coverage for `./internal/...`; expect at least 70% statement coverage for core application packages.
- [ ] Commit with `git commit -m "docs: complete logpilot release materials"`.

## Self-Review Result

- Spec coverage: FR-01 through FR-08 and NFR-01 through NFR-08 map to Tasks 1—10.
- Placeholder scan: no implementation task depends on unspecified behavior; optional v0.2 features are explicitly outside this plan.
- Type consistency: Tasks 3—7 consume the exact ports produced by preceding tasks; HTTP names follow the OpenAPI contract.
