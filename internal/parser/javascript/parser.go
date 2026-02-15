package javascript

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/typescript/typescript"

	"github.com/maraichr/lattice/internal/parser"
)

// Parser implements a tree-sitter based JavaScript/TypeScript parser.
type Parser struct {
	tsParser *sitter.Parser
	lang     string // "javascript" or "typescript"
}

func NewJS() *Parser {
	p := sitter.NewParser()
	p.SetLanguage(javascript.GetLanguage())
	return &Parser{tsParser: p, lang: "javascript"}
}

func NewTS() *Parser {
	p := sitter.NewParser()
	p.SetLanguage(typescript.GetLanguage())
	return &Parser{tsParser: p, lang: "typescript"}
}

func (p *Parser) Languages() []string {
	return []string{p.lang}
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

	for i := 0; i < int(root.ChildCount()); i++ {
		child := root.Child(i)
		syms, rfs := p.extractTopLevel(child, input.Content, "")
		symbols = append(symbols, syms...)
		refs = append(refs, rfs...)
	}

	return &parser.ParseResult{
		Symbols:    symbols,
		References: refs,
	}, nil
}

func (p *Parser) extractTopLevel(node *sitter.Node, src []byte, scope string) ([]parser.Symbol, []parser.RawReference) {
	switch node.Type() {
	case "function_declaration":
		sym, rfs := p.extractFunctionDecl(node, src, scope)
		return []parser.Symbol{sym}, rfs

	case "class_declaration":
		return p.extractClassDecl(node, src, scope)

	case "lexical_declaration", "variable_declaration":
		return p.extractVarDecl(node, src, scope)

	case "export_statement":
		return p.extractExportStatement(node, src, scope)

	case "import_statement":
		ref := p.extractImportStatement(node, src)
		return nil, ref

	case "interface_declaration":
		sym, rfs := p.extractInterfaceDecl(node, src, scope)
		return []parser.Symbol{sym}, rfs

	case "type_alias_declaration":
		sym := p.extractTypeAlias(node, src, scope)
		return []parser.Symbol{sym}, nil

	case "enum_declaration":
		sym := p.extractEnumDecl(node, src, scope)
		return []parser.Symbol{sym}, nil

	case "expression_statement":
		// Check for require() calls: module.exports = require(...)
		rfs := p.extractRequireFromExpression(node, src)
		return nil, rfs
	}

	return nil, nil
}

// --- Function declarations ---

func (p *Parser) extractFunctionDecl(node *sitter.Node, src []byte, scope string) (parser.Symbol, []parser.RawReference) {
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

	qname := qualify(scope, name)
	return parser.Symbol{
		Name:          name,
		QualifiedName: qname,
		Kind:          "function",
		Language:      p.lang,
		StartLine:     int(node.StartPoint().Row) + 1,
		EndLine:       int(node.EndPoint().Row) + 1,
		Signature:     sig,
	}, nil
}

// --- Class declarations ---

func (p *Parser) extractClassDecl(node *sitter.Node, src []byte, scope string) ([]parser.Symbol, []parser.RawReference) {
	var symbols []parser.Symbol
	var refs []parser.RawReference

	name := ""
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if (child.Type() == "identifier" || child.Type() == "type_identifier") && name == "" {
			name = child.Content(src)
			break
		}
	}
	if name == "" {
		return nil, nil
	}

	qname := qualify(scope, name)
	symbols = append(symbols, parser.Symbol{
		Name:          name,
		QualifiedName: qname,
		Kind:          "class",
		Language:      p.lang,
		StartLine:     int(node.StartPoint().Row) + 1,
		EndLine:       int(node.EndPoint().Row) + 1,
	})

	// Heritage clauses: extends / implements
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "class_heritage" {
			rfs := p.extractHeritage(child, src, qname)
			refs = append(refs, rfs...)
		}
	}

	// Class body members
	body := findChild(node, "class_body")
	if body != nil {
		memberSyms, memberRefs := p.extractClassMembers(body, src, name)
		symbols = append(symbols, memberSyms...)
		refs = append(refs, memberRefs...)
	}

	return symbols, refs
}

