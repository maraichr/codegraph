# CodeGraph

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8.svg)](https://go.dev/)
[![Status](https://img.shields.io/badge/Status-Alpha-orange.svg)]()

A semantic codebase indexing engine that extracts, indexes, and exposes rich dependency graphs, data lineage, and structural metadata across heterogeneous technology stacks. Built for database-heavy enterprise codebases spanning SQL Server, PostgreSQL, ASP Classic, Delphi, Java, and .NET.

## Key Features

- **7 Language Parsers** — T-SQL, PostgreSQL, C#, ASP Classic, Delphi, Java (tree-sitter), with dialect-aware SQL parsing
- **Symbol Graph** — Functions, procedures, classes, tables, views, columns, and their interrelationships stored in PostgreSQL + Neo4j
- **Column-Level Lineage** — Trace data from source tables through transformations, stored procedures, and views
- **MCP Tool Layer** — Expose the semantic graph to LLMs via Model Context Protocol tools for autonomous codebase research
- **Vector Embeddings** — Semantic search over symbols using pgvector with configurable embedding providers
- **Multi-Source Ingestion** — GitLab (PAT + webhooks), S3 buckets, ZIP uploads with incremental indexing
- **Impact Analysis** — Given a proposed change, enumerate all affected code paths and downstream consumers

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    INGESTION LAYER                       │
│   GitLab (PAT)  │  S3 Buckets  │  ZIP Upload  │ Webhooks│
│                         │                                │
│              ┌──────────▼──────────┐                     │
│              │  Valkey Streams     │                     │
│              │  (Ingestion Queue)  │                     │
│              └──────────┬──────────┘                     │
├─────────────────────────┼───────────────────────────────┤
│                   PARSING LAYER                          │
│   ┌───────┐ ┌──────┐ ┌─────┐ ┌──────┐ ┌────┐ ┌──────┐  │
│   │ T-SQL │ │PgSQL │ │ ASP │ │Delphi│ │Java│ │  C#  │  │
│   └───────┘ └──────┘ └─────┘ └──────┘ └────┘ └──────┘  │
│                         │                                │
│              Symbol Resolution & Cross-Ref Engine        │
├─────────────────────────┼───────────────────────────────┤
│                   STORAGE LAYER                          │
│   ┌──────────────┐  ┌────────┐  ┌───────────────────┐   │
│   │  PostgreSQL   │  │ Neo4j  │  │ MinIO (artifacts) │   │
│   │  + pgvector   │  │ (graph)│  │                   │   │
│   └──────────────┘  └────────┘  └───────────────────┘   │
├─────────────────────────┼───────────────────────────────┤
│                    ACCESS LAYER                           │
│   REST API  │  GraphQL  │  MCP Tools  │  React Frontend  │
└─────────────────────────────────────────────────────────┘
```

**Pipeline:** Clone → Parse → Resolve → GraphBuild → Embed

## Quick Start

```bash
# Clone the repository
git clone https://github.com/maraichr/codegraph.git
cd codegraph

# Start infrastructure (PostgreSQL, Neo4j, Valkey, MinIO)
docker compose up -d

# The API and worker services start automatically with hot-reload.
# API is available at http://localhost:8080
# Frontend dev server:
cd frontend && pnpm install && pnpm dev
```

The Docker Compose setup includes:
- **PostgreSQL 17** with pgvector extension
- **Neo4j** Community Edition with APOC plugin
- **Valkey** (Redis-compatible) for queuing and sessions
- **MinIO** for S3-compatible artifact storage
- **API + Worker** services with hot-reload via Air

## Development Setup

### Prerequisites

- **Go 1.25+**
- **Node.js 22+** with pnpm
- **Docker** and Docker Compose

### Building

```bash
# Build all binaries
make build

# Build individual services
make build-api
make build-worker
make build-mcp
```

### Testing

```bash
# Run all tests with race detection
make test

# With coverage report
make test-coverage
```

### Linting

```bash
make lint    # go vet + biome check
make fmt     # go fmt + biome format
```

### Code Generation

```bash
make generate          # SQLC + gqlgen
make generate-sqlc     # SQLC only
make generate-graphql  # gqlgen only
```

### Database Migrations

```bash
make migrate-up        # Apply all migrations
make migrate-down      # Rollback one migration
```

## Configuration

Copy `.env.example` to `.env` and configure:

```bash
cp .env.example .env
```

Key environment variables:
- `OPENROUTER_API_KEY` — API key for embedding provider
- `OPENROUTER_MODEL` — Embedding model (default: `openai/text-embedding-3-small`)
- `OPENROUTER_DIMENSIONS` — Embedding dimensions (default: `1024`)

Database and infrastructure settings are pre-configured in `docker-compose.yml` for local development.

## Project Structure

```
cmd/
  api/          # REST + GraphQL API server
  worker/       # Ingestion pipeline worker
  mcp/          # MCP tool server
  scheduler/    # Scheduled indexing jobs
internal/
  api/          # HTTP handlers, router, middleware
  analytics/    # Project analytics engine
  config/       # Environment configuration
  connector/    # Source connectors (GitLab, S3, ZIP)
  embedding/    # Vector embedding pipeline
  graph/        # Neo4j graph operations
  ingestion/    # Queue-based ingestion pipeline
  lineage/      # Lineage query engine
  mcp/          # MCP server, tools, session management
  parser/       # Language parsers (tsql, pgsql, asp, delphi, java, csharp)
  resolver/     # Cross-file symbol resolution
  store/        # PostgreSQL data access (SQLC-generated)
frontend/       # React 19 + TypeScript + Tailwind
migrations/     # PostgreSQL and Neo4j schema migrations
```

## Project Status

**Alpha** — Core functionality complete, approaching first public release.

- **Phase 1** ✅ — Foundation (DB, API, ingestion, connectors, parsers, frontend)
- **Phase 2** ✅ — Core parsers & graph (Neo4j, embeddings, symbol search, lineage, visualization)
- **Phase 3** ✅ — S3 source type, incremental indexing, column lineage, C# parser
- **Phase 4+** — .NET Roslyn sidecar, advanced lineage, RBAC

See [docs/STATUS.md](docs/STATUS.md) for detailed implementation tracking.

## License

[MIT](LICENSE)
