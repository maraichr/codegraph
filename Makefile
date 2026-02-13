.PHONY: build test lint fmt clean migrate-up migrate-down generate dev docker-up docker-down

# Variables
BINARY_DIR := bin
GO := go
GOFLAGS := -trimpath
MIGRATE := migrate
DB_URL ?= postgres://codegraph:codegraph@localhost:5432/codegraph?sslmode=disable

# Build all binaries
build:
	$(GO) build $(GOFLAGS) -o $(BINARY_DIR)/api ./cmd/api
	$(GO) build $(GOFLAGS) -o $(BINARY_DIR)/worker ./cmd/worker
	$(GO) build $(GOFLAGS) -o $(BINARY_DIR)/mcp ./cmd/mcp
	$(GO) build $(GOFLAGS) -o $(BINARY_DIR)/scheduler ./cmd/scheduler

# Build individual binaries
build-api:
	$(GO) build $(GOFLAGS) -o $(BINARY_DIR)/api ./cmd/api

build-worker:
	$(GO) build $(GOFLAGS) -o $(BINARY_DIR)/worker ./cmd/worker

build-mcp:
	$(GO) build $(GOFLAGS) -o $(BINARY_DIR)/mcp ./cmd/mcp

build-scheduler:
	$(GO) build $(GOFLAGS) -o $(BINARY_DIR)/scheduler ./cmd/scheduler

# Test
test:
	$(GO) test ./... -race -cover

test-coverage:
	$(GO) test ./... -race -coverprofile=coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html

# Lint and format
lint:
	$(GO) vet ./...
	cd frontend && pnpm biome check src/

fmt:
	$(GO) fmt ./...
	cd frontend && pnpm biome format --write src/

# Clean
clean:
	rm -rf $(BINARY_DIR)
	rm -f coverage.out coverage.html

# Database migrations
migrate-up:
	$(MIGRATE) -path migrations/postgres -database "$(DB_URL)" up

migrate-down:
	$(MIGRATE) -path migrations/postgres -database "$(DB_URL)" down 1

migrate-create:
	$(MIGRATE) create -ext sql -dir migrations/postgres -seq $(name)

# Code generation
generate: generate-sqlc generate-graphql

generate-sqlc:
	sqlc generate

generate-graphql:
	$(GO) run github.com/99designs/gqlgen generate

# Development
dev:
	$(GO) run ./cmd/api

dev-frontend:
	cd frontend && pnpm dev

# Docker
docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f
