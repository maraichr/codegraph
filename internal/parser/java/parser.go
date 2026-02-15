package java

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"

	"github.com/maraichr/lattice/internal/parser"
)

// Parser implements a tree-sitter based Java parser.
type Parser struct {
	tsParser *sitter.Parser
}

func New() *Parser {
	p := sitter.NewParser()
	p.SetLanguage(java.GetLanguage())
	return &Parser{tsParser: p}
}

func (p *Parser) Languages() []string {
	return []string{"java"}
}

func (p *Parser) Parse(input parser.FileInput) (*parser.ParseResult, error) {
	tree, err := p.tsParser.ParseCtx(context.Background(), nil, input.Content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	root := tree.RootNode()

	var symbols []parser.Symbol
	var refs []parser.RawReference

	packageName := ""

	// Walk tree to extract symbols
	for i := 0; i < int(root.ChildCount()); i++ {
		child := root.Child(i)
		switch child.Type() {
		case "package_declaration":
			packageName = extractPackageName(child, input.Content)

		case "import_declaration":
			importPath := extractImportPath(child, input.Content)
			if importPath != "" {
				refs = append(refs, parser.RawReference{
					ToName:        importPath,
					ToQualified:   importPath,
					ReferenceType: "imports",
					Line:          int(child.StartPoint().Row) + 1,
				})
			}

		case "class_declaration":
			syms, rfs := extractClass(child, input.Content, packageName)
			symbols = append(symbols, syms...)
			refs = append(refs, rfs...)

		case "interface_declaration":
			syms, rfs := extractInterface(child, input.Content, packageName)
			symbols = append(symbols, syms...)
			refs = append(refs, rfs...)

		case "enum_declaration":
			syms := extractEnum(child, input.Content, packageName)
			symbols = append(symbols, syms...)
		}
	}

	// Process annotations for Spring/JPA detection
	annoRefs := extractAnnotationRefs(root, input.Content, packageName)
	refs = append(refs, annoRefs...)

	return &parser.ParseResult{
		Symbols:    symbols,
		References: refs,
	}, nil
}

func extractPackageName(node *sitter.Node, src []byte) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "scoped_identifier" || child.Type() == "identifier" {
			return child.Content(src)
		}
	}
	return ""
}

func extractImportPath(node *sitter.Node, src []byte) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "scoped_identifier" || child.Type() == "identifier" {
			return child.Content(src)
		}
	}
	return ""
}

func extractClass(node *sitter.Node, src []byte, pkg string) ([]parser.Symbol, []parser.RawReference) {
	var symbols []parser.Symbol
	var refs []parser.RawReference

	name := ""
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "identifier" {
			name = child.Content(src)
			break
		}
	}

	if name == "" {
		return nil, nil
	}

	qname := qualifyJava(pkg, name)
	classSym := parser.Symbol{
		Name:          name,
		QualifiedName: qname,
		Kind:          "class",
		Language:      "java",
		StartLine:     int(node.StartPoint().Row) + 1,
		EndLine:       int(node.EndPoint().Row) + 1,
	}

	// Check for superclass/interfaces
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "superclass" {
			parent := extractTypeIdent(child, src)
			if parent != "" {
				refs = append(refs, parser.RawReference{
					FromSymbol:    qname,
					ToName:        parent,
					ReferenceType: "inherits",
					Line:          int(child.StartPoint().Row) + 1,
				})
			}
		}
		if child.Type() == "super_interfaces" {
			ifaces := extractTypeList(child, src)
			for _, iface := range ifaces {
				refs = append(refs, parser.RawReference{
					FromSymbol:    qname,
					ToName:        iface,
					ReferenceType: "implements",
					Line:          int(child.StartPoint().Row) + 1,
				})
			}
		}
	}

	symbols = append(symbols, classSym)

	// Extract members from class body
	body := findChild(node, "class_body")
	if body != nil {
		memberSyms, memberRefs := extractMembers(body, src, pkg, name)
		symbols = append(symbols, memberSyms...)
		refs = append(refs, memberRefs...)
	}

	return symbols, refs
}

