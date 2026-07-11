.PHONY: server worker cron test test-verbose test-auth test-mq test-support test-consumption-integration

GO ?= go

all: test server worker cron

server:
	$(GO) build -o build/server.exe ./cmd/server

worker:
	$(GO) build -o build/worker.exe ./cmd/worker

cron:
	$(GO) build -o build/cron.exe ./cmd/cron

test:
	$(GO) test ./...

test-verbose:
	$(GO) test -v ./...

test-auth:
	$(GO) test ./internal/usecase/auth

test-mq:
	$(GO) test ./internal/infra/mq

test-support:
	$(GO) test ./internal/usecase/support

test-consumption-integration:
	$(GO) test -v ./internal/repository/consumption
