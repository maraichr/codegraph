package csharp

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/csharp"

	"github.com/maraichr/lattice/internal/parser"
)

// Parser implements a tree-sitter based C# parser.
type Parser struct {
	tsParser *sitter.Parser
}

func New() *Parser {
	p := sitter.NewParser()
	p.SetLanguage(csharp.GetLanguage())
	return &Parser{tsParser: p}
}

func (p *Parser) Languages() []string {
	return []string{"csharp"}
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

	// First pass: extract namespace and using directives from root children
	namespace := ""
	for i := 0; i < int(root.ChildCount()); i++ {
		child := root.Child(i)
		switch child.Type() {
		case "using_directive":
			importPath := extractUsingDirective(child, input.Content)
			if importPath != "" {
				refs = append(refs, parser.RawReference{
					ToName:        importPath,
					ToQualified:   importPath,
					ReferenceType: "imports",
					Line:          int(child.StartPoint().Row) + 1,
				})
			}

		case "namespace_declaration":
			ns := extractNamespaceName(child, input.Content)
			if ns != "" {
				namespace = ns
			}
			body := findChild(child, "declaration_list")
			if body != nil {
				processDeclarationList(body, input.Content, ns, &symbols, &refs)
			}

		case "file_scoped_namespace_declaration":
			ns := extractNamespaceName(child, input.Content)
			if ns != "" {
				namespace = ns
			}
			// File-scoped: type declarations are root-level siblings, processed below

		default:
			// Root-level type declarations (with or without file-scoped namespace)
			processTopLevelDecl(child, input.Content, namespace, &symbols, &refs)
		}
	}

	// Build class ranges for enclosing-scope resolution (FromSymbol for SQL refs)
	classRanges := buildClassRanges(root, input.Content, namespace)

	// Extract attribute-based and inline SQL references (with FromSymbol set)
	attrRefs := extractAttributeRefs(root, input.Content, namespace, classRanges)
	refs = append(refs, attrRefs...)

	sqlRefs := extractInlineSQLRefs(root, input.Content, namespace, classRanges)
	refs = append(refs, sqlRefs...)

	procRefs := extractStoredProcRefs(root, input.Content, classRanges)
	refs = append(refs, procRefs...)

	return &parser.ParseResult{
		Symbols:    symbols,
		References: refs,
	}, nil
}

func processDeclarationList(body *sitter.Node, src []byte, ns string, symbols *[]parser.Symbol, refs *[]parser.RawReference) {
	for i := 0; i < int(body.ChildCount()); i++ {
		child := body.Child(i)
		processTopLevelDecl(child, src, ns, symbols, refs)
	}
}

func processTopLevelDecl(node *sitter.Node, src []byte, ns string, symbols *[]parser.Symbol, refs *[]parser.RawReference) {
	switch node.Type() {
	case "class_declaration":
		syms, rfs := extractClass(node, src, ns)
		*symbols = append(*symbols, syms...)
		*refs = append(*refs, rfs...)

	case "interface_declaration":
		syms, rfs := extractInterface(node, src, ns)
		*symbols = append(*symbols, syms...)
		*refs = append(*refs, rfs...)

	case "struct_declaration":
		syms, rfs := extractStruct(node, src, ns)
		*symbols = append(*symbols, syms...)
		*refs = append(*refs, rfs...)

	case "enum_declaration":
		syms := extractEnum(node, src, ns)
		*symbols = append(*symbols, syms...)
	}
}

func extractNamespaceName(node *sitter.Node, src []byte) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "qualified_name", "identifier":
			return child.Content(src)
		}
	}
	return ""
}

func extractUsingDirective(node *sitter.Node, src []byte) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "qualified_name", "identifier":
			return child.Content(src)
		}
	}
	return ""
}

func extractClass(node *sitter.Node, src []byte, ns string) ([]parser.Symbol, []parser.RawReference) {
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

	qname := qualifyCSharp(ns, name)
	symbols = append(symbols, parser.Symbol{
		Name:          name,
		QualifiedName: qname,
		Kind:          "class",
		Language:      "csharp",
		StartLine:     int(node.StartPoint().Row) + 1,
		EndLine:       int(node.EndPoint().Row) + 1,
	})

	// Check base_list for inheritance/implementation
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "base_list" {
			baseRefs := extractBaseList(child, src, qname)
			refs = append(refs, baseRefs...)
		}
	}

	// Extract members from class body
	body := findChild(node, "declaration_list")
	if body != nil {
		memberSyms, memberRefs := extractMembers(body, src, ns, name)
		symbols = append(symbols, memberSyms...)
		refs = append(refs, memberRefs...)
	}

	return symbols, refs
}