func extractInterface(node *sitter.Node, src []byte, pkg string) ([]parser.Symbol, []parser.RawReference) {
	var symbols []parser.Symbol
	var refs []parser.RawReference

	name := ""
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "identifier" {
			name = child.Content(src)
			break
		}
	}

	if name == "" {
		return nil, nil
	}

	qname := qualifyJava(pkg, name)
	symbols = append(symbols, parser.Symbol{
		Name:          name,
		QualifiedName: qname,
		Kind:          "interface",
		Language:      "java",
		StartLine:     int(node.StartPoint().Row) + 1,
		EndLine:       int(node.EndPoint().Row) + 1,
	})

	return symbols, refs
}

func extractEnum(node *sitter.Node, src []byte, pkg string) []parser.Symbol {
	name := ""
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "identifier" {
			name = child.Content(src)
			break
		}
	}
	if name == "" {
		return nil
	}

	return []parser.Symbol{{
		Name:          name,
		QualifiedName: qualifyJava(pkg, name),
		Kind:          "enum",
		Language:      "java",
		StartLine:     int(node.StartPoint().Row) + 1,
		EndLine:       int(node.EndPoint().Row) + 1,
	}}
}

func extractMembers(body *sitter.Node, src []byte, pkg, className string) ([]parser.Symbol, []parser.RawReference) {
	var symbols []parser.Symbol
	var refs []parser.RawReference

	for i := 0; i < int(body.ChildCount()); i++ {
		child := body.Child(i)
		switch child.Type() {
		case "method_declaration":
			name, sig := extractMethodDecl(child, src)
			if name != "" {
				qname := qualifyJava(pkg, className+"."+name)
				symbols = append(symbols, parser.Symbol{
					Name:          name,
					QualifiedName: qname,
					Kind:          "method",
					Language:      "java",
					StartLine:     int(child.StartPoint().Row) + 1,
					EndLine:       int(child.EndPoint().Row) + 1,
					Signature:     sig,
				})
			}

		case "constructor_declaration":
			name := className
			qname := qualifyJava(pkg, className+"."+name)
			symbols = append(symbols, parser.Symbol{
				Name:          name,
				QualifiedName: qname,
				Kind:          "method",
				Language:      "java",
				StartLine:     int(child.StartPoint().Row) + 1,
				EndLine:       int(child.EndPoint().Row) + 1,
			})

		case "field_declaration":
			fieldName := extractFieldName(child, src)
			if fieldName != "" {
				symbols = append(symbols, parser.Symbol{
					Name:          fieldName,
					QualifiedName: qualifyJava(pkg, className+"."+fieldName),
					Kind:          "field",
					Language:      "java",
					StartLine:     int(child.StartPoint().Row) + 1,
					EndLine:       int(child.EndPoint().Row) + 1,
				})
			}
		}
	}

	return symbols, refs
}

func extractMethodDecl(node *sitter.Node, src []byte) (string, string) {
	name := ""
	sig := ""
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "identifier" && name == "" {
			name = child.Content(src)
		}
		if child.Type() == "formal_parameters" {
			sig = child.Content(src)
		}
	}
	return name, sig
}

func extractFieldName(node *sitter.Node, src []byte) string {
	decl := findChild(node, "variable_declarator")
	if decl != nil {
		for i := 0; i < int(decl.ChildCount()); i++ {
			child := decl.Child(i)
			if child.Type() == "identifier" {
				return child.Content(src)
			}
		}
	}
	return ""
}

func extractTypeIdent(node *sitter.Node, src []byte) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "type_identifier" || child.Type() == "identifier" {
			return child.Content(src)
		}
	}
	return ""
}

func extractTypeList(node *sitter.Node, src []byte) []string {
	var types []string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "type_list" {
			for j := 0; j < int(child.ChildCount()); j++ {
				grandchild := child.Child(j)
				if grandchild.Type() == "type_identifier" || grandchild.Type() == "generic_type" {
					types = append(types, grandchild.Content(src))
				}
			}
		}
	}
	return types
}

func findChild(node *sitter.Node, nodeType string) *sitter.Node {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == nodeType {
			return child
		}
	}
	return nil
}

func qualifyJava(pkg, name string) string {
	if pkg != "" {
		return pkg + "." + name
	}
	return name
}

func qualifyAnnotated(pkg, className, refName string) string {
	if className != "" {
		return qualifyJava(pkg, className)
	}
	return refName
}

