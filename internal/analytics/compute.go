package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"

	"github.com/google/uuid"

	"github.com/maraichr/codegraph/internal/store"
	"github.com/maraichr/codegraph/internal/store/postgres"
)

const (
	pageRankIterations = 20
	pageRankDamping    = 0.85
	batchSize          = 500
)

// Engine computes graph analytics (centrality, summaries, bridges, layers) for a project.
type Engine struct {
	store  *store.Store
	logger *slog.Logger
}

// NewEngine creates a new analytics engine.
func NewEngine(s *store.Store, logger *slog.Logger) *Engine {
	return &Engine{store: s, logger: logger}
}

// ComputeAll runs all analytics for a project: degrees, PageRank, summaries, bridges, layers.
func (e *Engine) ComputeAll(ctx context.Context, projectID uuid.UUID) error {
	e.logger.Info("computing analytics", slog.String("project_id", projectID.String()))

	if err := e.ComputeDegrees(ctx, projectID); err != nil {
		return fmt.Errorf("compute degrees: %w", err)
	}

	if err := e.ComputePageRank(ctx, projectID); err != nil {
		return fmt.Errorf("compute pagerank: %w", err)
	}

	if err := e.ComputeLayers(ctx, projectID); err != nil {
		return fmt.Errorf("compute layers: %w", err)
	}

	if err := e.ComputeProjectSummaries(ctx, projectID); err != nil {
		return fmt.Errorf("compute summaries: %w", err)
	}

	if err := e.ComputeCrossLanguageBridges(ctx, projectID); err != nil {
		return fmt.Errorf("compute bridges: %w", err)
	}

	e.logger.Info("analytics complete", slog.String("project_id", projectID.String()))
	return nil
}

// ComputeDegrees calculates in-degree and out-degree for all symbols in a project.
func (e *Engine) ComputeDegrees(ctx context.Context, projectID uuid.UUID) error {
	degrees, err := e.store.GetSymbolDegrees(ctx, projectID)
	if err != nil {
		return fmt.Errorf("get symbol degrees: %w", err)
	}

	e.logger.Info("computing degrees", slog.Int("symbols", len(degrees)))

	for i := 0; i < len(degrees); i += batchSize {
		end := i + batchSize
		if end > len(degrees) {
			end = len(degrees)
		}
		batch := degrees[i:end]

		for _, d := range batch {
			meta := map[string]any{
				"in_degree":  d.InDegree,
				"out_degree": d.OutDegree,
			}
			metaJSON, err := json.Marshal(meta)
			if err != nil {
				continue
			}
			if err := e.store.UpdateSymbolMetadata(ctx, postgres.UpdateSymbolMetadataParams{
				AnalyticsJson: metaJSON,
				SymbolID:      d.ID,
			}); err != nil {
				e.logger.Warn("failed to update degree", slog.String("symbol_id", d.ID.String()), slog.String("error", err.Error()))
			}
		}
	}

	e.logger.Info("degrees computed", slog.Int("symbols", len(degrees)))
	return nil
}

// ComputePageRank runs iterative PageRank over the symbol graph.
func (e *Engine) ComputePageRank(ctx context.Context, projectID uuid.UUID) error {
	edges, err := e.store.GetEdgeList(ctx, projectID)
	if err != nil {
		return fmt.Errorf("get edge list: %w", err)
	}

	if len(edges) == 0 {
		e.logger.Info("no edges for pagerank")
		return nil
	}

	// Build adjacency lists and collect node set
	nodeSet := make(map[uuid.UUID]struct{})
	outLinks := make(map[uuid.UUID][]uuid.UUID)
	for _, edge := range edges {
		nodeSet[edge.SourceID] = struct{}{}
		nodeSet[edge.TargetID] = struct{}{}
		outLinks[edge.SourceID] = append(outLinks[edge.SourceID], edge.TargetID)
	}

	n := len(nodeSet)
	if n == 0 {
		return nil
	}

	e.logger.Info("computing pagerank",
		slog.Int("nodes", n),
		slog.Int("edges", len(edges)),
		slog.Int("iterations", pageRankIterations))

	// Initialize ranks
	initRank := 1.0 / float64(n)
	rank := make(map[uuid.UUID]float64, n)
	for node := range nodeSet {
		rank[node] = initRank
	}

	// Iterative computation
	for iter := range pageRankIterations {
		newRank := make(map[uuid.UUID]float64, n)
		sinkRank := 0.0

		// Accumulate sink node (no outlinks) rank
		for node := range nodeSet {
			if _, hasOut := outLinks[node]; !hasOut {
				sinkRank += rank[node]
			}
		}

		base := (1.0-pageRankDamping)/float64(n) + pageRankDamping*sinkRank/float64(n)

		for node := range nodeSet {
			newRank[node] = base
		}

		for src, targets := range outLinks {
			share := pageRankDamping * rank[src] / float64(len(targets))
			for _, tgt := range targets {
				newRank[tgt] += share
			}
		}

		rank = newRank

		if iter == pageRankIterations-1 {
			// Check convergence on last iteration
			var maxDiff float64
			for node := range nodeSet {
				diff := math.Abs(rank[node] - newRank[node])
				if diff > maxDiff {
					maxDiff = diff
				}
			}
			e.logger.Debug("pagerank iteration", slog.Int("iter", iter), slog.Float64("max_diff", maxDiff))
		}
	}

	// Persist PageRank values
	count := 0
	for node, pr := range rank {
		meta := map[string]any{"pagerank": math.Round(pr*1e6) / 1e6}
		metaJSON, err := json.Marshal(meta)
		if err != nil {
			continue
		}
		if err := e.store.UpdateSymbolMetadata(ctx, postgres.UpdateSymbolMetadataParams{
			AnalyticsJson: metaJSON,
			SymbolID:      node,
		}); err != nil {
			e.logger.Warn("failed to update pagerank", slog.String("symbol_id", node.String()))
		}
		count++
	}

	e.logger.Info("pagerank computed", slog.Int("nodes", count))
	return nil
}

