-- name: UpsertProjectAnalytics :one
INSERT INTO project_analytics (project_id, scope, scope_id, analytics, summary, computed_at)
VALUES ($1, $2, $3, $4, $5, now())
ON CONFLICT (project_id, scope, scope_id) DO UPDATE
SET analytics = EXCLUDED.analytics,
    summary = EXCLUDED.summary,
    computed_at = now()
RETURNING *;

-- name: GetProjectAnalytics :one
SELECT * FROM project_analytics
WHERE project_id = $1 AND scope = $2 AND scope_id = $3;

-- name: ListProjectAnalyticsByScope :many
SELECT * FROM project_analytics
WHERE project_id = $1 AND scope = $2
ORDER BY scope_id;

-- name: ListAllProjectAnalytics :many
SELECT * FROM project_analytics
WHERE project_id = $1
ORDER BY scope, scope_id;

-- name: DeleteProjectAnalytics :exec
DELETE FROM project_analytics WHERE project_id = $1;

-- Degree computation: count in-degree and out-degree per symbol
-- name: GetSymbolDegrees :many
SELECT
    s.id,
    COALESCE(inc.cnt, 0)::int AS in_degree,
    COALESCE(outc.cnt, 0)::int AS out_degree
FROM symbols s
LEFT JOIN (
    SELECT se.target_id, count(*) AS cnt
    FROM symbol_edges se WHERE se.project_id = @project_id
    GROUP BY se.target_id
) inc ON s.id = inc.target_id
LEFT JOIN (
    SELECT se2.source_id, count(*) AS cnt
    FROM symbol_edges se2 WHERE se2.project_id = @project_id
    GROUP BY se2.source_id
) outc ON s.id = outc.source_id
WHERE s.project_id = @project_id;

-- Update symbol metadata with computed analytics (degree, pagerank, layer)
-- name: UpdateSymbolMetadata :exec
UPDATE symbols
SET metadata = metadata || @analytics_json::jsonb,
    updated_at = now()
WHERE id = @symbol_id;

-- Batch update symbol metadata for a set of symbols
-- name: BatchUpdateSymbolMetadata :exec
UPDATE symbols
SET metadata = metadata || @analytics_json::jsonb,
    updated_at = now()
WHERE id = ANY(@symbol_ids::uuid[]);

-- Get edge list for PageRank computation
-- name: GetEdgeList :many
SELECT source_id, target_id FROM symbol_edges WHERE project_id = $1;

-- Cross-language bridge query: edges where source and target have different languages
-- name: GetCrossLanguageBridges :many
SELECT
    s1.language AS source_language,
    s2.language AS target_language,
    e.edge_type,
    count(*) AS edge_count,
    array_agg(DISTINCT s1.id) AS source_symbol_ids,
    array_agg(DISTINCT s2.id) AS target_symbol_ids
FROM symbol_edges e
JOIN symbols s1 ON e.source_id = s1.id
JOIN symbols s2 ON e.target_id = s2.id
WHERE e.project_id = $1 AND s1.language != s2.language
GROUP BY s1.language, s2.language, e.edge_type
ORDER BY edge_count DESC;

-- Project-level aggregate stats
-- name: GetProjectSymbolStats :one
SELECT
    count(*) AS total_symbols,
    count(DISTINCT language) AS language_count,
    count(DISTINCT kind) AS kind_count,
    count(DISTINCT file_id) AS file_count
FROM symbols WHERE project_id = $1;

-- Symbols grouped by language
-- name: GetSymbolCountsByLanguage :many
SELECT language, count(*) AS cnt
FROM symbols WHERE project_id = $1
GROUP BY language ORDER BY cnt DESC;

-- Symbols grouped by kind
-- name: GetSymbolCountsByKind :many
SELECT kind, count(*) AS cnt
FROM symbols WHERE project_id = $1
GROUP BY kind ORDER BY cnt DESC;

-- Top symbols by in-degree (most depended-upon)
-- name: TopSymbolsByInDegree :many
SELECT s.*, (s.metadata->>'in_degree')::int AS in_degree
FROM symbols s
WHERE s.project_id = $1
  AND s.metadata ? 'in_degree'
  AND (s.metadata->>'in_degree')::int > 0
ORDER BY (s.metadata->>'in_degree')::int DESC
LIMIT $2;

-- Top symbols by PageRank
-- name: TopSymbolsByPageRank :many
SELECT s.*, (s.metadata->>'pagerank')::float AS pagerank
FROM symbols s
WHERE s.project_id = $1
  AND s.metadata ? 'pagerank'
ORDER BY (s.metadata->>'pagerank')::float DESC
LIMIT $2;

-- Symbols by layer
-- name: GetSymbolsByLayer :many
SELECT * FROM symbols
WHERE project_id = $1
  AND metadata->>'layer' = $2
ORDER BY qualified_name
LIMIT $3 OFFSET $4;

-- Count symbols by layer
-- name: CountSymbolsByLayer :many
SELECT metadata->>'layer' AS layer, count(*) AS cnt
FROM symbols
WHERE project_id = $1
  AND metadata ? 'layer'
GROUP BY metadata->>'layer'
ORDER BY cnt DESC;

-- Source-level stats
-- name: GetSourceSymbolStats :many
SELECT
    f.source_id,
    count(DISTINCT s.id) AS symbol_count,
    count(DISTINCT f.id) AS file_count,
    count(DISTINCT s.language) AS language_count,
    array_agg(DISTINCT s.language) AS languages,
    array_agg(DISTINCT s.kind) AS kinds
FROM symbols s
JOIN files f ON s.file_id = f.id
WHERE s.project_id = $1
GROUP BY f.source_id;

-- Parser coverage: total files vs. files with at least one symbol per source
-- name: GetParserCoverage :many
SELECT
    f.source_id,
    count(DISTINCT f.id) AS total_files,
    count(DISTINCT s.file_id) AS parsed_files
FROM files f
LEFT JOIN symbols s ON f.id = s.file_id
WHERE f.project_id = $1
GROUP BY f.source_id;

-- Bridge coverage stats: confidence metrics for cross-language edges
-- name: GetBridgeCoverageStats :one
SELECT
    count(*) FILTER (WHERE e.metadata ? 'confidence') AS edges_with_confidence,
    COALESCE(avg((e.metadata->>'confidence')::float) FILTER (WHERE e.metadata ? 'confidence'), 0) AS avg_confidence,
    count(*) FILTER (WHERE e.metadata ? 'confidence' AND (e.metadata->>'confidence')::float < 0.8) AS low_confidence_edges,
    count(*) AS total_cross_lang_edges
FROM symbol_edges e
JOIN symbols s1 ON e.source_id = s1.id
JOIN symbols s2 ON e.target_id = s2.id
WHERE e.project_id = $1 AND s1.language != s2.language;

-- Namespace-level stats (extract namespace from qualified_name)
-- name: GetNamespaceStats :many
SELECT
    CASE
        WHEN position('.' IN qualified_name) > 0
        THEN left(qualified_name, length(qualified_name) - length(name) - 1)
        ELSE '(root)'
    END AS namespace,
    count(*) AS symbol_count,
    array_agg(DISTINCT kind) AS kinds,
    array_agg(DISTINCT language) AS languages
FROM symbols
WHERE project_id = $1
GROUP BY namespace
HAVING count(*) >= 2
ORDER BY symbol_count DESC
LIMIT $2;
