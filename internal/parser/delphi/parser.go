package delphi

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/maraichr/lattice/internal/parser"
)

// Parser implements a parser for Delphi/Object Pascal files.
type Parser struct{}

func New() *Parser {
	return &Parser{}
}

func (p *Parser) Languages() []string {
	return []string{"delphi", "pascal"}
}

func (p *Parser) Parse(input parser.FileInput) (*parser.ParseResult, error) {
	ext := strings.ToLower(filepath.Ext(input.Path))

	// DFM files get special handling
	if ext == ".dfm" {
		symbols, refs := ParseDFM(string(input.Content), 0)
		return &parser.ParseResult{
			Symbols:    symbols,
			References: refs,
		}, nil
	}

	// Pascal source files (.pas, .dpr)
	return parsePascal(input)
}

func parsePascal(input parser.FileInput) (*parser.ParseResult, error) {
	content := string(input.Content)
	lines := strings.Split(content, "\n")

	var symbols []parser.Symbol
	var refs []parser.RawReference

	unitName := ""
	inInterface := false
	inImplementation := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		lineNum := i + 1

		// Unit declaration
		if strings.HasPrefix(lower, "unit ") {
			name := extractIdentAfterKeyword(trimmed, "unit")
			if name != "" {
				unitName = name
				symbols = append(symbols, parser.Symbol{
					Name:          name,
					QualifiedName: name,
					Kind:          "module",
					Language:      "delphi",
					StartLine:     lineNum,
					EndLine:       len(lines),
				})
			}
		}

		// Program declaration
		if strings.HasPrefix(lower, "program ") {
			name := extractIdentAfterKeyword(trimmed, "program")
			if name != "" {
				unitName = name
				symbols = append(symbols, parser.Symbol{
					Name:          name,
					QualifiedName: name,
					Kind:          "module",
					Language:      "delphi",
					StartLine:     lineNum,
					EndLine:       len(lines),
				})
			}
		}

		// Track sections
		if lower == "interface" {
			inInterface = true
			inImplementation = false
		}
		if lower == "implementation" {
			inInterface = false
			inImplementation = true
		}

		// Uses clause
		if strings.HasPrefix(lower, "uses") {
			usesRefs := parseUsesClause(lines, i)
			refs = append(refs, usesRefs...)
		}

		// Type declarations
		// TMyClass = class(TParent)
		if classMatch := matchClassDecl(trimmed); classMatch != nil {
			qname := qualify(unitName, classMatch.name)
			sym := parser.Symbol{
				Name:          classMatch.name,
				QualifiedName: qname,
				Kind:          classMatch.kind,
				Language:      "delphi",
				StartLine:     lineNum,
				EndLine:       findPascalEnd(lines, i),
			}
			symbols = append(symbols, sym)

			if classMatch.parent != "" {
				refs = append(refs, parser.RawReference{
					FromSymbol:    qname,
					ToName:        classMatch.parent,
					ReferenceType: "inherits",
					Line:          lineNum,
				})
			}
		}

		// Procedure/function declarations
		if strings.HasPrefix(lower, "procedure ") || strings.HasPrefix(lower, "function ") ||
			strings.HasPrefix(lower, "class procedure ") || strings.HasPrefix(lower, "class function ") ||
			strings.HasPrefix(lower, "constructor ") || strings.HasPrefix(lower, "destructor ") {

			name, sig := parsePascalProcDecl(trimmed)
			if name != "" {
				kind := "procedure"
				if strings.Contains(lower, "function") {
					kind = "function"
				} else if strings.Contains(lower, "constructor") {
					kind = "method"
				} else if strings.Contains(lower, "destructor") {
					kind = "method"
				}

				qname := qualify(unitName, name)
				endLine := lineNum
				if inImplementation {
					endLine = findPascalProcEnd(lines, i)
				}
				symbols = append(symbols, parser.Symbol{
					Name:          name,
					QualifiedName: qname,
					Kind:          kind,
					Language:      "delphi",
					StartLine:     lineNum,
					EndLine:       endLine,
					Signature:     sig,
				})
			}
		}

		// Property declarations (in class body)
		if (inInterface || inImplementation) && strings.HasPrefix(lower, "property ") {
			name := extractIdentAfterKeyword(trimmed, "property")
			if name != "" {
				symbols = append(symbols, parser.Symbol{
					Name:          name,
					QualifiedName: qualify(unitName, name),
					Kind:          "property",
					Language:      "delphi",
					StartLine:     lineNum,
					EndLine:       lineNum,
				})
			}
		}

		// Include directives {$I filename.inc}
		if includeMatch := regexp.MustCompile(`\{\$[Ii]\s+(\S+)\}`).FindStringSubmatch(trimmed); len(includeMatch) >= 2 {
			refs = append(refs, parser.RawReference{
				ToName:        includeMatch[1],
				ReferenceType: "imports",
				Line:          lineNum,
			})
		}
	}

	_ = inInterface // used above

	return &parser.ParseResult{
		Symbols:    symbols,
		References: refs,
	}, nil
}

