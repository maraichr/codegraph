-- name: GetIndexRun :one
SELECT * FROM index_runs WHERE id = $1 LIMIT 1;

-- name: ListIndexRunsByProject :many
SELECT ir.* FROM index_runs ir
JOIN projects p ON ir.project_id = p.id
WHERE p.slug = $1
ORDER BY ir.created_at DESC
LIMIT $2 OFFSET $3;

-- name: CreateIndexRun :one
INSERT INTO index_runs (project_id, source_id, status)
VALUES ($1, $2, 'pending')
RETURNING *;

-- name: UpdateIndexRunStatus :exec
UPDATE index_runs
SET status = $2,
    started_at = CASE WHEN $2 = 'running' THEN now() ELSE started_at END,
    completed_at = CASE WHEN $2 IN ('completed', 'failed', 'cancelled') THEN now() ELSE completed_at END,
    error_message = $3
WHERE id = $1;

-- name: UpdateIndexRunStats :exec
UPDATE index_runs
SET files_processed = $2, symbols_found = $3, edges_found = $4
WHERE id = $1;

-- name: ListIndexRunsByProjectID :many
SELECT * FROM index_runs WHERE project_id = $1 ORDER BY created_at DESC LIMIT $2;
