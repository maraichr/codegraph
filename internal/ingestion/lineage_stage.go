package ingestion

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/codegraph-labs/codegraph/internal/lineage"
	"github.com/codegraph-labs/codegraph/internal/parser"
)

// LineageStage builds column-level lineage edges from parsed column references.
type LineageStage struct {
	engine *lineage.Engine
	logger *slog.Logger
}

func NewLineageStage(e *lineage.Engine, logger *slog.Logger) *LineageStage {
	return &LineageStage{engine: e, logger: logger}
}

func (s *LineageStage) Name() string { return "lineage" }

func (s *LineageStage) Execute(ctx context.Context, rc *IndexRunContext) error {
	// Gather all column references from parse results
	var allColRefs []parser.ColumnReference
	for _, fr := range rc.ParseResults {
		allColRefs = append(allColRefs, fr.ColumnReferences...)
	}

	if len(allColRefs) == 0 {
		s.logger.Info("no column references to process")
		return nil
	}

	created, err := s.engine.BuildColumnLineage(ctx, rc.ProjectID, allColRefs)
	if err != nil {
		return fmt.Errorf("build column lineage: %w", err)
	}

	rc.EdgesFound += created
	return nil
}
