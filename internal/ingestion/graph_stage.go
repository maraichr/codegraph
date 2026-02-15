package ingestion

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/maraichr/codegraph/internal/graph"
	"github.com/maraichr/codegraph/internal/store"
)

// GraphStage syncs symbols and edges from PostgreSQL to Neo4j.
type GraphStage struct {
	store  *store.Store
	graph  *graph.Client
	logger *slog.Logger
}

func NewGraphStage(s *store.Store, g *graph.Client, logger *slog.Logger) *GraphStage {
	return &GraphStage{store: s, graph: g, logger: logger}
}

func (s *GraphStage) Name() string { return "graph_build" }

func (s *GraphStage) Execute(ctx context.Context, rc *IndexRunContext) error {
	// Load all files for project
	files, err := s.store.ListFilesByProject(ctx, rc.ProjectID)
	if err != nil {
		return fmt.Errorf("load files: %w", err)
	}

	// Load all symbols for project
	symbols, err := s.store.ListSymbolsByProject(ctx, rc.ProjectID)
	if err != nil {
		return fmt.Errorf("load symbols: %w", err)
	}

	// Load all edges for project
	edges, err := s.store.ListEdgesByProject(ctx, rc.ProjectID)
	if err != nil {
		return fmt.Errorf("load edges: %w", err)
	}

	s.logger.Info("syncing to neo4j",
		slog.Int("files", len(files)),
		slog.Int("symbols", len(symbols)),
		slog.Int("edges", len(edges)))

	// Sync files
	s.logger.Info("neo4j: syncing files", slog.Int("count", len(files)))
	if err := s.graph.SyncFiles(ctx, rc.ProjectID, files); err != nil {
		return fmt.Errorf("sync files to neo4j: %w", err)
	}
	s.logger.Info("neo4j: files synced")

	// Sync symbols
	s.logger.Info("neo4j: syncing symbols", slog.Int("count", len(symbols)))
	if err := s.graph.SyncSymbols(ctx, rc.ProjectID, symbols); err != nil {
		return fmt.Errorf("sync symbols to neo4j: %w", err)
	}
	s.logger.Info("neo4j: symbols synced")

	// Sync edges (DEPENDS_ON relationships)
	s.logger.Info("neo4j: syncing edges", slog.Int("count", len(edges)))
	if err := s.graph.SyncEdges(ctx, rc.ProjectID, edges); err != nil {
		return fmt.Errorf("sync edges to neo4j: %w", err)
	}
	s.logger.Info("neo4j: edges synced")

	// Sync column-level edges (COLUMN_FLOW relationships)
	if err := s.graph.SyncColumnEdges(ctx, rc.ProjectID, edges); err != nil {
		return fmt.Errorf("sync column edges to neo4j: %w", err)
	}

	return nil
}