type classDecl struct {
	name   string
	kind   string // class, record, interface
	parent string
}

func matchClassDecl(line string) *classDecl {
	// TMyClass = class(TParent) or TMyRecord = record
	patterns := []struct {
		re   *regexp.Regexp
		kind string
	}{
		{regexp.MustCompile(`(?i)^\s*(\w+)\s*=\s*class\s*\(\s*(\w+)\s*\)`), "class"},
		{regexp.MustCompile(`(?i)^\s*(\w+)\s*=\s*class\b`), "class"},
		{regexp.MustCompile(`(?i)^\s*(\w+)\s*=\s*record\b`), "type"},
		{regexp.MustCompile(`(?i)^\s*(\w+)\s*=\s*interface\s*\(\s*(\w+)\s*\)`), "interface"},
		{regexp.MustCompile(`(?i)^\s*(\w+)\s*=\s*interface\b`), "interface"},
	}

	for _, p := range patterns {
		m := p.re.FindStringSubmatch(line)
		if len(m) >= 2 {
			decl := &classDecl{name: m[1], kind: p.kind}
			if len(m) >= 3 {
				decl.parent = m[2]
			}
			return decl
		}
	}
	return nil
}

func parsePascalProcDecl(line string) (string, string) {
	lower := strings.ToLower(line)
	for _, prefix := range []string{"class procedure ", "class function ", "procedure ", "function ", "constructor ", "destructor "} {
		if strings.HasPrefix(lower, prefix) {
			rest := line[len(prefix):]
			// Extract name (may be ClassName.MethodName)
			if idx := strings.IndexAny(rest, "(;"); idx >= 0 {
				name := strings.TrimSpace(rest[:idx])
				sig := ""
				if rest[idx] == '(' {
					endIdx := strings.Index(rest, ")")
					if endIdx > idx {
						sig = rest[idx : endIdx+1]
					}
				}
				return name, sig
			}
			return strings.TrimSpace(strings.TrimRight(rest, ";")), ""
		}
	}
	return "", ""
}

func parseUsesClause(lines []string, startIdx int) []parser.RawReference {
	var refs []parser.RawReference

	// Collect the full uses statement (may span multiple lines)
	var uses strings.Builder
	for i := startIdx; i < len(lines); i++ {
		uses.WriteString(lines[i])
		if strings.Contains(lines[i], ";") {
			break
		}
		uses.WriteString(" ")
	}

	text := uses.String()
	if idx := strings.IndexByte(text, ';'); idx >= 0 {
		text = text[:idx]
	}

	// Remove "uses" keyword
	lower := strings.ToLower(text)
	if idx := strings.Index(lower, "uses"); idx >= 0 {
		text = text[idx+4:]
	}

	// Split by comma and extract unit names (ignore "in 'path'" parts)
	for _, part := range strings.Split(text, ",") {
		part = strings.TrimSpace(part)
		if inIdx := strings.Index(strings.ToLower(part), " in "); inIdx >= 0 {
			part = part[:inIdx]
		}
		part = strings.TrimSpace(part)
		if part != "" {
			refs = append(refs, parser.RawReference{
				ToName:        part,
				ReferenceType: "imports",
				Line:          startIdx + 1,
			})
		}
	}

	return refs
}

func extractIdentAfterKeyword(line, keyword string) string {
	lower := strings.ToLower(line)
	idx := strings.Index(lower, strings.ToLower(keyword))
	if idx < 0 {
		return ""
	}
	rest := strings.TrimSpace(line[idx+len(keyword):])
	rest = strings.TrimRight(rest, ";")
	rest = strings.TrimSpace(rest)
	if spaceIdx := strings.IndexAny(rest, " \t(;"); spaceIdx >= 0 {
		return rest[:spaceIdx]
	}
	return rest
}

func qualify(unitName, name string) string {
	if unitName != "" {
		return unitName + "." + name
	}
	return name
}

func findPascalEnd(lines []string, startIdx int) int {
	depth := 0
	for i := startIdx; i < len(lines); i++ {
		lower := strings.ToLower(strings.TrimSpace(lines[i]))
		if lower == "end;" || lower == "end." {
			if depth <= 0 {
				return i + 1
			}
			depth--
		}
		// Count nested begin/record/class blocks
		if strings.HasPrefix(lower, "begin") || strings.HasSuffix(lower, "= record") ||
			strings.Contains(lower, "= class") {
			depth++
		}
	}
	return startIdx + 1
}

func findPascalProcEnd(lines []string, startIdx int) int {
	depth := 0
	foundBegin := false
	for i := startIdx + 1; i < len(lines); i++ {
		lower := strings.ToLower(strings.TrimSpace(lines[i]))
		if strings.HasPrefix(lower, "begin") {
			foundBegin = true
			depth++
		}
		if (lower == "end;" || lower == "end.") && foundBegin {
			depth--
			if depth <= 0 {
				return i + 1
			}
		}
	}
	return startIdx + 1
}
