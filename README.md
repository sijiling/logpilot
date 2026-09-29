# LogPilot

[![Go Version](https://img.shields.io/badge/Go-1.23%2B-00ADD8)](https://go.dev/) [![License](https://img.shields.io/badge/license-MIT-green)](./LICENSE)

LogPilot 是一个使用 Go、Gin 和 CloudWeGo Eino 构建的智能日志诊断 Agent。它通过 ReAct 编排知识检索、日志分析和服务健康检查工具，输出带执行轨迹的故障建议。

## Demo

```text
event: tool_result
data: {"name":"search_knowledge","status":"succeeded"}

event: answer
data: {"delta":"日志显示服务发生 nil pointer panic，请从堆栈首个业务函数检查空值来源。"}
```

## Getting Started

前置条件：Go 1.23+、Docker Desktop、DeepSeek API Key。

```powershell
$env:LOGPILOT_LLM_API_KEY='your-api-key'
docker compose up -d --build
powershell -ExecutionPolicy Bypass -File scripts\smoke.ps1
```

API Key 仅通过环境变量注入。不要写入仓库、镜像或配置文件。

## Features

- Eino ReAct Agent，最大 6 步、60 秒诊断超时。
- Markdown 分块与特征哈希向量，Qdrant Top-3 知识检索。
- 日志敏感信息脱敏、Go panic 与 HTTP 状态码提取。
- 服务名白名单健康检查，禁止任意 URL 和重定向。
- 普通 JSON 与 SSE 流式 API，会话和工具轨迹持久化。
- Prometheus 指标、JSON 日志、健康/就绪检查和优雅停机。

## Architecture

```text
Client -> Gin API -> Diagnosis Service -> Eino ReAct Agent -> DeepSeek API
                                      |-> analyze_log
                                      |-> search_knowledge -> Hash Embedder -> Qdrant
                                      `-> check_health
                 -> SQLite (documents, sessions, messages, diagnoses, tool runs)
```

项目采用模块化单体与手工构造函数注入，SQLite 保存业务数据，Qdrant 保存可重建的检索索引。

## Configuration

| 环境变量 | 默认值 | 说明 |
| --- | --- | --- |
| `LOGPILOT_HTTP_ADDR` | `:8080` | HTTP 监听地址 |
| `LOGPILOT_DATABASE_PATH` | `./data/logpilot.db` | SQLite 路径 |
| `LOGPILOT_LLM_BASE_URL` | 无 | OpenAI 兼容 API 地址 |
| `LOGPILOT_LLM_API_KEY` | 空 | API Key，不得提交 |
| `LOGPILOT_LLM_MODEL` | 无 | ChatModel ID，DeepSeek Flash 为 `deepseek-flash` |
| `LOGPILOT_EMBED_DIMENSION` | `384` | 特征哈希向量维度 |
| `LOGPILOT_QDRANT_URL` | `http://qdrant:6333` | Qdrant 地址 |
| `LOGPILOT_SERVICE_TARGETS` | 空 | `name=url` 格式的健康检查白名单 |

## API

- `POST /api/v1/documents`：上传 Markdown 知识文档。
- `POST /api/v1/diagnoses`：执行诊断。
- `POST /api/v1/diagnoses/stream`：SSE 诊断。
- `GET /api/v1/sessions/{id}`：查询会话。
- `GET /healthz`、`GET /readyz`、`GET /metrics`：运维端点。

完整契约见 [OpenAPI](docs/design/openapi.yaml)。

## Testing

```powershell
go test ./...
go test -race ./...
go vet ./...
golangci-lint run ./...
```

## Contributing

参见 [CONTRIBUTING.md](CONTRIBUTING.md)。

## License

[MIT](LICENSE)
