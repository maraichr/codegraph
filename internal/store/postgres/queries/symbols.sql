-- name: CreateSymbol :one
INSERT INTO symbols (project_id, file_id, name, qualified_name, kind, language, start_line, end_line, start_col, end_col, signature, doc_comment)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;

-- name: CountSymbolsByProject :one
SELECT count(*) FROM symbols WHERE project_id = $1;

-- name: DeleteSymbolsByFile :exec
DELETE FROM symbols WHERE file_id = $1;

-- name: GetSymbol :one
SELECT * FROM symbols WHERE id = $1;

-- name: SearchSymbols :many
SELECT * FROM symbols
WHERE project_id = (SELECT id FROM projects WHERE slug = @project_slug)
  AND (name ILIKE '%' || @query || '%' OR qualified_name ILIKE '%' || @query || '%')
  AND (cardinality(@kinds::text[]) = 0 OR kind = ANY(@kinds::text[]))
  AND (cardinality(@languages::text[]) = 0 OR language = ANY(@languages::text[]))
ORDER BY name
LIMIT @lim;

-- name: GetSymbolsByProject :many
SELECT * FROM symbols WHERE project_id = $1 ORDER BY qualified_name LIMIT $2 OFFSET $3;

-- name: ListSymbolsByProject :many
SELECT * FROM symbols WHERE project_id = $1;

-- name: ListSymbolsByFileIDs :many
SELECT * FROM symbols WHERE file_id = ANY($1::uuid[]);

-- name: GetSymbolByQualifiedName :one
SELECT * FROM symbols WHERE project_id = $1 AND qualified_name = $2;

-- name: ListSymbolsByNames :many
SELECT * FROM symbols WHERE project_id = $1 AND name = ANY($2::text[]);

-- name: DeleteSymbolsByFileID :exec
DELETE FROM symbols WHERE file_id = $1;

-- name: ListColumnSymbolsByProject :many
SELECT * FROM symbols WHERE project_id = $1 AND kind = 'column';
