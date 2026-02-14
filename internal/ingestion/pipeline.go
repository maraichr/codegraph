package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/codegraph-labs/codegraph/internal/store"
	"github.com/codegraph-labs/codegraph/internal/store/postgres"
)

// Pipeline orchestrates the indexing stages for each ingestion job.
type Pipeline struct {
	store  *store.Store
	stages []Stage
	logger *slog.Logger
}

func NewPipeline(s *store.Store, stages []Stage, logger *slog.Logger) *Pipeline {
	return &Pipeline{store: s, stages: stages, logger: logger}
}

// Run processes a single ingestion message through all pipeline stages.
func (p *Pipeline) Run(ctx context.Context, msg IngestMessage) error {
	p.logger.Info("pipeline started",
		slog.String("index_run_id", msg.IndexRunID.String()),
		slog.String("source_type", msg.SourceType))

	// Mark as running
	if err := p.store.UpdateIndexRunStatus(ctx, postgres.UpdateIndexRunStatusParams{
		ID:     msg.IndexRunID,
		Status: "running",
	}); err != nil {
		return fmt.Errorf("update status to running: %w", err)
	}

	rc := &IndexRunContext{
		IndexRunID: msg.IndexRunID,
		ProjectID:  msg.ProjectID,
		SourceID:   msg.SourceID,
		SourceType: msg.SourceType,
		Trigger:    msg.Trigger,
	}

	// Load project settings for optional lineage_exclude_paths
	if proj, err := p.store.GetProjectByID(ctx, msg.ProjectID); err == nil && len(proj.Settings) > 0 {
		var settings struct {
			LineageExcludePaths []string `json:"lineage_exclude_paths"`
		}
		if json.Unmarshal(proj.Settings, &settings) == nil && len(settings.LineageExcludePaths) > 0 {
			rc.LineageExcludePaths = settings.LineageExcludePaths
		}
	}

	for _, stage := range p.stages {
		p.logger.Info("stage started", slog.String("stage", stage.Name()),
			slog.String("index_run_id", msg.IndexRunID.String()))

		if err := stage.Execute(ctx, rc); err != nil {
			errMsg := err.Error()
			_ = p.store.UpdateIndexRunStatus(ctx, postgres.UpdateIndexRunStatusParams{
				ID:           msg.IndexRunID,
				Status:       "failed",
				ErrorMessage: &errMsg,
			})
			return fmt.Errorf("stage %s failed: %w", stage.Name(), err)
		}

		p.logger.Info("stage completed", slog.String("stage", stage.Name()),
			slog.String("index_run_id", msg.IndexRunID.String()))
	}

	// Save commit SHA for incremental indexing on next run
	if rc.CurrentSHA != "" {
		_ = p.store.UpdateSourceLastCommitSHA(ctx, postgres.UpdateSourceLastCommitSHAParams{
			ID:            rc.SourceID,
			LastCommitSha: &rc.CurrentSHA,
		})
	}

	// Update stats and mark complete
	_ = p.store.UpdateIndexRunStats(ctx, postgres.UpdateIndexRunStatsParams{
		ID:             msg.IndexRunID,
		FilesProcessed: int32(rc.FilesProcessed),
		SymbolsFound:   int32(rc.SymbolsFound),
		EdgesFound:     int32(rc.EdgesFound),
	})

	if err := p.store.UpdateIndexRunStatus(ctx, postgres.UpdateIndexRunStatusParams{
		ID:     msg.IndexRunID,
		Status: "completed",
	}); err != nil {
		return fmt.Errorf("update status to completed: %w", err)
	}

	p.logger.Info("pipeline completed",
		slog.String("index_run_id", msg.IndexRunID.String()),
		slog.Int("files", rc.FilesProcessed),
		slog.Int("symbols", rc.SymbolsFound),
		slog.Int("edges", rc.EdgesFound))

	return nil
}

// NoOpStage is a placeholder stage that just logs.
type NoOpStage struct {
	name string
}

func NewNoOpStage(name string) *NoOpStage {
	return &NoOpStage{name: name}
}

func (s *NoOpStage) Name() string { return s.name }

func (s *NoOpStage) Execute(_ context.Context, _ *IndexRunContext) error {
	return nil
}

