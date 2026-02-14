---
name: codegraph
description: >
  Skill for developing the CodeGraph semantic codebase indexing engine. Use this skill whenever
  working on ANY part of the CodeGraph codebase — backend Go code, parsers, resolvers, API
  handlers, GraphQL schema, MCP tools, frontend React components, database migrations, Docker
  configuration, or tests. Also trigger when discussing CodeGraph architecture, planning new
  features, writing ADRs, designing MCP tool responses, or working on the ingestion pipeline.
  This skill encodes project conventions, spec-vs-implementation divergences, parser development
  patterns, and context-frugal design principles critical for agentic use. If the work touches
  the codegraph/ directory or references CodeGraph concepts (symbols, edges, lineage, index runs,
  sources, parsers, MCP tools), use this skill.
---

# CodeGraph Development Skill

CodeGraph is an enterprise-grade semantic indexing engine for large codebases. It extracts
dependency graphs, data lineage, and structural metadata across SQL Server, PostgreSQL, ASP
Classic, Delphi, .NET, and Java — then exposes this graph to LLMs via MCP tools on AWS Bedrock
AgentCore.

The system is designed for agentic consumption: LLMs use the MCP tools to autonomously research,
query, and reason about enterprise codebases (1,000+ repos, 100M+ LOC).

## Quick Orientation

```
codegraph/
├── cmd/                    # Entrypoints: api, worker, mcp, scheduler
├── internal/               # Core business logic (DO NOT import from outside)
│   ├── api/                #   REST handlers + GraphQL resolvers
│   ├── mcp/                #   MCP server + tool implementations
│   ├── ingestion/          #   Queue consumer, pipeline stages, connectors
│   ├── parser/             #   Language parsers (tsql, pgsql, dotnet, java, asp, delphi)
│   ├── resolver/           #   Cross-file and cross-language symbol resolution
│   ├── lineage/            #   Column-level lineage engine
│   ├── graph/              #   Neo4j client + sync
│   ├── embedding/          #   Bedrock embedding client
│   ├── store/              #   PostgreSQL (sqlc) + Valkey clients
│   ├── auth/               #   JWT, RBAC, API keys
│   └── config/             #   Configuration loading
├── pkg/                    # Shared packages (importable externally)
│   ├── apierr/             #   Structured error system (codes, catalog)
│   └── models/             #   Domain models (Symbol, Edge, etc.)
├── frontend/               # React 19 + TypeScript 5.9 + Vite 7.2
├── migrations/postgres/    # golang-migrate SQL files (AUTHORITATIVE schema)
├── migrations/neo4j/       # Cypher constraints + indexes
├── deploy/                 # Helm chart, Dockerfiles, docker-compose
└── test/                   # Golden files, integration tests, fixtures
```

## Critical: Spec vs Implementation Divergences

The spec document (`SPEC.md`) is aspirational. The **authoritative schema** is always the
migration files. Before writing any database-touching code, read
[references/spec-divergences.md](references/spec-divergences.md).

Key divergences to remember:
- Column is `source_type` (not `type`), values are lowercase: `git`, `database`, `filesystem`, `upload`
- `sources` has `connection_uri` column; no `status` or `last_commit_sha`
- `index_runs` uses flat columns (`files_processed`, `symbols_found`, `edges_found`), not JSONB `stats`
- `symbols` uses `qualified_name` (not `fqn`), with integer location columns (not JSONB)
- `symbol_edges` uses `source_id`/`target_id` (not `source_symbol`/`target_symbol`)
- GraphQL enums are uppercase (`GIT`, `DATABASE`), DB values are lowercase (`git`, `database`)

## Workflow: Before You Code

1. **Identify the area** — parser, API, frontend, infra, MCP?
2. **Read the relevant reference**:
   - Parser work → [references/parser-development.md](references/parser-development.md)
   - API/GraphQL → [references/api-conventions.md](references/api-conventions.md)
   - MCP tools → [references/mcp-tool-design.md](references/mcp-tool-design.md)
   - DB schema changes → [references/spec-divergences.md](references/spec-divergences.md)
   - Architecture decisions → [references/adr-index.md](references/adr-index.md)
