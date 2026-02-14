-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "vector";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- =============================================================================
-- RBAC Tables
-- =============================================================================

CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email       TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    avatar_url  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE api_keys (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    key_hash    TEXT NOT NULL UNIQUE,
    prefix      TEXT NOT NULL,
    scopes      TEXT[] NOT NULL DEFAULT '{}',
    expires_at  TIMESTAMPTZ,
    last_used   TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX idx_api_keys_prefix ON api_keys(prefix);

-- =============================================================================
-- Core Tables
-- =============================================================================

CREATE TABLE projects (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    description TEXT,
    settings    JSONB NOT NULL DEFAULT '{}',
    created_by  UUID REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_projects_slug ON projects(slug);

CREATE TABLE project_members (
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role        TEXT NOT NULL CHECK (role IN ('owner', 'admin', 'editor', 'viewer')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, user_id)
);

CREATE TABLE sources (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id      UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    source_type     TEXT NOT NULL CHECK (source_type IN ('git', 'database', 'filesystem', 'upload')),
    connection_uri  TEXT,
    config          JSONB NOT NULL DEFAULT '{}',
    last_synced_at  TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (project_id, name)
);

CREATE INDEX idx_sources_project_id ON sources(project_id);

CREATE TABLE index_runs (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id      UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    source_id       UUID REFERENCES sources(id) ON DELETE SET NULL,
    status          TEXT NOT NULL CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled')) DEFAULT 'pending',
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    files_processed INTEGER NOT NULL DEFAULT 0,
    symbols_found   INTEGER NOT NULL DEFAULT 0,
    edges_found     INTEGER NOT NULL DEFAULT 0,
    error_message   TEXT,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_index_runs_project_id ON index_runs(project_id);
CREATE INDEX idx_index_runs_status ON index_runs(status);

-- =============================================================================
-- Symbol Tables
-- =============================================================================

CREATE TABLE files (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    source_id   UUID NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    path        TEXT NOT NULL,
    language    TEXT NOT NULL,
    size_bytes  BIGINT NOT NULL DEFAULT 0,
    hash        TEXT NOT NULL,
    last_indexed_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (project_id, source_id, path)
);

CREATE INDEX idx_files_project_id ON files(project_id);
CREATE INDEX idx_files_language ON files(language);
CREATE INDEX idx_files_path_trgm ON files USING gin (path gin_trgm_ops);

CREATE TABLE symbols (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id      UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    file_id         UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    qualified_name  TEXT NOT NULL,
    kind            TEXT NOT NULL,
    language        TEXT NOT NULL,
    start_line      INTEGER NOT NULL,
    end_line        INTEGER NOT NULL,
    start_col       INTEGER,
    end_col         INTEGER,
    signature       TEXT,
    doc_comment     TEXT,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_symbols_project_id ON symbols(project_id);
CREATE INDEX idx_symbols_file_id ON symbols(file_id);
CREATE INDEX idx_symbols_kind ON symbols(kind);
CREATE INDEX idx_symbols_qualified_name ON symbols(qualified_name);
CREATE INDEX idx_symbols_name_trgm ON symbols USING gin (name gin_trgm_ops);
CREATE INDEX idx_symbols_qualified_name_trgm ON symbols USING gin (qualified_name gin_trgm_ops);

CREATE TABLE symbol_edges (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    source_id   UUID NOT NULL REFERENCES symbols(id) ON DELETE CASCADE,
    target_id   UUID NOT NULL REFERENCES symbols(id) ON DELETE CASCADE,
    edge_type   TEXT NOT NULL,
    metadata    JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (project_id, source_id, target_id, edge_type)
);

CREATE INDEX idx_symbol_edges_project_id ON symbol_edges(project_id);
CREATE INDEX idx_symbol_edges_source_id ON symbol_edges(source_id);
CREATE INDEX idx_symbol_edges_target_id ON symbol_edges(target_id);
CREATE INDEX idx_symbol_edges_edge_type ON symbol_edges(edge_type);

CREATE TABLE symbol_embeddings (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    symbol_id   UUID NOT NULL REFERENCES symbols(id) ON DELETE CASCADE UNIQUE,
    embedding   vector(1024) NOT NULL,
    model       TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_symbol_embeddings_hnsw ON symbol_embeddings
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);

-- =============================================================================
-- Seed Data
-- =============================================================================

INSERT INTO users (id, email, name) VALUES
    ('00000000-0000-0000-0000-000000000001', 'admin@codegraph.dev', 'Admin'),
    ('00000000-0000-0000-0000-000000000002', 'dev@codegraph.dev', 'Developer')
ON CONFLICT (email) DO NOTHING;
