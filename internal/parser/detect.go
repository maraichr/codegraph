package parser

import (
	"strings"
)

// DetectDialect determines whether a .sql file is T-SQL or PostgreSQL.
func DetectDialect(content []byte) string {
	text := strings.ToUpper(string(content))

	tsqlScore := 0
	pgsqlScore := 0

	// T-SQL indicators
	if strings.Contains(text, "\nGO\n") || strings.Contains(text, "\nGO\r\n") || strings.HasSuffix(text, "\nGO") {
		tsqlScore += 10 // GO batch separator is definitive
	}
	for _, kw := range []string{"DECLARE @", "SET @", "NVARCHAR", "VARCHAR(MAX)", "BIT", "IDENTITY(",
		"EXEC ", "EXECUTE ", "SP_", "NOCOUNT", "BEGIN TRY", "BEGIN CATCH",
		"@@ROWCOUNT", "@@ERROR", "@@IDENTITY", "GETDATE()", "ISNULL(",
		"CHARINDEX(", "TOP ", "WITH (NOLOCK)", "CROSS APPLY", "OUTER APPLY"} {
		if strings.Contains(text, kw) {
			tsqlScore += 2
		}
	}

	// PostgreSQL indicators
	for _, kw := range []string{"$$", "LANGUAGE PLPGSQL", "LANGUAGE SQL", "RETURNS SETOF",
		"RETURNS TABLE", "CREATE EXTENSION", "CREATE SCHEMA", "SERIAL", "BIGSERIAL",
		"BOOLEAN", "TEXT NOT NULL", "TIMESTAMPTZ", "UUID", "JSONB",
		"::TEXT", "::INTEGER", "::UUID", "ILIKE", "SIMILAR TO",
		"CREATE OR REPLACE FUNCTION", "RAISE NOTICE", "RAISE EXCEPTION",
		"PERFORM ", "IMMUTABLE", "STABLE", "VOLATILE"} {
		if strings.Contains(text, kw) {
			pgsqlScore += 2
		}
	}

	if tsqlScore > pgsqlScore {
		return "tsql"
	}
	if pgsqlScore > tsqlScore {
		return "pgsql"
	}

	// Default to pgsql for ambiguous cases
	return "pgsql"
}
