# Spec vs Implementation Divergences

The spec (`SPEC.md`) was written as an aspirational design document. The **authoritative schema**
is always `migrations/postgres/000001_initial_schema.up.sql`. This file documents every known
divergence. Check here before writing any code that touches the database.

## Table of Contents

1. [Sources table](#sources)
2. [Index runs table](#index-runs)
3. [Symbols table](#symbols)
4. [Symbol edges table](#symbol-edges)
5. [Projects table](#projects)
6. [GraphQL enum mapping](#graphql-enum-mapping)
7. [Index run status values](#index-run-status-values)

---

## Sources

| Aspect | Spec says | Implementation has |
|---|---|---|
| Type column | `type` with values `GITLAB`, `S3`, `ZIP_UPLOAD` | `source_type` with values `git`, `database`, `filesystem`, `upload` |
| Connection | Part of JSONB `config` | Separate `connection_uri` TEXT column for git clone URLs |
| Status | `status` enum column (`ACTIVE`, `DISABLED`, `ERROR`) | No `status` column |
| Last commit | `last_commit_sha` TEXT column | No `last_commit_sha` column |
| Config | Rich JSONB with nested gitlab/S3/ZIP fields | Simpler JSONB `config` |

When writing source-related code:
- Use `source_type` not `type` in SQL queries
- Use lowercase values: `git`, `database`, `filesystem`, `upload`
- Use `connection_uri` for git clone URLs, not `config.gitlab_url`
- Do not assume `status` or `last_commit_sha` columns exist

## Index Runs

| Aspect | Spec says | Implementation has |
|---|---|---|
| Status values | 7-stage: `QUEUED`, `CLONING`, `PARSING`, `RESOLVING`, `GRAPH_BUILDING`, `EMBEDDING`, `COMPLETE`, `FAILED` | 5-value: `pending`, `running`, `completed`, `failed`, `cancelled` |
| Trigger | `trigger` enum column (`MANUAL`, `WEBHOOK`, `SCHEDULE`, `UPLOAD`) | No `trigger` column |
| Stats | JSONB `stats` with nested fields | Flat integer columns: `files_processed`, `symbols_found`, `edges_found` |
| Error | `error_message` TEXT | `error_message` TEXT (same) |

When writing index run code:
- Use lowercase status values: `pending`, `running`, `completed`, `failed`, `cancelled`
- Access stats as separate columns, not JSON: `index_runs.files_processed`
- Do not try to set `trigger` — it's not persisted
- The multi-stage pipeline status (CLONING → PARSING → ...) is aspirational for Phase 2+

## Symbols

| Aspect | Spec says | Implementation has |
|---|---|---|
| Qualified name | `fqn` TEXT | `qualified_name` TEXT |
| Language | Not a column (derived from file) | `language` TEXT column on symbols table |
| Location | JSONB `location` with `{start_line, end_line, start_col, end_col}` | Flat integer columns: `start_line`, `end_line`, `start_col`, `end_col` |
| Metadata | JSONB `metadata` | JSONB `metadata` (same) |

When writing symbol queries:
- Use `qualified_name` not `fqn` in SQL
- Use `s.start_line`, `s.end_line` etc., not `s.location->>'start_line'`
- The `language` column is available directly on symbols

## Symbol Edges

| Aspect | Spec says | Implementation has |
|---|---|---|
| Source reference | `source_symbol` UUID FK | `source_id` UUID FK |
| Target reference | `target_symbol` UUID FK | `target_id` UUID FK |
| Confidence | `confidence` REAL column | No `confidence` column (deferred) |
| Location | JSONB `location` with file_id reference | No `location` column (deferred) |

When writing edge queries:
- Use `source_id` / `target_id` not `source_symbol` / `target_symbol`
- Do not assume `confidence` or edge `location` columns exist
- Confidence scoring is planned for Phase 2 resolution engine

## Projects

| Aspect | Spec says | Implementation has |
|---|---|---|
| Owner | `owner_id` UUID NOT NULL (no FK) | `created_by` UUID nullable FK → `users(id)` |
| Settings | JSONB `settings` | JSONB `settings` (same) |

When writing project code:
- Use `created_by` not `owner_id`
- `created_by` is nullable and has a FK constraint to `users`

## GraphQL Enum Mapping

The GraphQL layer maps between DB lowercase values and API uppercase enums:

```
DB: git        → GraphQL: GIT
DB: database   → GraphQL: DATABASE
DB: filesystem → GraphQL: FILESYSTEM
DB: upload     → GraphQL: UPLOAD
```

The resolver layer handles this translation. When adding new source types:
1. Add lowercase value to the DB CHECK constraint via migration
2. Add uppercase value to GraphQL enum in schema
3. Add mapping in the resolver's enum conversion functions

## Index Run Status Values

```
DB values:     pending | running | completed | failed | cancelled
GraphQL enums: PENDING | RUNNING | COMPLETED | FAILED | CANCELLED
```

The spec's 7-stage statuses (QUEUED, CLONING, PARSING, etc.) may be implemented in Phase 2
as a separate `stage` column or as a sub-status within the `running` state.
