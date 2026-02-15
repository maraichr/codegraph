package asp

import (
	"regexp"
	"strings"

	"github.com/maraichr/codegraph/internal/parser"
)

// Parser implements a parser for ASP Classic (VBScript) files.
type Parser struct{}

func New() *Parser {
	return &Parser{}
}

func (p *Parser) Languages() []string {
	return []string{"asp", "aspx"}
}

func (p *Parser) Parse(input parser.FileInput) (*parser.ParseResult, error) {
	content := string(input.Content)

	var symbols []parser.Symbol
	var refs []parser.RawReference

	// Extract ASP.NET directives from <%@ ... %> blocks
	dirRefs := extractDirectives(content)
	refs = append(refs, dirRefs...)

	// Extract VBScript regions from <% ... %>
	regions := extractScriptRegions(content)

	for _, region := range regions {
		// Parse VBScript constructs
		syms, rfs := parseVBScript(region.code, region.startLine)
		symbols = append(symbols, syms...)
		refs = append(refs, rfs...)

		// Extract embedded SQL from ADO patterns; set FromSymbol to enclosing function/sub for cross-language resolution
		sqlFragments := ExtractSQL(region.code)
		for _, frag := range sqlFragments {
			sqlRefs := extractSQLRefs(frag.SQL, frag.Line+region.startLine)
			line := frag.Line + region.startLine
			fromSymbol := enclosingSymbol(line, syms)
			for i := range sqlRefs {
				sqlRefs[i].FromSymbol = fromSymbol
				sqlRefs[i].ToQualified = "dbo." + sqlRefs[i].ToName
			}
			refs = append(refs, sqlRefs...)
		}
	}

	// Parse include directives
	includes := parseIncludes(content)
	refs = append(refs, includes...)

	return &parser.ParseResult{
		Symbols:    symbols,
		References: refs,
	}, nil
}

type scriptRegion struct {
	code      string
	startLine int
}

func extractScriptRegions(content string) []scriptRegion {
	var regions []scriptRegion

	// Match <% ... %> blocks (non-greedy)
	re := regexp.MustCompile(`(?s)<%([^=].*?)%>`)
	matches := re.FindAllStringSubmatchIndex(content, -1)

	for _, loc := range matches {
		code := content[loc[2]:loc[3]]
		startLine := strings.Count(content[:loc[2]], "\n") + 1
		regions = append(regions, scriptRegion{code: code, startLine: startLine})
	}

	return regions
}

func parseVBScript(code string, baseOffset int) ([]parser.Symbol, []parser.RawReference) {
	var symbols []parser.Symbol
	var refs []parser.RawReference

	lines := strings.Split(code, "\n")

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		lineNum := baseOffset + i

		// Function Name(params)
		if strings.HasPrefix(lower, "function ") || strings.HasPrefix(lower, "public function ") || strings.HasPrefix(lower, "private function ") {
			name, sig := parseProcDecl(trimmed)
			if name != "" {
				symbols = append(symbols, parser.Symbol{
					Name:          name,
					QualifiedName: name,
					Kind:          "function",
					Language:      "asp",
					StartLine:     lineNum,
					EndLine:       findEndLine(lines, i, "function"),
					Signature:     sig,
				})
			}
		}

		// Sub Name(params)
		if strings.HasPrefix(lower, "sub ") || strings.HasPrefix(lower, "public sub ") || strings.HasPrefix(lower, "private sub ") {
			name, sig := parseProcDecl(trimmed)
			if name != "" {
				symbols = append(symbols, parser.Symbol{
					Name:          name,
					QualifiedName: name,
					Kind:          "procedure",
					Language:      "asp",
					StartLine:     lineNum,
					EndLine:       findEndLine(lines, i, "sub"),
					Signature:     sig,
				})
			}
		}

		// Class Name
		if strings.HasPrefix(lower, "class ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				name := parts[1]
				symbols = append(symbols, parser.Symbol{
					Name:          name,
					QualifiedName: name,
					Kind:          "class",
					Language:      "asp",
					StartLine:     lineNum,
					EndLine:       findEndLine(lines, i, "class"),
				})
			}
		}

		// Const declarations
		if strings.HasPrefix(lower, "const ") || strings.HasPrefix(lower, "public const ") || strings.HasPrefix(lower, "private const ") {
			name := parseConstDecl(trimmed)
			if name != "" {
				symbols = append(symbols, parser.Symbol{
					Name:          name,
					QualifiedName: name,
					Kind:          "constant",
					Language:      "asp",
					StartLine:     lineNum,
					EndLine:       lineNum,
				})
			}
		}

		// Server.CreateObject references
		if strings.Contains(lower, "server.createobject") {
			re := regexp.MustCompile(`(?i)Server\.CreateObject\s*\(\s*"([^"]+)"\s*\)`)
			if m := re.FindStringSubmatch(trimmed); len(m) >= 2 {
				refs = append(refs, parser.RawReference{
					ToName:        m[1],
					ReferenceType: "references",
					Line:          lineNum,
				})
			}
		}
	}

	return symbols, refs
}

