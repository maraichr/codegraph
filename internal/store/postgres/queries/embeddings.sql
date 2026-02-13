-- name: UpsertSymbolEmbedding :exec
INSERT INTO symbol_embeddings (symbol_id, embedding, model)
VALUES ($1, $2, $3)
ON CONFLICT (symbol_id) DO UPDATE SET embedding = $2, model = $3, created_at = now();

-- name: ListSymbolsWithoutEmbeddings :many
SELECT s.* FROM symbols s
LEFT JOIN symbol_embeddings se ON s.id = se.symbol_id
WHERE s.project_id = $1 AND se.id IS NULL;

-- name: SemanticSearch :many
SELECT s.*, (se.embedding <=> @query_embedding::vector) AS distance
FROM symbols s
JOIN symbol_embeddings se ON s.id = se.symbol_id
WHERE s.project_id = @project_id
  AND (cardinality(@kinds::text[]) = 0 OR s.kind = ANY(@kinds::text[]))
ORDER BY se.embedding <=> @query_embedding::vector
LIMIT @lim;
