# Contributing

## Prerequisites

- Go 1.23+
- Docker Desktop
- golangci-lint v2

## Development

```powershell
git clone https://github.com/sijiling/logpilot.git
cd logpilot
go mod download
go test ./...
```

新增行为遵循测试先行：先运行失败测试，再写最小实现。提交前执行：

```powershell
gofmt -w .
go vet ./...
golangci-lint run ./...
go test ./...
go test -race ./...
```

提交信息使用 `feat:`、`fix:`、`test:`、`docs:` 或 `build:` 前缀。Pull Request 应说明行为变化、验证命令及安全影响。