func extractInterface(node *sitter.Node, src []byte, ns string) ([]parser.Symbol, []parser.RawReference) {
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

	qname := qualifyCSharp(ns, name)
	symbols = append(symbols, parser.Symbol{
		Name:          name,
		QualifiedName: qname,
		Kind:          "interface",
		Language:      "csharp",
		StartLine:     int(node.StartPoint().Row) + 1,
		EndLine:       int(node.EndPoint().Row) + 1,
	})

	return symbols, refs
}

func extractStruct(node *sitter.Node, src []byte, ns string) ([]parser.Symbol, []parser.RawReference) {
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

	qname := qualifyCSharp(ns, name)
	symbols = append(symbols, parser.Symbol{
		Name:          name,
		QualifiedName: qname,
		Kind:          "class",
		Language:      "csharp",
		StartLine:     int(node.StartPoint().Row) + 1,
		EndLine:       int(node.EndPoint().Row) + 1,
	})

	// Struct body members
	body := findChild(node, "declaration_list")
	if body != nil {
		memberSyms, memberRefs := extractMembers(body, src, ns, name)
		symbols = append(symbols, memberSyms...)
		refs = append(refs, memberRefs...)
	}

	return symbols, refs
}

func extractEnum(node *sitter.Node, src []byte, ns string) []parser.Symbol {
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
		QualifiedName: qualifyCSharp(ns, name),
		Kind:          "enum",
		Language:      "csharp",
		StartLine:     int(node.StartPoint().Row) + 1,
		EndLine:       int(node.EndPoint().Row) + 1,
	}}
}

func extractMembers(body *sitter.Node, src []byte, ns, typeName string) ([]parser.Symbol, []parser.RawReference) {
	var symbols []parser.Symbol
	var refs []parser.RawReference

	for i := 0; i < int(body.ChildCount()); i++ {
		child := body.Child(i)
		switch child.Type() {
		case "method_declaration":
			name, sig := extractMethodDecl(child, src)
			if name != "" {
				qname := qualifyCSharp(ns, typeName+"."+name)
				symbols = append(symbols, parser.Symbol{
					Name:          name,
					QualifiedName: qname,
					Kind:          "method",
					Language:      "csharp",
					StartLine:     int(child.StartPoint().Row) + 1,
					EndLine:       int(child.EndPoint().Row) + 1,
					Signature:     sig,
				})
			}

		case "constructor_declaration":
			name := typeName
			qname := qualifyCSharp(ns, typeName+"."+name)
			symbols = append(symbols, parser.Symbol{
				Name:          name,
				QualifiedName: qname,
				Kind:          "method",
				Language:      "csharp",
				StartLine:     int(child.StartPoint().Row) + 1,
				EndLine:       int(child.EndPoint().Row) + 1,
			})

		case "property_declaration":
			propName := extractPropertyName(child, src)
			if propName != "" {
				symbols = append(symbols, parser.Symbol{
					Name:          propName,
					QualifiedName: qualifyCSharp(ns, typeName+"."+propName),
					Kind:          "property",
					Language:      "csharp",
					StartLine:     int(child.StartPoint().Row) + 1,
					EndLine:       int(child.EndPoint().Row) + 1,
				})

				// Check for DbSet<T> properties
				dbSetType := extractDbSetType(child, src)
				if dbSetType != "" {
					refs = append(refs, parser.RawReference{
						FromSymbol:    qualifyCSharp(ns, typeName),
						ToName:        dbSetType,
						ReferenceType: "uses_table",
						Line:          int(child.StartPoint().Row) + 1,
					})
				}

				// Check for EF navigation properties (virtual ICollection<T>, virtual T)
				navType := extractNavigationProperty(child, src)
				if navType != "" {
					refs = append(refs, parser.RawReference{
						FromSymbol:    qualifyCSharp(ns, typeName),
						ToName:        navType,
						ReferenceType: "references",
						Confidence:    0.85,
						Line:          int(child.StartPoint().Row) + 1,
					})
				}
			}

		case "field_declaration":
			fieldName := extractFieldName(child, src)
			if fieldName != "" {
				symbols = append(symbols, parser.Symbol{
					Name:          fieldName,
					QualifiedName: qualifyCSharp(ns, typeName+"."+fieldName),
					Kind:          "field",
					Language:      "csharp",
					StartLine:     int(child.StartPoint().Row) + 1,
					EndLine:       int(child.EndPoint().Row) + 1,
				})
			}

		// Nested types
		case "class_declaration":
			syms, rfs := extractClass(child, src, ns)
			symbols = append(symbols, syms...)
			refs = append(refs, rfs...)

		case "interface_declaration":
			syms, rfs := extractInterface(child, src, ns)
			symbols = append(symbols, syms...)
			refs = append(refs, rfs...)
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
		if child.Type() == "parameter_list" {
			sig = child.Content(src)
		}
	}
	return name, sig
}

func extractPropertyName(node *sitter.Node, src []byte) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "identifier" {
			return child.Content(src)
		}
	}
	return ""
}

