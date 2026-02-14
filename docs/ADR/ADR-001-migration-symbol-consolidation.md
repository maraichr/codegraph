# ADR-001: Migration and schema file handling for column lineage

## Status

Accepted (2026-02-15)

## Context

Large codebases (e.g. DNN Platform) contain many SQL migration or schema scripts that define and alter the same tables across dozens of files. Extracting column-level lineage from every `INSERT...SELECT`, `UPDATE SET col = x`, and `SELECT col` in these files produces a very large number of `direct_copy` and `transforms_to` edges. These edges:

- Duplicate logical relationships (the same table appears in many files)
- Add little value for "where does this column's data flow at runtime"
- Clutter the graph and slow lineage queries

Migrations document **schema evolution**, not **runtime data flow**. Runtime lineage is better captured from stored procedures, views, and application code that execute in production.

## Decision

1. **Classify migration/schema files** in the parse stage by path and optional project settings:
   - Paths containing `Database/`, `Migrations/`, `Scripts/`, or suffixes `.Install.sql` / `.Upgrade.sql`
   - DNN-style paths: `DNN Platform/`, `Dnn.AdminExperience/`, `Providers/`
   - Project `settings.lineage_exclude_paths` (glob or substring patterns)

2. **Skip column-level lineage extraction** for classified files: set `FileInput.SkipColumnLineage = true` and have the T-SQL (and future PgSQL) parser omit appending to `colRefs` for SELECT/INSERT/UPDATE/SET/MERGE when this flag is set.

3. **Keep symbol extraction** for migration files (tables, views, procedures are still indexed) so that the rest of the graph and search remain complete.

4. **Optional future work:** Prefer "canonical" symbols over migration-defined duplicates when resolving FQNs (e.g. via symbol metadata `is_migration` and lineage resolution ordering). Not implemented in the initial change.

## Consequences

- Fewer column edges and a cleaner lineage graph for large repos with many migration scripts.
- Runtime-focused lineage (procs, views, app code) is unchanged and remains the primary signal.
- Project maintainers can add `lineage_exclude_paths` in project settings to exclude additional paths without code changes.