// ComputeProjectSummaries generates aggregate analytics stored in project_analytics.
func (e *Engine) ComputeProjectSummaries(ctx context.Context, projectID uuid.UUID) error {
	// Project-level summary
	stats, err := e.store.GetProjectSymbolStats(ctx, projectID)
	if err != nil {
		return fmt.Errorf("get project symbol stats: %w", err)
	}

	langCounts, err := e.store.GetSymbolCountsByLanguage(ctx, projectID)
	if err != nil {
		return fmt.Errorf("get language counts: %w", err)
	}

	kindCounts, err := e.store.GetSymbolCountsByKind(ctx, projectID)
	if err != nil {
		return fmt.Errorf("get kind counts: %w", err)
	}

	edgeCount, err := e.store.CountEdgesByProject(ctx, projectID)
	if err != nil {
		return fmt.Errorf("count edges: %w", err)
	}

	// Top 10 hotspots by in-degree
	hotspots, err := e.store.TopSymbolsByInDegree(ctx, postgres.TopSymbolsByInDegreeParams{
		ProjectID: projectID,
		Limit:     10,
	})
	if err != nil {
		e.logger.Warn("failed to get hotspots", slog.String("error", err.Error()))
	}

	projectAnalytics := map[string]any{
		"total_symbols":  stats.TotalSymbols,
		"total_files":    stats.FileCount,
		"total_edges":    edgeCount,
		"language_count": stats.LanguageCount,
		"kind_count":     stats.KindCount,
	}

	// Language breakdown
	languages := make(map[string]int64)
	for _, lc := range langCounts {
		languages[lc.Language] = lc.Cnt
	}
	projectAnalytics["languages"] = languages

	// Kind breakdown
	kinds := make(map[string]int64)
	for _, kc := range kindCounts {
		kinds[kc.Kind] = kc.Cnt
	}
	projectAnalytics["kinds"] = kinds

	// Hotspots
	hotspotList := make([]map[string]any, 0, len(hotspots))
	for _, h := range hotspots {
		hotspotList = append(hotspotList, map[string]any{
			"id":        h.ID,
			"name":      h.Name,
			"kind":      h.Kind,
			"in_degree": h.InDegree,
		})
	}
	projectAnalytics["hotspots"] = hotspotList

	analyticsJSON, _ := json.Marshal(projectAnalytics)

	// Generate text summary
	summary := generateProjectSummary(stats, langCounts, kindCounts, edgeCount)

	if _, err := e.store.UpsertProjectAnalytics(ctx, postgres.UpsertProjectAnalyticsParams{
		ProjectID: projectID,
		Scope:     "project",
		ScopeID:   "overview",
		Analytics: analyticsJSON,
		Summary:   &summary,
	}); err != nil {
		return fmt.Errorf("upsert project analytics: %w", err)
	}

	// Source-level summaries
	sourceStats, err := e.store.GetSourceSymbolStats(ctx, projectID)
	if err != nil {
		e.logger.Warn("failed to get source stats", slog.String("error", err.Error()))
	} else {
		for _, ss := range sourceStats {
			sourceAnalytics := map[string]any{
				"symbol_count":   ss.SymbolCount,
				"file_count":     ss.FileCount,
				"language_count": ss.LanguageCount,
				"languages":      ss.Languages,
				"kinds":          ss.Kinds,
			}
			sourceJSON, _ := json.Marshal(sourceAnalytics)
			sourceSummary := fmt.Sprintf("Source contains %d symbols across %d files in %d language(s).",
				ss.SymbolCount, ss.FileCount, ss.LanguageCount)

			if _, err := e.store.UpsertProjectAnalytics(ctx, postgres.UpsertProjectAnalyticsParams{
				ProjectID: projectID,
				Scope:     "source",
				ScopeID:   ss.SourceID.String(),
				Analytics: sourceJSON,
				Summary:   &sourceSummary,
			}); err != nil {
				e.logger.Warn("failed to upsert source analytics", slog.String("source_id", ss.SourceID.String()))
			}
		}
	}

	// Namespace-level summaries
	nsStats, err := e.store.GetNamespaceStats(ctx, postgres.GetNamespaceStatsParams{
		ProjectID: projectID,
		Limit:     50,
	})
	if err != nil {
		e.logger.Warn("failed to get namespace stats", slog.String("error", err.Error()))
	} else {
		for _, ns := range nsStats {
			nsAnalytics := map[string]any{
				"symbol_count": ns.SymbolCount,
				"kinds":        ns.Kinds,
				"languages":    ns.Languages,
			}
			nsJSON, _ := json.Marshal(nsAnalytics)
			nsSummary := fmt.Sprintf("Namespace %s contains %d symbols.", ns.Namespace, ns.SymbolCount)

			if _, err := e.store.UpsertProjectAnalytics(ctx, postgres.UpsertProjectAnalyticsParams{
				ProjectID: projectID,
				Scope:     "namespace",
				ScopeID:   ns.Namespace,
				Analytics: nsJSON,
				Summary:   &nsSummary,
			}); err != nil {
				e.logger.Warn("failed to upsert namespace analytics", slog.String("namespace", ns.Namespace))
			}
		}
	}

	e.logger.Info("project summaries computed")
	return nil
}