func extractFieldName(node *sitter.Node, src []byte) string {
	decl := findChild(node, "variable_declaration")
	if decl == nil {
		return ""
	}
	declarator := findChild(decl, "variable_declarator")
	if declarator != nil {
		for i := 0; i < int(declarator.ChildCount()); i++ {
			child := declarator.Child(i)
			if child.Type() == "identifier" {
				return child.Content(src)
			}
		}
	}
	return ""
}

func extractDbSetType(node *sitter.Node, src []byte) string {
	// Look for DbSet<T> in the property type
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "generic_name" {
			text := child.Content(src)
			if strings.HasPrefix(text, "DbSet<") {
				// Extract T from DbSet<T>
				inner := text[6 : len(text)-1]
				return inner
			}
		}
	}
	return ""
}

// extractNavigationProperty detects EF navigation properties:
// - virtual ICollection<Order> Orders { get; set; }
// - virtual IEnumerable<Order> Orders { get; set; }
// - virtual List<Order> Orders { get; set; }
// - virtual Customer Customer { get; set; }
func extractNavigationProperty(node *sitter.Node, src []byte) string {
	text := node.Content(src)

	// Must have 'virtual' modifier
	if !strings.Contains(text, "virtual") {
		return ""
	}

	// Collection navigation: ICollection<T>, IEnumerable<T>, List<T>, IList<T>, HashSet<T>
	collectionTypes := []string{"ICollection<", "IEnumerable<", "List<", "IList<", "HashSet<"}
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "generic_name" {
			childText := child.Content(src)
			for _, ct := range collectionTypes {
				if strings.HasPrefix(childText, ct) {
					inner := childText[len(ct) : len(childText)-1]
					return inner
				}
			}
		}
	}

	// Single navigation: virtual Customer Customer { get; set; }
	// Look for type_identifier that's a PascalCase name (not a primitive)
	hasVirtual := false
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "modifier" && child.Content(src) == "virtual" {
			hasVirtual = true
		}
	}
	if hasVirtual {
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "identifier" || child.Type() == "nullable_type" {
				typeName := child.Content(src)
				typeName = strings.TrimSuffix(typeName, "?")
				// Skip primitive types and common non-navigation types
				if !isPrimitiveType(typeName) && len(typeName) > 0 && typeName[0] >= 'A' && typeName[0] <= 'Z' {
					return typeName
				}
			}
		}
	}

	return ""
}

func isPrimitiveType(t string) bool {
	primitives := map[string]bool{
		"string": true, "String": true, "int": true, "Int32": true,
		"long": true, "Int64": true, "bool": true, "Boolean": true,
		"double": true, "Double": true, "float": true, "Single": true,
		"decimal": true, "Decimal": true, "DateTime": true, "DateTimeOffset": true,
		"Guid": true, "byte": true, "Byte": true, "char": true,
		"object": true, "Object": true, "void": true,
	}
	return primitives[t]
}

