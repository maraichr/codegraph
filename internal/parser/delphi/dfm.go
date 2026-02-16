package delphi

import (
	"regexp"
	"strings"

	"github.com/maraichr/lattice/internal/parser"
)

// DFMComponent represents a component in a DFM file.
type DFMComponent struct {
	Name      string
	ClassName string
	Line      int
	SQL       []string // SQL strings from query components
}

// ParseDFM parses a Delphi DFM (text format) file.
func ParseDFM(content string, baseOffset int) ([]parser.Symbol, []parser.RawReference) {
	var symbols []parser.Symbol
	var refs []parser.RawReference

	components := extractComponents(content)

	for _, comp := range components {
		sym := parser.Symbol{
			Name:          comp.Name,
			QualifiedName: comp.ClassName + "." + comp.Name,
			Kind:          "variable", // DFM components are instance variables
			Language:      "delphi",
			StartLine:     comp.Line + baseOffset,
			EndLine:       comp.Line + baseOffset,
			Signature:     comp.ClassName,
		}
		symbols = append(symbols, sym)

		// If this is a query component, extract SQL references
		for _, sql := range comp.SQL {
			sqlRefs := extractDFMSQLRefs(sql, comp.Name, comp.Line+baseOffset)
			refs = append(refs, sqlRefs...)
		}
	}

	return symbols, refs
}

func extractComponents(content string) []DFMComponent {
	var components []DFMComponent

	// Match: object ComponentName: TClassName
	objectRe := regexp.MustCompile(`(?m)^\s*object\s+(\w+):\s*(\w+)`)
	sqlStringsRe := regexp.MustCompile(`(?i)(SQL\.Strings|SelectSQL\.Strings|SQL\.Text)\s*=\s*\(`)
	commandTextRe := regexp.MustCompile(`(?i)CommandText\s*=\s*'(.+?)'`)

	lines := strings.Split(content, "\n")

	var current *DFMComponent
	inSQLStrings := false
	var sqlBuilder strings.Builder

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if m := objectRe.FindStringSubmatch(trimmed); len(m) >= 3 {
			if current != nil {
				components = append(components, *current)
			}
			current = &DFMComponent{
				Name:      m[1],
				ClassName: m[2],
				Line:      i + 1,
			}
			continue
		}

		if trimmed == "end" && current != nil {
			components = append(components, *current)
			current = nil
			continue
		}

		// Detect SQL.Strings / SelectSQL.Strings / SQL.Text multi-line property
		if current != nil && sqlStringsRe.MatchString(trimmed) {
			inSQLStrings = true
			sqlBuilder.Reset()
			continue
		}

		// Detect CommandText = 'SQL string' (single-line)
		if current != nil {
			if m := commandTextRe.FindStringSubmatch(trimmed); len(m) >= 2 {
				current.SQL = append(current.SQL, m[1])
				continue
			}
		}

		if inSQLStrings {
			if trimmed == ")" {
				inSQLStrings = false
				if current != nil {
					current.SQL = append(current.SQL, sqlBuilder.String())
				}
			} else {
				// DFM SQL strings are like: 'SELECT * FROM table'
				cleaned := strings.Trim(trimmed, "'")
				cleaned = strings.TrimSuffix(cleaned, " +")
				sqlBuilder.WriteString(cleaned + " ")
			}
		}
	}

	if current != nil {
		components = append(components, *current)
	}

	return components
}

func extractDFMSQLRefs(sql, componentName string, line int) []parser.RawReference {
	var refs []parser.RawReference

	tablePatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bFROM\s+(\w+)`),
		regexp.MustCompile(`(?i)\bJOIN\s+(\w+)`),
		regexp.MustCompile(`(?i)\bINTO\s+(\w+)`),
		regexp.MustCompile(`(?i)\bUPDATE\s+(\w+)`),
	}

	for _, pat := range tablePatterns {
		for _, m := range pat.FindAllStringSubmatch(sql, -1) {
			if len(m) >= 2 {
				name := m[1]
				if !isSQLReserved(name) {
					refs = append(refs, parser.RawReference{
						FromSymbol:    componentName,
						ToName:        name,
						ReferenceType: "uses_table",
						Line:          line,
					})
				}
			}
		}
	}

	return refs
}

func isSQLReserved(s string) bool {
	reserved := map[string]bool{
		"SELECT": true, "FROM": true, "WHERE": true, "AND": true,
		"OR": true, "NOT": true, "NULL": true, "SET": true,
		"VALUES": true, "AS": true, "ON": true, "IN": true,
	}
	return reserved[strings.ToUpper(s)]
}
