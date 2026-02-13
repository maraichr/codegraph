package models

import (
	"time"

	"github.com/google/uuid"
)

type EdgeType string

const (
	EdgeTypeCalls        EdgeType = "calls"
	EdgeTypeImports      EdgeType = "imports"
	EdgeTypeInherits     EdgeType = "inherits"
	EdgeTypeImplements   EdgeType = "implements"
	EdgeTypeReferences   EdgeType = "references"
	EdgeTypeContains     EdgeType = "contains"
	EdgeTypeDependsOn    EdgeType = "depends_on"
	EdgeTypeReadsFrom    EdgeType = "reads_from"
	EdgeTypeWritesTo     EdgeType = "writes_to"
	EdgeTypeUsesTable    EdgeType = "uses_table"
	EdgeTypeUsesColumn   EdgeType = "uses_column"
	EdgeTypeJoins        EdgeType = "joins"
	EdgeTypeTransformsTo EdgeType = "transforms_to"
)

type SymbolEdge struct {
	ID        uuid.UUID      `json:"id"`
	ProjectID uuid.UUID      `json:"project_id"`
	SourceID  uuid.UUID      `json:"source_id"`
	TargetID  uuid.UUID      `json:"target_id"`
	EdgeType  EdgeType       `json:"edge_type"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}