func extractBaseList(node *sitter.Node, src []byte, fromQName string) []parser.RawReference {
	var refs []parser.RawReference
	line := int(node.StartPoint().Row) + 1

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		typeName := ""

		switch child.Type() {
		case "identifier", "qualified_name":
			typeName = child.Content(src)
		case "generic_name":
			// Extract the base name before <T>
			for j := 0; j < int(child.ChildCount()); j++ {
				gc := child.Child(j)
				if gc.Type() == "identifier" {
					typeName = gc.Content(src)
					break
				}
			}
		case "simple_base_type":
			// simple_base_type wraps the actual type
			for j := 0; j < int(child.ChildCount()); j++ {
				gc := child.Child(j)
				switch gc.Type() {
				case "identifier", "qualified_name":
					typeName = gc.Content(src)
				case "generic_name":
					for k := 0; k < int(gc.ChildCount()); k++ {
						ggc := gc.Child(k)
						if ggc.Type() == "identifier" {
							typeName = ggc.Content(src)
							break
						}
					}
				}
				if typeName != "" {
					break
				}
			}
		}

		if typeName == "" {
			continue
		}

		if isInterfaceName(typeName) {
			refs = append(refs, parser.RawReference{
				FromSymbol:    fromQName,
				ToName:        typeName,
				ReferenceType: "implements",
				Line:          line,
			})
		} else {
			refs = append(refs, parser.RawReference{
				FromSymbol:    fromQName,
				ToName:        typeName,
				ReferenceType: "inherits",
				Line:          line,
			})
		}
	}

	return refs
}

// classRange holds byte range and qualified name for a class (used to resolve FromSymbol).
type classRange struct {
	start, end uint32
	qname      string
}

// buildClassRanges collects all class declarations with their ranges and qualified names.
func buildClassRanges(root *sitter.Node, src []byte, namespace string) []classRange {
	var ranges []classRange
	walkTree(root, func(node *sitter.Node) {
		if node.Type() != "class_declaration" {
			return
		}
		name := ""
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "identifier" {
				name = child.Content(src)
				break
			}
		}
		if name == "" {
			return
		}
		qname := qualifyCSharp(namespace, name)
		ranges = append(ranges, classRange{
			start: node.StartByte(),
			end:   node.EndByte(),
			qname: qname,
		})
	})
	return ranges
}

// findEnclosingClass returns the qualified name of the innermost class containing the given node.
func findEnclosingClass(node *sitter.Node, classRanges []classRange) string {
	start := node.StartByte()
	end := node.EndByte()
	var best *classRange
	for i := range classRanges {
		r := &classRanges[i]
		if r.start <= start && end <= r.end {
			if best == nil || (r.end-r.start) < (best.end-best.start) {
				best = r
			}
		}
	}
	if best == nil {
		return ""
	}
	return best.qname
}

func extractAttributeRefs(root *sitter.Node, src []byte, _ string, classRanges []classRange) []parser.RawReference {
	var refs []parser.RawReference

	walkTree(root, func(node *sitter.Node) {
		if node.Type() != "attribute" {
			return
		}

		text := node.Content(src)
		line := int(node.StartPoint().Row) + 1
		fromSymbol := findEnclosingClass(node, classRanges)

		// [Table("Users")]
		if strings.Contains(text, "Table") {
			tableName := extractAttributeStringParam(text)
			if tableName != "" {
				refs = append(refs, parser.RawReference{
					FromSymbol:    fromSymbol,
					ToName:        tableName,
					ToQualified:   "dbo." + tableName,
					ReferenceType: "uses_table",
					Line:          line,
				})
			}
		}
	})

	return refs
}