// ComputeCrossLanguageBridges finds and stores cross-language boundary edges.
func (e *Engine) ComputeCrossLanguageBridges(ctx context.Context, projectID uuid.UUID) error {
	bridges, err := e.store.GetCrossLanguageBridges(ctx, projectID)
	if err != nil {
		return fmt.Errorf("get cross-language bridges: %w", err)
	}

	if len(bridges) == 0 {
		e.logger.Info("no cross-language bridges found")
		return nil
	}

	for _, bridge := range bridges {
		scopeID := fmt.Sprintf("%s→%s", bridge.SourceLanguage, bridge.TargetLanguage)
		bridgeAnalytics := map[string]any{
			"source_language": bridge.SourceLanguage,
			"target_language": bridge.TargetLanguage,
			"edge_type":       bridge.EdgeType,
			"edge_count":      bridge.EdgeCount,
		}
		bridgeJSON, _ := json.Marshal(bridgeAnalytics)
		summary := fmt.Sprintf("%s → %s: %d %s edges",
			bridge.SourceLanguage, bridge.TargetLanguage, bridge.EdgeCount, bridge.EdgeType)

		if _, err := e.store.UpsertProjectAnalytics(ctx, postgres.UpsertProjectAnalyticsParams{
			ProjectID: projectID,
			Scope:     "bridge",
			ScopeID:   scopeID,
			Analytics: bridgeJSON,
			Summary:   &summary,
		}); err != nil {
			e.logger.Warn("failed to upsert bridge analytics", slog.String("bridge", scopeID))
		}
	}

	e.logger.Info("cross-language bridges computed", slog.Int("bridge_types", len(bridges)))
	return nil
}

func generateProjectSummary(
	stats postgres.GetProjectSymbolStatsRow,
	langCounts []postgres.GetSymbolCountsByLanguageRow,
	kindCounts []postgres.GetSymbolCountsByKindRow,
	edgeCount int64,
) string {
	var summary string
	summary = fmt.Sprintf("This project contains %d symbols across %d files with %d edges. ",
		stats.TotalSymbols, stats.FileCount, edgeCount)

	if len(langCounts) > 0 {
		summary += "Languages: "
		for i, lc := range langCounts {
			if i > 0 {
				summary += ", "
			}
			if i >= 5 {
				summary += fmt.Sprintf("and %d more", len(langCounts)-5)
				break
			}
			summary += fmt.Sprintf("%s (%d)", lc.Language, lc.Cnt)
		}
		summary += ". "
	}

	if len(kindCounts) > 0 {
		summary += "Primary symbol kinds: "
		for i, kc := range kindCounts {
			if i > 0 {
				summary += ", "
			}
			if i >= 5 {
				summary += fmt.Sprintf("and %d more", len(kindCounts)-5)
				break
			}
			summary += fmt.Sprintf("%s (%d)", kc.Kind, kc.Cnt)
		}
		summary += "."
	}

	return summary
}
