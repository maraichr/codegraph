-- Deduplicate existing symbols before adding constraint:
-- Keep the symbol from the latest file (highest path lexicographically,
-- since DNN files are versioned like 01.00.00, 10.02.03).
DELETE FROM symbols
WHERE id NOT IN (
    SELECT DISTINCT ON (s.project_id, s.qualified_name, s.kind) s.id
    FROM symbols s
    JOIN files f ON s.file_id = f.id
    ORDER BY s.project_id, s.qualified_name, s.kind, f.path DESC, s.id DESC
);

-- Add unique constraint
CREATE UNIQUE INDEX IF NOT EXISTS idx_symbols_project_qname_kind
    ON symbols (project_id, qualified_name, kind);