func extractInlineSQLRefs(root *sitter.Node, src []byte, _ string, classRanges []classRange) []parser.RawReference {
	var refs []parser.RawReference

	// Methods that take SQL statement strings (SELECT, INSERT, etc.)
	sqlStatementMethods := map[string]bool{
		"FromSqlRaw":             true,
		"FromSqlInterpolated":    true,
		"ExecuteSqlRaw":          true,
		"ExecuteSqlInterpolated": true,
		"SqlQuery":               true,
		"Query":                  true,
		"QueryFirst":             true,
		"QuerySingle":            true,
		"QueryFirstOrDefault":    true,
		"QueryAsync":             true,
		"QueryMultiple":          true,
		"QueryFirstAsync":        true,
		"QuerySingleAsync":       true,
	}

	// Methods that take a stored procedure NAME as first string arg
	procNameMethods := map[string]bool{
		"ExecuteNonQuery": true,
		"ExecuteReader":   true,
		"ExecuteScalar":   true,
		"Execute":         true,
		"ExecuteAsync":    true,
		"GetDataReader":   true,
		"GetData":         true,
		"BulkInsert":      true,
		"IDataReader":     true,
	}

	walkTree(root, func(node *sitter.Node) {
		if node.Type() != "invocation_expression" {
			return
		}

		line := int(node.StartPoint().Row) + 1
		fromSymbol := findEnclosingClass(node, classRanges)

		// Check if invocation calls a SQL method
		memberAccess := findChild(node, "member_access_expression")
		if memberAccess == nil {
			return
		}

		// The method name is the last identifier in the member access
		methodName := ""
		for i := 0; i < int(memberAccess.ChildCount()); i++ {
			child := memberAccess.Child(i)
			if child.Type() == "identifier" {
				methodName = child.Content(src)
			}
		}

		// Extract string literal argument
		argList := findChild(node, "argument_list")
		if argList == nil {
			return
		}

		if sqlStatementMethods[methodName] {
			// Existing behavior: extract SQL string, parse table refs
			for i := 0; i < int(argList.ChildCount()); i++ {
				arg := argList.Child(i)
				sqlStr := extractStringLiteral(arg, src)
				if sqlStr != "" && looksLikeSQL(sqlStr) {
					tableRefs := extractSQLTableRefs(sqlStr, line, fromSymbol)
					refs = append(refs, tableRefs...)
				}
			}
		} else if procNameMethods[methodName] {
			// First string arg is the proc name (or inline SQL)
			firstStr := extractFirstStringArg(argList, src)
			if firstStr == "" {
				return
			}
			if looksLikeSQL(firstStr) {
				// It's an inline SQL statement, extract table refs
				tableRefs := extractSQLTableRefs(firstStr, line, fromSymbol)
				refs = append(refs, tableRefs...)
			} else {
				// It's a stored procedure name
				procName := strings.TrimPrefix(firstStr, "dbo.")
				refs = append(refs, parser.RawReference{
					FromSymbol:    fromSymbol,
					ToName:        procName,
					ToQualified:   "dbo." + procName,
					ReferenceType: "calls",
					Line:          line,
				})
			}
		} else if methodName == "Include" || methodName == "ThenInclude" {
			// .Include("Orders") or .Include("Customer")
			firstStr := extractFirstStringArg(argList, src)
			if firstStr != "" {
				refs = append(refs, parser.RawReference{
					FromSymbol:    fromSymbol,
					ToName:        firstStr,
					ReferenceType: "references",
					Confidence:    0.8,
					Line:          line,
				})
			}
		}
	})

	return refs
}

// extractFirstStringArg returns the first string literal found in an argument list.
func extractFirstStringArg(argList *sitter.Node, src []byte) string {
	for i := 0; i < int(argList.ChildCount()); i++ {
		arg := argList.Child(i)
		if s := extractStringLiteral(arg, src); s != "" {
			return s
		}
	}
	return ""
}

// extractStoredProcRefs detects SqlCommand constructor and CommandText assignment patterns.
func extractStoredProcRefs(root *sitter.Node, src []byte, classRanges []classRange) []parser.RawReference {
	var refs []parser.RawReference

	walkTree(root, func(node *sitter.Node) {
		line := int(node.StartPoint().Row) + 1
		fromSymbol := findEnclosingClass(node, classRanges)

		switch node.Type() {
		case "object_creation_expression":
			// new SqlCommand("ProcName", ...)
			typeName := ""
			for i := 0; i < int(node.ChildCount()); i++ {
				child := node.Child(i)
				if child.Type() == "identifier" || child.Type() == "qualified_name" {
					typeName = child.Content(src)
					break
				}
			}
			if typeName != "SqlCommand" {
				return
			}
			argList := findChild(node, "argument_list")
			if argList == nil {
				return
			}
			firstStr := extractFirstStringArg(argList, src)
			if firstStr == "" {
				return
			}
			if looksLikeSQL(firstStr) {
				tableRefs := extractSQLTableRefs(firstStr, line, fromSymbol)
				refs = append(refs, tableRefs...)
			} else {
				procName := strings.TrimPrefix(firstStr, "dbo.")
				refs = append(refs, parser.RawReference{
					FromSymbol:    fromSymbol,
					ToName:        procName,
					ToQualified:   "dbo." + procName,
					ReferenceType: "calls",
					Line:          line,
				})
			}

		case "assignment_expression":
			// cmd.CommandText = "ProcName"
			left := node.Child(0)
			if left == nil {
				return
			}
			leftText := left.Content(src)
			if !strings.HasSuffix(leftText, ".CommandText") && leftText != "CommandText" {
				return
			}
			// Right side is the value after '='
			for i := 0; i < int(node.ChildCount()); i++ {
				child := node.Child(i)
				valStr := extractStringLiteral(child, src)
				if valStr == "" {
					continue
				}
				if looksLikeSQL(valStr) {
					tableRefs := extractSQLTableRefs(valStr, line, fromSymbol)
					refs = append(refs, tableRefs...)
				} else {
					procName := strings.TrimPrefix(valStr, "dbo.")
					refs = append(refs, parser.RawReference{
						FromSymbol:    fromSymbol,
						ToName:        procName,
						ToQualified:   "dbo." + procName,
						ReferenceType: "calls",
						Line:          line,
					})
				}
				return
			}
		}
	})

	return refs
}

