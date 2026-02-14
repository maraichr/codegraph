package ingestion

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/codegraph-labs/codegraph/internal/analytics"
)

// AnalyticsStage computes graph analytics after embedding.
// Runs: degree counts, PageRank, layer classification, project summaries, cross-language bridges.
type AnalyticsStage struct {
	engine *analytics.Engine
	logger *slog.Logger
}

func NewAnalyticsStage(engine *analytics.Engine, logger *slog.Logger) *AnalyticsStage {
	return &AnalyticsStage{engine: engine, logger: logger}
}

func (s *AnalyticsStage) Name() string { return "analytics" }

func (s *AnalyticsStage) Execute(ctx context.Context, rc *IndexRunContext) error {
	s.logger.Info("running analytics stage", slog.String("project_id", rc.ProjectID.String()))

	if err := s.engine.ComputeAll(ctx, rc.ProjectID); err != nil {
		return fmt.Errorf("compute analytics: %w", err)
	}

	return nil
}
