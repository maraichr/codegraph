package parser

import "github.com/google/uuid"

// Parser extracts symbols and references from source files.
type Parser interface {
	// Parse processes a single file and returns extracted symbols and references.
	Parse(input FileInput) (*ParseResult, error)

	// Languages returns the SQL dialects/languages this parser handles.
	Languages() []string
}

// FileInput represents a file to be parsed.
type FileInput struct {
	Path     string
	Content  []byte
	Language string
}

// ColumnReference represents a column-level data flow relationship.
type ColumnReference struct {
	SourceColumn   string // qualified: schema.table.column
	TargetColumn   string // qualified: schema.table.column
	DerivationType string // direct_copy, transform, aggregate, filter, join, conditional
	Expression     string // SQL expression (e.g., "UPPER(first_name)")
	Context        string // containing symbol qualified name (the proc/view)
	Line           int
}

// ParseResult contains extracted symbols and raw references from a file.
type ParseResult struct {
	Symbols          []Symbol
	References       []RawReference
	ColumnReferences []ColumnReference
}

// Symbol represents a code symbol (table, view, procedure, function, etc.)
type Symbol struct {
	Name          string
	QualifiedName string
	Kind          string // table, view, procedure, function, trigger, column, type, etc.
	Language      string
	StartLine     int
	EndLine       int
	StartCol      int
	EndCol        int
	Signature     string
	DocComment    string
	Children      []Symbol // e.g., columns within a table
}

// RawReference represents an unresolved reference from one symbol to another.
type RawReference struct {
	FromSymbol    string // qualified name of the source symbol
	ToName        string // name being referenced (may be unqualified)
	ToQualified   string // qualified name if available
	ReferenceType string // calls, reads_from, writes_to, uses_table, etc.
	Line          int
	Col           int
}

// FileResult pairs parse results with file metadata for persistence.
type FileResult struct {
	ProjectID        uuid.UUID
	SourceID         uuid.UUID
	Path             string
	Language         string
	SizeBytes        int64
	Hash             string
	Symbols          []Symbol
	References       []RawReference
	ColumnReferences []ColumnReference
}