func (p *Parser) extractHeritage(node *sitter.Node, src []byte, fromQName string) []parser.RawReference {
	var refs []parser.RawReference
	line := int(node.StartPoint().Row) + 1

	// Check direct children of class_heritage.
	// JS: class_heritage → extends + identifier (no extends_clause wrapper)
	// TS: class_heritage → extends_clause + implements_clause
	hasExtClause := false
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "extends_clause" || child.Type() == "implements_clause" {
			hasExtClause = true
			break
		}
	}

	if !hasExtClause {
		// JS pattern: class_heritage direct children are "extends" keyword + identifier
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "identifier" || child.Type() == "member_expression" {
				refs = append(refs, parser.RawReference{
					FromSymbol:    fromQName,
					ToName:        child.Content(src),
					ReferenceType: "inherits",
					Line:          line,
				})
			}
		}
		return refs
	}

	// TS pattern: walk for extends_clause / implements_clause
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "extends_clause":
			for j := 0; j < int(child.ChildCount()); j++ {
				gc := child.Child(j)
				if gc.Type() == "identifier" || gc.Type() == "member_expression" {
					refs = append(refs, parser.RawReference{
						FromSymbol:    fromQName,
						ToName:        gc.Content(src),
						ReferenceType: "inherits",
						Line:          line,
					})
				}
			}
		case "implements_clause":
			for j := 0; j < int(child.ChildCount()); j++ {
				gc := child.Child(j)
				switch gc.Type() {
				case "type_identifier", "identifier", "generic_type":
					typeName := gc.Content(src)
					if gc.Type() == "generic_type" {
						for k := 0; k < int(gc.ChildCount()); k++ {
							ggc := gc.Child(k)
							if ggc.Type() == "type_identifier" || ggc.Type() == "identifier" {
								typeName = ggc.Content(src)
								break
							}
						}
					}
					refs = append(refs, parser.RawReference{
						FromSymbol:    fromQName,
						ToName:        typeName,
						ReferenceType: "implements",
						Line:          line,
					})
				}
			}
		}
	}

	return refs
}

func (p *Parser) extractClassMembers(body *sitter.Node, src []byte, className string) ([]parser.Symbol, []parser.RawReference) {
	var symbols []parser.Symbol
	var refs []parser.RawReference

	for i := 0; i < int(body.ChildCount()); i++ {
		child := body.Child(i)
		switch child.Type() {
		case "method_definition":
			sym, rfs := p.extractMethodDef(child, src, className)
			if sym.Name != "" {
				symbols = append(symbols, sym)
			}
			refs = append(refs, rfs...)

		case "public_field_definition", "field_definition":
			name := p.extractPropertyName(child, src)
			if name != "" {
				symbols = append(symbols, parser.Symbol{
					Name:          name,
					QualifiedName: className + "." + name,
					Kind:          "property",
					Language:      p.lang,
					StartLine:     int(child.StartPoint().Row) + 1,
					EndLine:       int(child.EndPoint().Row) + 1,
				})
			}
		}
	}

	return symbols, refs
}

func (p *Parser) extractMethodDef(node *sitter.Node, src []byte, className string) (parser.Symbol, []parser.RawReference) {
	name := ""
	sig := ""
	kind := "method"
	var refs []parser.RawReference

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "property_identifier":
			name = child.Content(src)
		case "formal_parameters":
			sig = child.Content(src)
		case "get", "set":
			kind = "property"
		}
	}

	// Check for constructor
	if name == "constructor" {
		kind = "method"
	}

	// Check for decorators
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "decorator" {
			decoratorName := extractDecoratorName(child, src)
			if decoratorName != "" {
				refs = append(refs, parser.RawReference{
					FromSymbol:    className + "." + name,
					ToName:        decoratorName,
					ReferenceType: "references",
					Line:          int(child.StartPoint().Row) + 1,
				})
			}
		}
	}

	if name == "" {
		return parser.Symbol{}, refs
	}

	qname := className + "." + name
	return parser.Symbol{
		Name:          name,
		QualifiedName: qname,
		Kind:          kind,
		Language:      p.lang,
		StartLine:     int(node.StartPoint().Row) + 1,
		EndLine:       int(node.EndPoint().Row) + 1,
		Signature:     sig,
	}, refs
}

func (p *Parser) extractPropertyName(node *sitter.Node, src []byte) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "property_identifier" || child.Type() == "identifier" {
			return child.Content(src)
		}
	}
	return ""
}

// --- Variable/const declarations (arrow functions, exported vars) ---