func parseProcDecl(line string) (name, signature string) {
	// Remove access modifier
	lower := strings.ToLower(line)
	for _, prefix := range []string{"public ", "private "} {
		if strings.HasPrefix(lower, prefix) {
			line = line[len(prefix):]
			break
		}
	}

	// Remove Function/Sub keyword
	lower = strings.ToLower(line)
	for _, prefix := range []string{"function ", "sub "} {
		if strings.HasPrefix(lower, prefix) {
			line = line[len(prefix):]
			break
		}
	}

	// Extract name and params
	if idx := strings.Index(line, "("); idx >= 0 {
		name = strings.TrimSpace(line[:idx])
		endIdx := strings.Index(line, ")")
		if endIdx > idx {
			signature = line[idx : endIdx+1]
		}
	} else {
		name = strings.TrimSpace(line)
	}

	return name, signature
}

func parseConstDecl(line string) string {
	lower := strings.ToLower(line)
	for _, prefix := range []string{"public const ", "private const ", "const "} {
		if strings.HasPrefix(lower, prefix) {
			rest := line[len(prefix):]
			if idx := strings.Index(rest, "="); idx >= 0 {
				return strings.TrimSpace(rest[:idx])
			}
			return strings.TrimSpace(rest)
		}
	}
	return ""
}

func findEndLine(lines []string, startIdx int, kind string) int {
	endKW := "end " + kind
	for i := startIdx + 1; i < len(lines); i++ {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(lines[i])), endKW) {
			return i + 1
		}
	}
	return startIdx + 1
}

// enclosingSymbol returns the qualified name of the innermost symbol (function/sub/class) that contains the given line.
func enclosingSymbol(line int, symbols []parser.Symbol) string {
	var best *parser.Symbol
	for i := range symbols {
		s := &symbols[i]
		if s.StartLine <= line && line <= s.EndLine {
			if best == nil || (s.EndLine-s.StartLine) < (best.EndLine-best.StartLine) {
				best = s
			}
		}
	}
	if best == nil {
		return ""
	}
	return best.QualifiedName
}

func parseIncludes(content string) []parser.RawReference {
	var refs []parser.RawReference

	re := regexp.MustCompile(`<!--\s*#include\s+(file|virtual)\s*=\s*"([^"]+)"\s*-->`)
	for _, match := range re.FindAllStringSubmatch(content, -1) {
		if len(match) >= 3 {
			refs = append(refs, parser.RawReference{
				ToName:        match[2],
				ReferenceType: "imports",
			})
		}
	}

	return refs
}

