SHELL := /usr/bin/env bash
.DEFAULT_GOAL := help

COMPOSE := docker compose
SAFE_LOCAL := ./scripts/safe-local.sh
SAFE_COMPOSE := ./scripts/safe-compose.sh

.PHONY: help bootstrap infra migrate seed up down restart ps logs fmt lint test test-race build web-install web-check verify clean

help: ## Show available commands
	@awk 'BEGIN {FS = ":.*## "; printf "Fleet Tracking commands:\n\n"} /^[a-zA-Z_-]+:.*?## / {printf "  %-16s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

bootstrap: ## Create .env, start dependencies, migrate, seed, and start the stack
	@test -f .env || cp .env.example .env
	$(SAFE_COMPOSE) up -d postgres redis rabbitmq prometheus grafana otel-collector
	$(COMPOSE) --profile tools run --rm migrate
	$(COMPOSE) --profile tools run --rm seed
	$(SAFE_COMPOSE) up -d --build api worker outbox-publisher web

infra: ## Start PostgreSQL, Redis, RabbitMQ, and observability
	$(COMPOSE) up -d postgres redis rabbitmq prometheus grafana otel-collector

migrate: ## Apply all database migrations
	$(COMPOSE) --profile tools run --rm migrate

seed: ## Load rerunnable development seed data
	$(COMPOSE) --profile tools run --rm seed

up: ## Build and start all Milestone 1 services
	$(SAFE_COMPOSE) up -d --build api worker outbox-publisher web prometheus grafana otel-collector

down: ## Stop the local stack without deleting data
	$(COMPOSE) down

restart: ## Restart application processes without rebuilding
	$(COMPOSE) restart api worker outbox-publisher web

ps: ## Show service status
	$(COMPOSE) ps

logs: ## Follow application logs
	$(COMPOSE) logs -f --tail=200 api worker outbox-publisher web

fmt: ## Format Go sources
	gofmt -w $$(find cmd internal tests -name '*.go' -type f 2>/dev/null)

lint: ## Run Go linters through the host resource guard
	$(SAFE_LOCAL) golangci-lint run ./...

test: ## Run Go tests through the host resource guard
	$(SAFE_LOCAL) go test ./cmd/... ./internal/...

test-race: ## Run Go tests with the race detector
	$(SAFE_LOCAL) go test -race ./cmd/... ./internal/...

build: ## Build all Go binaries through the host resource guard
	$(SAFE_LOCAL) go build ./cmd/...

web-install: ## Install pinned frontend dependencies
	cd web && npm ci

web-check: ## Lint, typecheck, and build the frontend safely
	$(SAFE_LOCAL) bash -lc 'cd web && npm run lint && npm run typecheck && npm run build'

verify: ## Run the local release gate
	$(SAFE_LOCAL) bash -lc 'go test -race ./cmd/... ./internal/... && go vet ./cmd/... ./internal/... && go build ./cmd/...'
	$(SAFE_LOCAL) bash -lc 'cd web && npm ci && npm run lint && npm run typecheck && npm run build'

clean: ## Remove build outputs only
	rm -rf bin dist coverage.out web/.next web/coverage
