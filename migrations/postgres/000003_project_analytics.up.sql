-- Project analytics: pre-computed structural summaries and metrics
CREATE TABLE project_analytics (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    scope       TEXT NOT NULL,  -- 'project', 'source', 'schema', 'namespace', 'cluster', 'bridge'
    scope_id    TEXT NOT NULL,
    analytics   JSONB NOT NULL DEFAULT '{}',
    summary     TEXT,
    computed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (project_id, scope, scope_id)
);

CREATE INDEX idx_project_analytics_project_id ON project_analytics(project_id);
CREATE INDEX idx_project_analytics_scope ON project_analytics(scope);

-- GIN index on symbols.metadata for centrality/layer queries
CREATE INDEX IF NOT EXISTS idx_symbols_metadata ON symbols USING gin (metadata jsonb_path_ops);