func extractSQLRefs(sql string, line int) []parser.RawReference {
	var refs []parser.RawReference

	upper := strings.ToUpper(sql)

	// Extract table names from FROM/JOIN/INTO/UPDATE clauses
	tablePatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bFROM\s+(\[?\w+\]?\.?\[?\w+\]?)`),
		regexp.MustCompile(`(?i)\bJOIN\s+(\[?\w+\]?\.?\[?\w+\]?)`),
		regexp.MustCompile(`(?i)\bINTO\s+(\[?\w+\]?\.?\[?\w+\]?)`),
		regexp.MustCompile(`(?i)\bUPDATE\s+(\[?\w+\]?\.?\[?\w+\]?)`),
	}

	for _, pat := range tablePatterns {
		for _, m := range pat.FindAllStringSubmatch(sql, -1) {
			if len(m) >= 2 {
				tableName := strings.Trim(m[1], "[]")
				if !isSQLKeyword(tableName) {
					refType := "reads_from"
					if strings.Contains(upper, "INSERT") || strings.Contains(upper, "UPDATE") || strings.Contains(upper, "DELETE") {
						refType = "writes_to"
					}
					refs = append(refs, parser.RawReference{
						ToName:        tableName,
						ReferenceType: refType,
						Line:          line,
					})
				}
			}
		}
	}

	// Extract EXEC calls
	execPat := regexp.MustCompile(`(?i)\bEXEC(?:UTE)?\s+(\[?\w+\]?\.?\[?\w+\]?)`)
	for _, m := range execPat.FindAllStringSubmatch(sql, -1) {
		if len(m) >= 2 {
			refs = append(refs, parser.RawReference{
				ToName:        strings.Trim(m[1], "[]"),
				ReferenceType: "calls",
				Line:          line,
			})
		}
	}

	return refs
}

func isSQLKeyword(s string) bool {
	kw := map[string]bool{
		"SELECT": true, "FROM": true, "WHERE": true, "SET": true,
		"VALUES": true, "INTO": true, "TABLE": true, "AS": true,
	}
	return kw[strings.ToUpper(s)]
}

// extractDirectives parses ASP.NET <%@ ... %> directive blocks.
func extractDirectives(content string) []parser.RawReference {
	var refs []parser.RawReference

	re := regexp.MustCompile(`(?i)<%@\s*(Page|Control|Master|Register|Import)\s+([^%]+?)%>`)
	for _, match := range re.FindAllStringSubmatch(content, -1) {
		if len(match) < 3 {
			continue
		}
		directive := strings.ToLower(match[1])
		attrs := match[2]
		line := strings.Count(content[:strings.Index(content, match[0])], "\n") + 1

		switch directive {
		case "page", "control", "master":
			// CodeBehind="Foo.aspx.cs" or CodeFile="Foo.aspx.cs"
			if cb := extractAttrValue(attrs, "CodeBehind"); cb != "" {
				refs = append(refs, parser.RawReference{
					ToName:        cb,
					ReferenceType: "imports",
					Line:          line,
				})
			}
			if cf := extractAttrValue(attrs, "CodeFile"); cf != "" {
				refs = append(refs, parser.RawReference{
					ToName:        cf,
					ReferenceType: "imports",
					Line:          line,
				})
			}
			// Inherits="MyApp.UsersPage"
			if inh := extractAttrValue(attrs, "Inherits"); inh != "" {
				refs = append(refs, parser.RawReference{
					ToName:        inh,
					ReferenceType: "inherits",
					Line:          line,
				})
			}

		case "import":
			// Namespace="System.Data"
			if ns := extractAttrValue(attrs, "Namespace"); ns != "" {
				refs = append(refs, parser.RawReference{
					ToName:        ns,
					ReferenceType: "imports",
					Line:          line,
				})
			}

		case "register":
			// Assembly="..." Namespace="..."
			if ns := extractAttrValue(attrs, "Namespace"); ns != "" {
				refs = append(refs, parser.RawReference{
					ToName:        ns,
					ReferenceType: "imports",
					Line:          line,
				})
			}
			if src := extractAttrValue(attrs, "Src"); src != "" {
				refs = append(refs, parser.RawReference{
					ToName:        src,
					ReferenceType: "imports",
					Line:          line,
				})
			}
		}
	}

	return refs
}

func extractAttrValue(attrs, name string) string {
	re := regexp.MustCompile(`(?i)` + name + `\s*=\s*"([^"]*)"`)
	if m := re.FindStringSubmatch(attrs); len(m) >= 2 {
		return m[1]
	}
	return ""
}
