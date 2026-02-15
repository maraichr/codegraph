# CodeGraph todo

## Indexing improvements (DNN Platform / complex codebases)

- [x] **Phase 3: C# → SQL cross-language** — Set `FromSymbol` on C# `[Table]`, DbSet, and inline SQL refs; add C# → T-SQL bridge rules; infer source from file symbols when `FromSymbol` empty.
- [x] **Phase 1: Migration-aware symbol consolidation** — Classify migration/schema files by path; `SkipColumnLineage` on `FileInput`; T-SQL parser skips `colRefs` for those files.
- [x] **Phase 2: Reduce direct copy edge volume** — Confidence in lineage edge metadata; optional `lineage_exclude_paths` in project settings; `GetProjectByID` for loading settings.
- [x] **Phase 4: ASP and JavaScript cross-language** — ASP SQL refs get `FromSymbol` from enclosing function/sub; add JS/TS → T-SQL bridge rules.
- [x] **Phase 5: DNN-specific** — Path heuristics for DNN Platform, Providers, Dnn.AdminExperience in migration classification.
- [x] **Documentation** — codegrapspec §6.6, §7.2, §6.8; ADR-001; this todo.

## MCP server

- [x] **Streamable HTTP transport** — Add MCP Go SDK; start Streamable HTTP listener in `cmd/mcp`; register `extract_subgraph` and `ask_codebase`; config `MCP_ADDR` (default `:8080`); graceful shutdown.

## Possible follow-ups

- Prefer canonical (non-migration) symbols when resolving FQNs in lineage (symbol metadata `is_migration`).
- Add `confidence` filtering in lineage queries (e.g. Neo4j filter edges below 0.7).
- PgSQL parser: support `SkipColumnLineage` for migration-classified files.
