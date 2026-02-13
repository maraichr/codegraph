-- name: GetProject :one
SELECT * FROM projects WHERE slug = $1 LIMIT 1;

-- name: ListProjects :many
SELECT * FROM projects ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: CreateProject :one
INSERT INTO projects (name, slug, description, created_by)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateProject :one
UPDATE projects
SET name = $2, description = $3, settings = $4, updated_at = now()
WHERE slug = $1
RETURNING *;

-- name: DeleteProject :exec
DELETE FROM projects WHERE slug = $1;

-- name: CountProjects :one
SELECT count(*) FROM projects;
