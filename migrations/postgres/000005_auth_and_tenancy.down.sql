-- 000005_auth_and_tenancy.down.sql

DROP INDEX IF EXISTS idx_projects_tenant_id;
ALTER TABLE projects DROP COLUMN IF EXISTS tenant_id;
DROP TABLE IF EXISTS memberships;
ALTER TABLE users DROP COLUMN IF EXISTS sub;
DROP TABLE IF EXISTS tenants;