func (p *Parser) extractVarDecl(node *sitter.Node, src []byte, scope string) ([]parser.Symbol, []parser.RawReference) {
	var symbols []parser.Symbol
	var refs []parser.RawReference

	walkChildren(node, func(child *sitter.Node) {
		if child.Type() != "variable_declarator" {
			return
		}

		name := ""
		isArrow := false
		for j := 0; j < int(child.ChildCount()); j++ {
			gc := child.Child(j)
			if gc.Type() == "identifier" && name == "" {
				name = gc.Content(src)
			}
			if gc.Type() == "arrow_function" || gc.Type() == "function" || gc.Type() == "function_expression" {
				isArrow = true
			}
		}

		if name == "" {
			return
		}

		// Check for require() calls
		reqRef := p.extractRequireFromDeclarator(child, src)
		if reqRef != nil {
			refs = append(refs, *reqRef)
		}

		if isArrow {
			sig := ""
			walkTree(child, func(n *sitter.Node) {
				if n.Type() == "formal_parameters" && sig == "" {
					sig = n.Content(src)
				}
			})
			symbols = append(symbols, parser.Symbol{
				Name:          name,
				QualifiedName: qualify(scope, name),
				Kind:          "function",
				Language:      p.lang,
				StartLine:     int(node.StartPoint().Row) + 1,
				EndLine:       int(node.EndPoint().Row) + 1,
				Signature:     sig,
			})
		}
	})

	return symbols, refs
}

// --- Export statements ---

func (p *Parser) extractExportStatement(node *sitter.Node, src []byte, scope string) ([]parser.Symbol, []parser.RawReference) {
	var symbols []parser.Symbol
	var refs []parser.RawReference

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "function_declaration":
			sym, rfs := p.extractFunctionDecl(child, src, scope)
			symbols = append(symbols, sym)
			refs = append(refs, rfs...)

		case "class_declaration":
			syms, rfs := p.extractClassDecl(child, src, scope)
			symbols = append(symbols, syms...)
			refs = append(refs, rfs...)

		case "lexical_declaration", "variable_declaration":
			syms, rfs := p.extractVarDecl(child, src, scope)
			symbols = append(symbols, syms...)
			refs = append(refs, rfs...)

		case "interface_declaration":
			sym, rfs := p.extractInterfaceDecl(child, src, scope)
			symbols = append(symbols, sym)
			refs = append(refs, rfs...)

		case "type_alias_declaration":
			sym := p.extractTypeAlias(child, src, scope)
			symbols = append(symbols, sym)

		case "enum_declaration":
			sym := p.extractEnumDecl(child, src, scope)
			symbols = append(symbols, sym)

		case "string", "string_fragment":
			// export { foo } from './bar'  — the source string
			source := extractStringContent(child, src)
			if source != "" {
				refs = append(refs, parser.RawReference{
					ToName:        source,
					ReferenceType: "imports",
					Line:          int(child.StartPoint().Row) + 1,
				})
			}
		}
	}

	// Check for re-export: export { x } from 'module'
	// The "from" source is a string node that may be a direct child
	source := findChild(node, "string")
	if source != nil {
		s := extractStringContent(source, src)
		if s != "" {
			// Avoid duplicate if we already added it above
			found := false
			for _, r := range refs {
				if r.ToName == s && r.ReferenceType == "imports" {
					found = true
					break
				}
			}
			if !found {
				refs = append(refs, parser.RawReference{
					ToName:        s,
					ReferenceType: "imports",
					Line:          int(source.StartPoint().Row) + 1,
				})
			}
		}
	}

	// Handle default export of identifier (export default App)
	// handled as expression export — no symbol created, that's normal

	return symbols, refs
}

// --- Import statements ---

func (p *Parser) extractImportStatement(node *sitter.Node, src []byte) []parser.RawReference {
	var refs []parser.RawReference

	// Find the source string: import ... from 'source'
	source := findChild(node, "string")
	if source != nil {
		s := extractStringContent(source, src)
		if s != "" {
			refs = append(refs, parser.RawReference{
				ToName:        s,
				ReferenceType: "imports",
				Line:          int(node.StartPoint().Row) + 1,
			})
		}
	}

	return refs
}

// --- Require calls ---

func (p *Parser) extractRequireFromDeclarator(node *sitter.Node, src []byte) *parser.RawReference {
	var ref *parser.RawReference
	walkTree(node, func(n *sitter.Node) {
		if ref != nil {
			return
		}
		if n.Type() == "call_expression" {
			fn := findChild(n, "identifier")
			if fn != nil && fn.Content(src) == "require" {
				args := findChild(n, "arguments")
				if args != nil {
					str := findChild(args, "string")
					if str != nil {
						s := extractStringContent(str, src)
						if s != "" {
							ref = &parser.RawReference{
								ToName:        s,
								ReferenceType: "imports",
								Line:          int(n.StartPoint().Row) + 1,
							}
						}
					}
				}
			}
		}
	})
	return ref
}

