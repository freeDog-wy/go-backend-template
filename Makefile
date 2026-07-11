.PHONY: server worker cron migrate-up migrate-down migrate-version test test-unit test-integration test-db-integration test-redis-integration test-kafka-integration test-ci test-verbose test-auth test-mq test-support test-consumption-integration

GO ?= go

all: test server worker cron

server:
	$(GO) build -o build/server.exe ./cmd/server

worker:
	$(GO) build -o build/worker.exe ./cmd/worker

cron:
	$(GO) build -o build/cron.exe ./cmd/cron

migrate-up:
	$(GO) run ./cmd/migrate -direction up

migrate-down:
	$(GO) run ./cmd/migrate -direction down -allow-destructive

migrate-version:
	$(GO) run ./cmd/migrate -version

test: test-unit

test-unit:
	$(GO) test ./...

test-integration: test-db-integration test-redis-integration test-kafka-integration

test-db-integration:
	$(GO) test -tags=integration ./internal/infra/database ./internal/repository/...

test-redis-integration:
	$(GO) test -tags=integration ./internal/infra/cache ./pkg/captcha ./pkg/ratelimit

test-kafka-integration:
	$(GO) test -tags=integration ./internal/infra/mq

test-ci: test-unit test-integration

test-verbose:
	$(GO) test -v ./...

test-auth:
	$(GO) test ./internal/usecase/auth

test-mq:
	$(GO) test ./internal/infra/mq

test-support:
	$(GO) test ./internal/usecase/support

test-consumption-integration:
	$(GO) test -v -tags=integration ./internal/repository/consumption
