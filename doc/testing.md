# 测试指南

项目将测试分为无需外部服务的单元测试和依赖真实基础设施的集成测试。

## 单元测试

单元测试使用 stub、fake 或内存实现验证领域规则、Usecase 编排、Handler 响应映射和基础设施适配逻辑。它们不依赖 PostgreSQL、Redis 或 Kafka。

```bash
make test-unit
```

`make test` 是 `make test-unit` 的别名，适合本地快速反馈和每个 PR 的默认质量门槛。

## PostgreSQL 集成测试

集成测试使用 `integration` build tag，不会在普通 `go test ./...` 中编译或执行。执行前必须显式设置 `TEST_DATABASE_DSN`；未设置会失败，避免测试意外连接开发数据库。

先启动本地依赖：

```bash
docker compose -f deploy/docker-compose.yml up -d postgres
```

PowerShell：

```powershell
$env:TEST_DATABASE_DSN = "host=localhost user=postgres password=postgres dbname=go_backend port=5432 sslmode=disable TimeZone=Asia/Shanghai"
make test-integration
```

集成测试必须：

- 使用 `internal/testkit.OpenPostgres` 建立连接；不得内置默认 DSN。
- 使用测试专用数据库或 schema，绝不使用生产数据库。
- 使用唯一测试数据并在 `t.Cleanup` 中清理。
- 覆盖数据库约束、事务、并发和真实 SQL 查询语义。

## CI

CI 应启动独立 PostgreSQL 服务，并注入 `TEST_DATABASE_DSN`，随后执行：

```bash
make test-ci
go vet ./...
go build ./cmd/server ./cmd/worker ./cmd/cron
```

Redis 和 Kafka 集成测试采用相同约定：使用独立 build tag、显式环境变量和 CI 服务容器，不能隐式依赖开发者本机服务。