func (p *Parser) extractRequireFromExpression(node *sitter.Node, src []byte) []parser.RawReference {
	var refs []parser.RawReference
	walkTree(node, func(n *sitter.Node) {
		if n.Type() == "call_expression" {
			fn := findChild(n, "identifier")
			if fn != nil && fn.Content(src) == "require" {
				args := findChild(n, "arguments")
				if args != nil {
					str := findChild(args, "string")
					if str != nil {
						s := extractStringContent(str, src)
						if s != "" {
							refs = append(refs, parser.RawReference{
								ToName:        s,
								ReferenceType: "imports",
								Line:          int(n.StartPoint().Row) + 1,
							})
						}
					}
				}
			}
		}
	})
	return refs
}

// --- TypeScript: Interface ---

func (p *Parser) extractInterfaceDecl(node *sitter.Node, src []byte, scope string) (parser.Symbol, []parser.RawReference) {
	name := ""
	var refs []parser.RawReference

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "type_identifier", "identifier":
			if name == "" {
				name = child.Content(src)
			}
		case "extends_type_clause":
			// interface Foo extends Bar, Baz
			for j := 0; j < int(child.ChildCount()); j++ {
				gc := child.Child(j)
				if gc.Type() == "type_identifier" || gc.Type() == "identifier" || gc.Type() == "generic_type" {
					refs = append(refs, parser.RawReference{
						FromSymbol:    qualify(scope, name),
						ToName:        gc.Content(src),
						ReferenceType: "inherits",
						Line:          int(gc.StartPoint().Row) + 1,
					})
				}
			}
		}
	}

	qname := qualify(scope, name)
	return parser.Symbol{
		Name:          name,
		QualifiedName: qname,
		Kind:          "interface",
		Language:      p.lang,
		StartLine:     int(node.StartPoint().Row) + 1,
		EndLine:       int(node.EndPoint().Row) + 1,
	}, refs
}

// --- TypeScript: Type alias ---

func (p *Parser) extractTypeAlias(node *sitter.Node, src []byte, scope string) parser.Symbol {
	name := ""
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if (child.Type() == "type_identifier" || child.Type() == "identifier") && name == "" {
			name = child.Content(src)
		}
	}

	return parser.Symbol{
		Name:          name,
		QualifiedName: qualify(scope, name),
		Kind:          "type",
		Language:      p.lang,
		StartLine:     int(node.StartPoint().Row) + 1,
		EndLine:       int(node.EndPoint().Row) + 1,
	}
}

// --- TypeScript: Enum ---

func (p *Parser) extractEnumDecl(node *sitter.Node, src []byte, scope string) parser.Symbol {
	name := ""
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "identifier" && name == "" {
			name = child.Content(src)
		}
	}

	return parser.Symbol{
		Name:          name,
		QualifiedName: qualify(scope, name),
		Kind:          "enum",
		Language:      p.lang,
		StartLine:     int(node.StartPoint().Row) + 1,
		EndLine:       int(node.EndPoint().Row) + 1,
	}
}

// --- Decorators (TS) ---

func extractDecoratorName(node *sitter.Node, src []byte) string {
	// @Decorator or @Decorator()
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "identifier":
			return child.Content(src)
		case "call_expression":
			fn := findChild(child, "identifier")
			if fn != nil {
				return fn.Content(src)
			}
		}
	}
	return ""
}

// --- Helpers ---

func qualify(scope, name string) string {
	if scope != "" {
		return scope + "." + name
	}
	return name
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

func walkTree(node *sitter.Node, fn func(*sitter.Node)) {
	fn(node)
	for i := 0; i < int(node.ChildCount()); i++ {
		walkTree(node.Child(i), fn)
	}
}

func walkChildren(node *sitter.Node, fn func(*sitter.Node)) {
	for i := 0; i < int(node.ChildCount()); i++ {
		fn(node.Child(i))
	}
}

func extractStringContent(node *sitter.Node, src []byte) string {
	text := node.Content(src)
	// Strip quotes: "foo" or 'foo' or `foo`
	if len(text) >= 2 {
		return strings.Trim(text, `"'`+"`")
	}
	return ""
}
