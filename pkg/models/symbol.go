package models

import (
	"time"

	"github.com/google/uuid"
)

type SymbolKind string

const (
	SymbolKindFunction  SymbolKind = "function"
	SymbolKindMethod    SymbolKind = "method"
	SymbolKindClass     SymbolKind = "class"
	SymbolKindInterface SymbolKind = "interface"
	SymbolKindVariable  SymbolKind = "variable"
	SymbolKindConstant  SymbolKind = "constant"
	SymbolKindPackage   SymbolKind = "package"
	SymbolKindModule    SymbolKind = "module"
	SymbolKindTable     SymbolKind = "table"
	SymbolKindColumn    SymbolKind = "column"
	SymbolKindView      SymbolKind = "view"
	SymbolKindProcedure SymbolKind = "procedure"
	SymbolKindTrigger   SymbolKind = "trigger"
	SymbolKindType      SymbolKind = "type"
	SymbolKindEnum      SymbolKind = "enum"
	SymbolKindField     SymbolKind = "field"
	SymbolKindProperty  SymbolKind = "property"
)

type Symbol struct {
	ID            uuid.UUID  `json:"id"`
	ProjectID     uuid.UUID  `json:"project_id"`
	FileID        uuid.UUID  `json:"file_id"`
	Name          string     `json:"name"`
	QualifiedName string     `json:"qualified_name"`
	Kind          SymbolKind `json:"kind"`
	Language      string     `json:"language"`
	StartLine     int        `json:"start_line"`
	EndLine       int        `json:"end_line"`
	StartCol      *int       `json:"start_col,omitempty"`
	EndCol        *int       `json:"end_col,omitempty"`
	Signature     *string    `json:"signature,omitempty"`
	DocComment    *string    `json:"doc_comment,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type File struct {
	ID            uuid.UUID  `json:"id"`
	ProjectID     uuid.UUID  `json:"project_id"`
	SourceID      uuid.UUID  `json:"source_id"`
	Path          string     `json:"path"`
	Language      string     `json:"language"`
	SizeBytes     int64      `json:"size_bytes"`
	Hash          string     `json:"hash"`
	LastIndexedAt *time.Time `json:"last_indexed_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}
