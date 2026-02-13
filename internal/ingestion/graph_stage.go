package ingestion

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/codegraph-labs/codegraph/internal/graph"
	"github.com/codegraph-labs/codegraph/internal/store"
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
	if err := s.graph.SyncFiles(ctx, rc.ProjectID, files); err != nil {
		return fmt.Errorf("sync files to neo4j: %w", err)
	}

	// Sync symbols
	if err := s.graph.SyncSymbols(ctx, rc.ProjectID, symbols); err != nil {
		return fmt.Errorf("sync symbols to neo4j: %w", err)
	}

	// Sync edges
	if err := s.graph.SyncEdges(ctx, rc.ProjectID, edges); err != nil {
		return fmt.Errorf("sync edges to neo4j: %w", err)
	}

	return nil
}
