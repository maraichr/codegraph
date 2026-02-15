-- 000005_auth_and_tenancy.up.sql
-- Multi-tenancy + OIDC auth support

-- =============================================================================
-- Tenants
-- =============================================================================

CREATE TABLE tenants (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name       TEXT NOT NULL,
    slug       TEXT NOT NULL UNIQUE,
    settings   JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Seed default tenant
INSERT INTO tenants (id, name, slug) VALUES
    ('00000000-0000-0000-0000-000000000099', 'Default', 'default');

-- =============================================================================
-- Users: add OIDC subject identifier
-- =============================================================================

ALTER TABLE users ADD COLUMN IF NOT EXISTS sub TEXT UNIQUE;

-- =============================================================================
-- Memberships (tenant-scoped roles)
-- =============================================================================

CREATE TABLE memberships (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_sub  TEXT NOT NULL,
    role      TEXT NOT NULL CHECK (role IN ('owner', 'admin', 'member', 'viewer')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, user_sub)
);

CREATE INDEX idx_memberships_user_sub ON memberships(user_sub);

-- =============================================================================
-- Projects: add tenant_id
-- =============================================================================

ALTER TABLE projects ADD COLUMN tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE;

-- Migrate existing projects to default tenant
UPDATE projects SET tenant_id = '00000000-0000-0000-0000-000000000099' WHERE tenant_id IS NULL;

-- Now make it NOT NULL
ALTER TABLE projects ALTER COLUMN tenant_id SET NOT NULL;

CREATE INDEX idx_projects_tenant_id ON projects(tenant_id);
