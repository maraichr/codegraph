package graphql

import (
	"encoding/json"
	"strings"

	"github.com/codegraph-labs/codegraph/internal/impact"
	"github.com/codegraph-labs/codegraph/internal/store/postgres"
	"github.com/codegraph-labs/codegraph/pkg/models"
)

func dbProjectToGQL(p postgres.Project) *Project {
	return &Project{
		ID:          p.ID.String(),
		Name:        p.Name,
		Slug:        p.Slug,
		Description: p.Description,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func dbSourceToGQL(s postgres.Source) *Source {
	st := SourceType(strings.ToUpper(s.SourceType))
	result := &Source{
		ID:         s.ID.String(),
		Name:       s.Name,
		SourceType: st,
		CreatedAt:  s.CreatedAt,
	}
	if s.LastSyncedAt.Valid {
		t := s.LastSyncedAt.Time
		result.LastSyncedAt = &t
	}
	return result
}

func dbIndexRunToGQL(ir postgres.IndexRun) *IndexRun {
	result := &IndexRun{
		ID:             ir.ID.String(),
		Status:         IndexRunStatus(strings.ToUpper(ir.Status)),
		FilesProcessed: int(ir.FilesProcessed),
		SymbolsFound:   int(ir.SymbolsFound),
		EdgesFound:     int(ir.EdgesFound),
		ErrorMessage:   ir.ErrorMessage,
		CreatedAt:      ir.CreatedAt,
	}
	if ir.StartedAt.Valid {
		t := ir.StartedAt.Time
		result.StartedAt = &t
	}
	if ir.CompletedAt.Valid {
		t := ir.CompletedAt.Time
		result.CompletedAt = &t
	}
	return result
}

func dbSymbolToGQL(s postgres.Symbol) *models.Symbol {
	sym := &models.Symbol{
		ID:            s.ID,
		ProjectID:     s.ProjectID,
		FileID:        s.FileID,
		Name:          s.Name,
		QualifiedName: s.QualifiedName,
		Kind:          models.SymbolKind(s.Kind),
		Language:      s.Language,
		StartLine:     int(s.StartLine),
		EndLine:       int(s.EndLine),
		Signature:     s.Signature,
		DocComment:    s.DocComment,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}
	if s.StartCol != nil {
		v := int(*s.StartCol)
		sym.StartCol = &v
	}
	if s.EndCol != nil {
		v := int(*s.EndCol)
		sym.EndCol = &v
	}
	if len(s.Metadata) > 0 {
		_ = json.Unmarshal(s.Metadata, &sym.Metadata)
	}
	return sym
}

func dbFileToGQL(f postgres.File) *models.File {
	file := &models.File{
		ID:        f.ID,
		ProjectID: f.ProjectID,
		SourceID:  f.SourceID,
		Path:      f.Path,
		Language:  f.Language,
		SizeBytes: f.SizeBytes,
		Hash:      f.Hash,
		CreatedAt: f.CreatedAt,
		UpdatedAt: f.UpdatedAt,
	}
	if f.LastIndexedAt.Valid {
		t := f.LastIndexedAt.Time
		file.LastIndexedAt = &t
	}
	return file
}

func dbEdgeToGQL(e postgres.SymbolEdge) *models.SymbolEdge {
	edge := &models.SymbolEdge{
		ID:        e.ID,
		ProjectID: e.ProjectID,
		SourceID:  e.SourceID,
		TargetID:  e.TargetID,
		EdgeType:  models.EdgeType(e.EdgeType),
		CreatedAt: e.CreatedAt,
	}
	if len(e.Metadata) > 0 {
		_ = json.Unmarshal(e.Metadata, &edge.Metadata)
	}
	return edge
}

func impactNodeToGQL(n impact.ImpactNode) *ImpactNode {
	return &ImpactNode{
		Symbol: &ImpactSymbol{
			ID:            n.Symbol.ID,
			Name:          n.Symbol.Name,
			QualifiedName: n.Symbol.QualifiedName,
			Kind:          n.Symbol.Kind,
			Language:      n.Symbol.Language,
		},
		Depth:    n.Depth,
		Severity: Severity(strings.ToUpper(n.Severity)),
		EdgeType: n.EdgeType,
		Path:     n.Path,
	}
}

func filterEdges(dbEdges []postgres.SymbolEdge, types []models.EdgeType) []*models.SymbolEdge {
	typeSet := make(map[string]bool, len(types))
	for _, t := range types {
		typeSet[strings.ToLower(string(t))] = true
	}

	var result []*models.SymbolEdge
	for _, e := range dbEdges {
		if len(typeSet) > 0 && !typeSet[strings.ToLower(e.EdgeType)] {
			continue
		}
		result = append(result, dbEdgeToGQL(e))
	}
	if result == nil {
		result = []*models.SymbolEdge{}
	}
	return result
}