func extractStringLiteral(node *sitter.Node, src []byte) string {
	// Walk into argument node to find string_literal or interpolated_string
	var result string
	walkTree(node, func(n *sitter.Node) {
		if result != "" {
			return
		}
		if n.Type() == "string_literal" || n.Type() == "verbatim_string_literal" {
			content := n.Content(src)
			// Strip quotes
			if len(content) >= 2 {
				if content[0] == '@' && len(content) >= 3 {
					result = content[2 : len(content)-1] // @"..."
				} else {
					result = content[1 : len(content)-1] // "..."
				}
			}
		}
	})
	return result
}

func extractAttributeStringParam(text string) string {
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

func qualifyCSharp(namespace, name string) string {
	if namespace != "" {
		return namespace + "." + name
	}
	return name
}

func isInterfaceName(name string) bool {
	// C# convention: interfaces start with 'I' followed by an uppercase letter
	if len(name) < 2 {
		return false
	}
	return name[0] == 'I' && name[1] >= 'A' && name[1] <= 'Z'
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

func looksLikeSQL(s string) bool {
	upper := strings.ToUpper(strings.TrimSpace(s))
	// Check for SQL keywords that should appear as whole words at the start
	// or preceded by whitespace (not as substrings of identifiers like "DeleteUser")
	for _, kw := range []string{"SELECT", "INSERT", "UPDATE", "DELETE", "FROM", "EXEC", "EXECUTE"} {
		if containsSQLKeyword(upper, kw) {
			return true
		}
	}
	return false
}

// containsSQLKeyword checks if kw appears as a word boundary in s
// (at start of string or after whitespace, followed by end/whitespace/punctuation).
func containsSQLKeyword(upper, kw string) bool {
	idx := 0
	for {
		pos := strings.Index(upper[idx:], kw)
		if pos < 0 {
			return false
		}
		absPos := idx + pos
		// Check left boundary: must be at start or preceded by whitespace/punctuation
		if absPos > 0 {
			ch := upper[absPos-1]
			if ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '_' {
				idx = absPos + len(kw)
				continue
			}
		}
		// Check right boundary: must be at end or followed by whitespace/punctuation
		endPos := absPos + len(kw)
		if endPos < len(upper) {
			ch := upper[endPos]
			if ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '_' {
				idx = absPos + len(kw)
				continue
			}
		}
		return true
	}
}

func extractSQLTableRefs(sql string, line int, fromSymbol string) []parser.RawReference {
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
			end := strings.IndexAny(rest, " \t\n,;)")
			tableName := rest
			if end > 0 {
				tableName = rest[:end]
			}
			tableName = strings.TrimSpace(tableName)
			if tableName != "" && !isSQLKeyword(tableName) {
				refs = append(refs, parser.RawReference{
					FromSymbol:    fromSymbol,
					ToName:        tableName,
					ToQualified:   "dbo." + tableName,
					ReferenceType: "uses_table",
					Line:          line,
				})
			}
			idx = pos
		}
	}

	// Extract EXEC/EXECUTE proc references
	for _, execKw := range []string{"EXEC ", "EXECUTE "} {
		idx := 0
		for {
			pos := strings.Index(upper[idx:], execKw)
			if pos < 0 {
				break
			}
			pos += idx + len(execKw)
			rest := strings.TrimSpace(sql[pos:])
			end := strings.IndexAny(rest, " \t\n,;(@")
			procName := rest
			if end > 0 {
				procName = rest[:end]
			}
			procName = strings.TrimSpace(procName)
			if procName != "" && !isSQLKeyword(procName) {
				refs = append(refs, parser.RawReference{
					FromSymbol:    fromSymbol,
					ToName:        procName,
					ToQualified:   "dbo." + procName,
					ReferenceType: "calls",
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
