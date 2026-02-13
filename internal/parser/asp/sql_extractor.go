package asp

import (
	"regexp"
	"strings"
)

// SQLFragment represents an extracted SQL string from ASP code.
type SQLFragment struct {
	SQL        string
	Line       int
	Confidence float64
}

var adoExecPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\.Execute\s*\(\s*"([^"]+)"\s*\)`),
	regexp.MustCompile(`(?i)\.Execute\s*\(\s*(.+?)\s*\)`),
	regexp.MustCompile(`(?i)\.Open\s+"([^"]+)"`),
	regexp.MustCompile(`(?i)\.Open\s+(.+?)[\s,]`),
	regexp.MustCompile(`(?i)\.CommandText\s*=\s*"([^"]+)"`),
}

// ExtractSQL finds SQL strings from ASP/VBScript code regions.
func ExtractSQL(code string) []SQLFragment {
	var fragments []SQLFragment

	lines := strings.Split(code, "\n")

	// Look for ADO execution patterns
	for i, line := range lines {
		for _, pat := range adoExecPatterns {
			matches := pat.FindStringSubmatch(line)
			if len(matches) >= 2 {
				sql := cleanSQL(matches[1])
				if looksLikeSQL(sql) {
					fragments = append(fragments, SQLFragment{
						SQL:        sql,
						Line:       i + 1,
						Confidence: 0.9,
					})
				}
			}
		}
	}

	// Look for multi-line SQL string concatenation patterns
	// sql = "SELECT" & _
	//       "FROM" & _
	//       "WHERE"
	fragments = append(fragments, extractConcatenatedSQL(lines)...)

	return fragments
}

func extractConcatenatedSQL(lines []string) []SQLFragment {
	var fragments []SQLFragment

	sqlAssign := regexp.MustCompile(`(?i)^\s*(?:str)?sql\s*=\s*"(.+)"`)
	sqlConcat := regexp.MustCompile(`(?i)^\s*(?:str)?sql\s*=\s*(?:str)?sql\s*&\s*"(.+)"`)
	sqlAppend := regexp.MustCompile(`(?i)^\s*"(.+)"`)

	var current strings.Builder
	startLine := 0
	inConcat := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if !inConcat {
			if m := sqlAssign.FindStringSubmatch(trimmed); len(m) >= 2 {
				current.Reset()
				current.WriteString(cleanSQL(m[1]))
				startLine = i + 1
				inConcat = strings.HasSuffix(trimmed, "& _") || strings.HasSuffix(trimmed, "&_")
				if !inConcat && looksLikeSQL(current.String()) {
					fragments = append(fragments, SQLFragment{
						SQL:        current.String(),
						Line:       startLine,
						Confidence: 0.8,
					})
					current.Reset()
				}
			}
		} else {
			if m := sqlConcat.FindStringSubmatch(trimmed); len(m) >= 2 {
				current.WriteString(" " + cleanSQL(m[1]))
			} else if m := sqlAppend.FindStringSubmatch(trimmed); len(m) >= 2 {
				current.WriteString(" " + cleanSQL(m[1]))
			}

			if !strings.HasSuffix(trimmed, "& _") && !strings.HasSuffix(trimmed, "&_") {
				inConcat = false
				if looksLikeSQL(current.String()) {
					fragments = append(fragments, SQLFragment{
						SQL:        current.String(),
						Line:       startLine,
						Confidence: 0.7,
					})
				}
				current.Reset()
			}
		}
	}

	return fragments
}

func cleanSQL(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, `"`)
	s = strings.ReplaceAll(s, `""`, `"`)
	return s
}

func looksLikeSQL(s string) bool {
	upper := strings.ToUpper(strings.TrimSpace(s))
	sqlKeywords := []string{"SELECT", "INSERT", "UPDATE", "DELETE", "CREATE", "ALTER", "DROP", "EXEC", "EXECUTE"}
	for _, kw := range sqlKeywords {
		if strings.HasPrefix(upper, kw) {
			return true
		}
	}
	return false
}
