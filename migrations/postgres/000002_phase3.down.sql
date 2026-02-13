-- Revert Phase 3 changes

DROP INDEX IF EXISTS idx_symbol_edges_metadata;

ALTER TABLE sources DROP COLUMN IF EXISTS last_commit_sha;

ALTER TABLE sources DROP CONSTRAINT IF EXISTS sources_source_type_check;
ALTER TABLE sources ADD CONSTRAINT sources_source_type_check
    CHECK (source_type IN ('git', 'database', 'filesystem', 'upload'));
