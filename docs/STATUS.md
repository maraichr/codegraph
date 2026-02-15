# Lattice Implementation Status

Tracks what's built vs. planned. See `latticespec.md` §20 for the full roadmap.

## Phase 1: Foundation — ✅ COMPLETE

| Component | Status | Notes |
|---|---|---|
| PostgreSQL schema + migrations | ✅ | All tables: projects, sources, index_runs, files, symbols, symbol_edges, symbol_embeddings, RBAC (users, project_members, api_keys) |
| REST API (17 endpoints) | ✅ | Projects CRUD, sources CRUD, index runs, upload, webhooks, health/readiness |
| GraphQL API (CRUD) | ✅ | Mutations: createProject, updateProject, deleteProject, createSource, deleteSource, triggerIndexRun. Queries: projects, project |
| Centralized error handling | ✅ | `pkg/apierr` — structured codes, catalog, wire format `{"error":{"code":"...","message":"..."}}` |
| T-SQL parser | ✅ | Custom recursive-descent (tables, views, stored procedures, basic lineage) |
| PostgreSQL parser | ✅ | pg_query_go/v6 |
| GitLab connector | ✅ | PAT auth, shallow clone, webhook receiver |
| ZIP upload connector | ✅ | MinIO storage, zip-slip protection |
| Ingestion queue | ✅ | Valkey Streams with consumer groups |
| React frontend | ✅ | Project management, sources, index runs, upload, error states (ErrorState, ToastContainer, ApiError) |
| Docker Compose dev env | ✅ | PostgreSQL, Neo4j, Valkey, MinIO |
| Ingestion pipeline stages | ✅ | CloneStage (ZIP extract + git clone) and ParseStage wired with SQL dialect router |
| GraphQL symbol/lineage queries | ✅ | Symbol, SearchSymbols, File, SymbolEdge resolvers + Project field resolvers (lineageGraph deferred to Phase 2) |

## Phase 2: Core Parsers & Graph — ✅ COMPLETE

| Component | Status | Notes |
|---|---|---|
| Neo4j integration + graph sync | ✅ | `internal/graph/` — client, batched UNWIND sync (symbols, edges, files), Cypher lineage queries |
| Cross-file symbol resolution engine | ✅ | `internal/resolver/` — FQN, short-name, case-insensitive matching; ResolveStage in pipeline |
| Embeddings pipeline (Cohere Embed v4) | ✅ | `internal/embedding/` — AWS Bedrock client, batch embed (96/call), pgvector cosine search; optional stage |
| Lineage graph query | ✅ | Neo4j variable-length path traversals (upstream/downstream/both), REST + GraphQL endpoints |
| REST API: symbol search, lineage, impact | ✅ | `GET /symbols/{id}/lineage`, `GET /symbols/{id}/impact`, `GET /projects/{slug}/symbols` |
| REST API: semantic search | ✅ | `POST /projects/{slug}/search/semantic` — embeds query via Bedrock, pgvector kNN |
| GraphQL: semanticSearch query | ✅ | `semanticSearch(projectSlug, query, kinds, topK)` → `[SemanticSearchResult]` |
| Graph visualization (Cytoscape.js) | ✅ | GraphExplorer page, kind-colored nodes, dagre/cose/breadthfirst layouts, filters, node detail panel |
| ASP Classic parser | ✅ | `internal/parser/asp/` — VBScript regions, Function/Sub/Class extraction, ADO SQL extraction, includes |
| Delphi parser | ✅ | `internal/parser/delphi/` — Pascal units, classes, DFM component-SQL extraction, `{$I}` includes |
| Java parser (tree-sitter) | ✅ | `internal/parser/java/` — classes, interfaces, enums, methods, fields; Spring/JPA annotation handling |
| .NET parser (Roslyn sidecar) | ❌ | Deferred to Phase 3 |

Pipeline is now fully wired: **Clone → Parse → Resolve → GraphBuild → Embed**

Parser registry: `.sql`, `.asp`, `.pas`, `.dfm`, `.dpr`, `.java`

## Phase 3–5

See `latticespec.md` §20. Next up: .NET Roslyn parser, advanced lineage features.

## Legend

- ✅ Complete
- ⚠️ Partial — usable but incomplete
- ❌ Not started
