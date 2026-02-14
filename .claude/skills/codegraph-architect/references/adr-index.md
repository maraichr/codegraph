# Architectural Decision Records

All significant technical decisions are documented here. When making a new decision that
affects architecture, add an ADR to `docs/adr/` and update this index.

## ADR Format

```markdown
# ADR-NNN: Title

## Status
Proposed | Accepted | Deprecated | Superseded by ADR-NNN

## Context
What problem are we solving? What constraints exist?

## Decision
What did we decide and why?

## Consequences
What are the trade-offs? What follow-up work is needed?
```

---

## Index

### ADR-001: Migration and Schema File Handling for Column Lineage

**Status**: Accepted (2026-02-15)

**Decision**: Classify migration/schema files and skip column-level lineage extraction
for them. Symbols are still extracted; only column-to-column edges are omitted.

**Rationale**: Migrations document schema evolution, not runtime data flow. Extracting
lineage from every `INSERT...SELECT` in migration scripts floods the graph with
low-value `direct_copy` edges that duplicate logical relationships and slow lineage queries.

**Implementation**: `FileInput.SkipColumnLineage` flag, checked by T-SQL parser (and
future SQL parsers) when building `colRefs`. Classification by path patterns and project
`settings.lineage_exclude_paths`.

---

### Technology Decisions (from Spec §21)

These were decided during Phase 1 and are documented in the spec's Open Questions table.
Recording them here for completeness:

**Graph database: Neo4j 2026.01** — Native graph traversal for dependency/lineage queries.
Calendar versioning, GQL conformance. Enterprise edition for clustering in production;
Community edition acceptable for development.

**T-SQL parser: Custom Go recursive-descent** — ANTLR T-SQL grammars are incomplete for
enterprise T-SQL (dynamic SQL, linked servers, synonyms, four-part names). Custom parser
gives full control. Implemented in Phase 1.

**PostgreSQL parser: pg_query_go** — Go binding for PostgreSQL's own C parser (`libpg_query`).
Zero ambiguity — it's the same parser PostgreSQL uses. Implemented in Phase 1.

**.NET parser: Roslyn sidecar** — Only option for full semantic analysis (type resolution,
overload resolution, generic specialization). Runs as a separate .NET process spawned by
the Go worker. Tree-sitter C# is used as a lightweight fallback for Phase 1.

**Embedding model: Cohere Embed v4 on Bedrock** — Multimodal, 100+ languages, 1024
dimensions. Better multilingual support than Titan v2. Available on Bedrock.

**Frontend graph visualization: Cytoscape.js** — Best combination of performance at scale
(tested with 50K+ nodes), feature set (layouts, filtering, context menus), and React
integration.

**Queue system: Valkey Streams (8.1+)** — Valkey is the Linux Foundation BSD-3 fork of
Redis. API-compatible with Redis 7.x. Streams provide consumer groups for distributed
worker processing. Lighter weight than RabbitMQ or NATS for our use case.

**API router: chi v5** — stdlib-compatible, lightweight, middleware-friendly. No magic.

**SQL generation: sqlc** — Compile-time SQL validation, generated type-safe Go code.
No ORM overhead. Excellent for complex queries.

**Delphi DFM binary parsing: Custom Go parser** — Avoids dependency on Delphi toolchain.
DFM binary format is well-documented and straightforward to parse.

---

## Decision Template

When proposing a new ADR, create a file `docs/adr/ADR-NNN-short-title.md` and add it to
this index. Use the next available number. All decisions should be reviewed before
implementation begins.