// extractAnnotationRefs walks the tree looking for Spring/JPA annotations.
func extractAnnotationRefs(root *sitter.Node, src []byte, pkg string) []parser.RawReference {
	var refs []parser.RawReference
	walkTree(root, func(node *sitter.Node) {
		if node.Type() != "marker_annotation" && node.Type() != "annotation" {
			return
		}

		annoText := node.Content(src)
		line := int(node.StartPoint().Row) + 1

		// Find the annotated class/method name
		parent := node.Parent()
		className := ""
		if parent != nil {
			for i := 0; i < int(parent.ChildCount()); i++ {
				child := parent.Child(i)
				if child.Type() == "identifier" {
					className = child.Content(src)
					break
				}
			}
		}

		// @Entity or @Table(name="...")
		if strings.Contains(annoText, "@Entity") || strings.Contains(annoText, "@Table") {
			tableName := extractAnnotationParam(annoText, "name")
			if tableName == "" {
				tableName = className
			}
			if tableName != "" {
				refs = append(refs, parser.RawReference{
					FromSymbol:    qualifyAnnotated(pkg, className, ""),
					ToName:        tableName,
					ReferenceType: "uses_table",
					Line:          line,
				})
			}
		}

		// @Query("SELECT ...")
		if strings.Contains(annoText, "@Query") {
			query := extractAnnotationStringParam(annoText)
			if query != "" && looksLikeSQL(query) {
				tableRefs := extractSQLTableRefs(query, line)
				refs = append(refs, tableRefs...)
			}
		}

		// @RequestMapping, @GetMapping, etc.
		if strings.Contains(annoText, "Mapping") {
			path := extractAnnotationStringParam(annoText)
			if path != "" {
				refs = append(refs, parser.RawReference{
					FromSymbol:    qualifyAnnotated(pkg, className, ""),
					ToName:        path,
					ReferenceType: "references",
					Line:          line,
				})
			}
		}
	})

	return refs
}

func walkTree(node *sitter.Node, fn func(*sitter.Node)) {
	fn(node)
	for i := 0; i < int(node.ChildCount()); i++ {
		walkTree(node.Child(i), fn)
	}
}

func extractAnnotationParam(text, param string) string {
	// Look for param = "value" or param = 'value'
	_, rest, found := strings.Cut(text, param)
	if !found {
		return ""
	}
	rest = strings.TrimSpace(rest)
	if len(rest) > 0 && rest[0] == '=' {
		rest = strings.TrimSpace(rest[1:])
		if len(rest) > 0 && (rest[0] == '"' || rest[0] == '\'') {
			end := strings.IndexByte(rest[1:], rest[0])
			if end >= 0 {
				return rest[1 : end+1]
			}
		}
	}
	return ""
}

func extractAnnotationStringParam(text string) string {
	// Extract first string literal from annotation
	idx := strings.IndexByte(text, '"')
	if idx < 0 {
		return ""
	}
	end := strings.IndexByte(text[idx+1:], '"')
	if end < 0 {
		return ""
	}
	return text[idx+1 : idx+1+end]
}

func looksLikeSQL(s string) bool {
	upper := strings.ToUpper(strings.TrimSpace(s))
	for _, kw := range []string{"SELECT", "INSERT", "UPDATE", "DELETE", "FROM"} {
		if strings.Contains(upper, kw) {
			return true
		}
	}
	return false
}

func extractSQLTableRefs(sql string, line int) []parser.RawReference {
	var refs []parser.RawReference
	upper := strings.ToUpper(sql)
	keywords := []string{"FROM", "JOIN", "INTO", "UPDATE"}

	for _, kw := range keywords {
		idx := 0
		for {
			pos := strings.Index(upper[idx:], kw+" ")
			if pos < 0 {
				break
			}
			pos += idx + len(kw) + 1
			rest := strings.TrimSpace(sql[pos:])
			// Extract table name (first word)
			end := strings.IndexAny(rest, " \t\n,;)")
			tableName := rest
			if end > 0 {
				tableName = rest[:end]
			}
			tableName = strings.TrimSpace(tableName)
			if tableName != "" && !isSQLKeyword(tableName) {
				refs = append(refs, parser.RawReference{
					ToName:        tableName,
					ReferenceType: "uses_table",
					Line:          line,
				})
			}
			idx = pos
		}
	}

	return refs
}

func isSQLKeyword(s string) bool {
	kw := map[string]bool{
		"SELECT": true, "FROM": true, "WHERE": true, "AND": true,
		"OR": true, "SET": true, "VALUES": true, "AS": true,
		"ON": true, "IN": true, "NOT": true, "NULL": true,
	}
	return kw[strings.ToUpper(s)]
}
