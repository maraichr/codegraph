# CodeGraph Roadmap

Current status: **Alpha** — Phases 1–3 complete, approaching first public release.

## Completed

### Phase 1: Foundation
- PostgreSQL schema + migrations (projects, sources, index runs, files, symbols, edges, embeddings, RBAC)
- REST API (17 endpoints) + GraphQL API (CRUD mutations/queries)
- T-SQL parser (custom recursive-descent) and PostgreSQL parser (pg_query_go)
- GitLab connector (PAT auth, shallow clone, webhooks) and ZIP upload connector
- Ingestion queue (Valkey Streams with consumer groups)
- React frontend (project management, sources, index runs, uploads)
- Docker Compose dev environment

### Phase 2: Core Parsers & Graph
- Neo4j integration + batched graph sync
- Cross-file symbol resolution engine
- Embeddings pipeline (configurable provider, pgvector cosine search)
- ASP Classic, Delphi, and Java (tree-sitter) parsers
- Lineage graph queries (Neo4j variable-length path traversals)
- Symbol search, semantic search, and impact analysis APIs
- Graph visualization (Cytoscape.js with multiple layouts)

### Phase 3: Lineage & Incremental Indexing
- S3 source connector
- Incremental indexing for Git sources
- Column-level lineage tracking
- C# parser (tree-sitter, EF/Dapper SQL extraction)
- MCP tool server (ask_codebase, extract_subgraph)

## In Progress

### Phase 4: Frontend Overhaul

The frontend is being redesigned for a comprehensive user experience.

**Dashboard & Analytics**
- [ ] Project overview dashboard with key metrics (symbol counts, edge density, coverage)
- [ ] Index run history with status timeline
- [ ] Parser coverage breakdown per project
- [ ] Source health indicators

**Lineage Visualization**
- [ ] Enhanced graph explorer with column-level lineage display
- [ ] Interactive lineage tracing (click a column, see full upstream/downstream path)
- [ ] Lineage diff view (compare lineage between index runs)
- [ ] Export lineage diagrams (SVG/PNG)

**Search & Navigation**
- [ ] Global symbol search with filters (kind, language, project)
- [ ] Code navigation (click-through from symbol to source file)
- [ ] Semantic search UI with relevance tuning
- [ ] Impact analysis explorer (select a symbol, see blast radius)
- [ ] Saved searches and bookmarks

### Phase 5: MCP & Intelligence
- [ ] Expand MCP tool suite (full 10-tool spec)
- [ ] Bedrock AgentCore registration
- [ ] Snapshot comparison tool
- [ ] WebSocket progress updates for long-running index runs
- [ ] Advanced graph queries (shortest path, clustering coefficients)

## Planned

### Phase 6: Production Hardening
- [ ] Helm chart + Kubernetes deployment
- [ ] RBAC implementation (project-level access control)
- [ ] Audit logging
- [ ] Observability (OpenTelemetry metrics + tracing)
- [ ] Performance optimization (query caching, batch processing)
- [ ] .NET Roslyn sidecar parser (full semantic analysis)

---

Have ideas or want to contribute? See [CONTRIBUTING.md](CONTRIBUTING.md) or open an issue.
