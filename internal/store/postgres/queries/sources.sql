-- name: GetSource :one
SELECT * FROM sources WHERE id = $1 LIMIT 1;

-- name: ListSourcesByProject :many
SELECT s.* FROM sources s
JOIN projects p ON s.project_id = p.id
WHERE p.slug = $1
ORDER BY s.created_at DESC
LIMIT $2 OFFSET $3;

-- name: CreateSource :one
INSERT INTO sources (project_id, name, source_type, connection_uri, config)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateSourceLastSynced :exec
UPDATE sources SET last_synced_at = now(), updated_at = now() WHERE id = $1;

-- name: DeleteSource :exec
DELETE FROM sources WHERE id = $1;

-- name: ListSourcesByProjectID :many
SELECT * FROM sources WHERE project_id = $1 ORDER BY created_at DESC;

-- name: UpdateSourceLastCommitSHA :exec
UPDATE sources SET last_commit_sha = $2, updated_at = now() WHERE id = $1;
