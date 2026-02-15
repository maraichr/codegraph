-- name: GetTenant :one
SELECT * FROM tenants WHERE slug = $1 LIMIT 1;

-- name: GetTenantByID :one
SELECT * FROM tenants WHERE id = $1 LIMIT 1;

-- name: ListTenants :many
SELECT * FROM tenants ORDER BY created_at DESC;

-- name: CreateTenant :one
INSERT INTO tenants (name, slug, settings)
VALUES ($1, $2, $3)
RETURNING *;

-- name: DeleteTenant :exec
DELETE FROM tenants WHERE id = $1;

-- name: GetMembership :one
SELECT * FROM memberships WHERE tenant_id = $1 AND user_sub = $2 LIMIT 1;

-- name: ListMembershipsByTenant :many
SELECT * FROM memberships WHERE tenant_id = $1 ORDER BY created_at DESC;

-- name: CreateMembership :one
INSERT INTO memberships (tenant_id, user_sub, role)
VALUES ($1, $2, $3)
RETURNING *;

-- name: DeleteMembership :exec
DELETE FROM memberships WHERE tenant_id = $1 AND user_sub = $2;

-- name: CheckTenantAccess :one
SELECT EXISTS(
    SELECT 1 FROM memberships WHERE tenant_id = $1 AND user_sub = $2
) AS has_access;
