-- Phase 3: S3 source type, incremental indexing, column lineage

-- Add S3 source type
ALTER TABLE sources DROP CONSTRAINT IF EXISTS sources_source_type_check;
ALTER TABLE sources ADD CONSTRAINT sources_source_type_check
    CHECK (source_type IN ('git', 'database', 'filesystem', 'upload', 's3'));

-- Track last commit SHA for incremental indexing
ALTER TABLE sources ADD COLUMN IF NOT EXISTS last_commit_sha TEXT;

-- GIN index on edge metadata for column lineage queries
CREATE INDEX IF NOT EXISTS idx_symbol_edges_metadata
    ON symbol_edges USING gin (metadata jsonb_path_ops);