3. **Check the actual migration** in `migrations/postgres/` — never trust the spec for column names
4. **Check STATUS.md** for what's implemented vs planned
5. **Write tests first** for parsers (golden file tests are mandatory)

## Technology Stack (Canonical Versions)

Backend: Go 1.25, chi v5, gqlgen, sqlc, pgx v5, neo4j-go-driver v5, valkey-go, mcp-go
Frontend: React 19, TypeScript 5.9, Vite 7.2, TanStack Query, Zustand v5, Tailwind v4, Biome 2.x
Infra: PostgreSQL 17+ with pgvector 0.8.0, Neo4j 2026.01, Valkey 8.1+, MinIO, Kubernetes 1.31+

## Parser Development

This is where most Phase 2 effort lands. Read [references/parser-development.md](references/parser-development.md) before any parser work.

Every parser implements:
```go
type Parser interface {
    Languages() []Language
    Parse(ctx context.Context, file *FileInput) (*ParseResult, error)
    ResolveReferences(ctx context.Context, refs []RawReference, symbolTable *SymbolTable) ([]ResolvedEdge, error)
}
```

Key rules:
- `FileInput.SkipColumnLineage` — when true, skip column-level lineage edges (used for migration files)
- All symbols get a `kind` from the unified enum (TABLE, VIEW, CLASS, METHOD, etc.)
- References are initially unresolved (`RawReference`) and resolved in a later phase
- Cross-language resolution bridges app code to SQL objects via schema-qualified matching
- Golden file tests are mandatory: `test/testdata/{language}/` contains input → expected output pairs

## Error Handling

Use `pkg/apierr` for all REST errors. Never return raw error strings.

```go
// Correct
apierr.NotFound("PROJECT_NOT_FOUND", "project not found")

// Wrong
http.Error(w, "not found", 404)
```

GraphQL errors use extensions: `{ "extensions": { "code": "PROJECT_NOT_FOUND" } }`

See `pkg/apierr/code.go` for the full error catalog.

## MCP Tool Design — Context-Frugal Principles

The MCP tools are the primary interface for agentic systems. Read
[references/mcp-tool-design.md](references/mcp-tool-design.md) for full guidance.

Core principles:
- **Progressive disclosure**: summary → standard → full verbosity tiers
- **Token budgets**: tools accept optional `max_tokens` to let agents control response size
- **Session awareness**: track seen symbols per session to deduplicate
- **Ranked results**: always return most relevant/impactful items first with counts of remaining
- **Stop-at filters**: lineage/impact tools accept `stop_at_kinds` to prune traversals
- **Symbol IDs in every response**: agents need IDs for follow-up tool calls

## Testing Patterns

- **Parsers**: Golden file tests (`test/testdata/{lang}/`), edge case tests, fuzz tests
- **API**: Integration tests with Docker Compose (PG + Neo4j + Valkey + MinIO + WireMock)
- **Frontend**: Vitest + Testing Library, mock API responses
- **E2E**: Upload ZIP → index → query → verify graph integrity

## Implementation Status

Phase 1 ✅ complete: CRUD APIs, GitLab connector, ZIP upload, PG schema, T-SQL parser,
PgSQL parser, error handling, ingestion queue, React frontend, Docker Compose dev env.

Phase 2 in progress: symbol search, Neo4j integration, embeddings, parse stage wiring,
.NET/Java/ASP/Delphi parsers, dependency graph API, graph visualization.

## Decision Log

See [references/adr-index.md](references/adr-index.md) for all architectural decisions.
Key decisions: Neo4j for graph, Roslyn sidecar for .NET, Cohere Embed v4, custom Go T-SQL
parser, pg_query_go for PostgreSQL, Cytoscape.js for visualization, Valkey Streams for queuing.
