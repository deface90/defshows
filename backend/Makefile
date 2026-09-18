# defShows backend — developer tasks
#
# DB connection for migrations (override via env or .env in deploy/):
DB_DSN ?= postgres://defshows:defshows@localhost:5432/defshows?sslmode=disable
MIGRATIONS_DIR := migrations/migrate
COMPOSE := docker compose -f deploy/docker-compose.yml

# Tooling is invoked via `go run` so no global installs are required.
GOOSE := go run github.com/pressly/goose/v3/cmd/goose@latest -dir $(MIGRATIONS_DIR) postgres "$(DB_DSN)"

.PHONY: help generate migrate-up migrate-down migrate-new test lint build up down tidy

help: ## show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS=":.*?## "}{printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

generate: ## run code generation (openapi, mocks, etc.)
	go generate ./...

migrate-up: ## apply all pending migrations
	$(GOOSE) up

migrate-down: ## roll back the last migration
	$(GOOSE) down

migrate-new: ## create a new migration: make migrate-new name=create_users
	go run github.com/pressly/goose/v3/cmd/goose@latest -dir $(MIGRATIONS_DIR) create $(name) sql

test: ## run unit tests
	go test ./...

lint: ## run linters
	golangci-lint run

build: ## build all binaries
	go build ./...

tidy: ## tidy go.mod
	go mod tidy

up: ## start local stack (postgres, minio, services)
	$(COMPOSE) up -d --build

down: ## stop local stack
	$(COMPOSE) down
