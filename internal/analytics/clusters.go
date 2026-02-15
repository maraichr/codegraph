package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand/v2"

	"github.com/google/uuid"

	"github.com/maraichr/lattice/internal/store/postgres"
)

const (
	maxLabelPropIterations = 20
	clusterMinSize         = 3
)

// ComputeClusters runs label propagation community detection and stores cluster IDs.
func (e *Engine) ComputeClusters(ctx context.Context, projectID uuid.UUID) error {
	edges, err := e.store.GetEdgeList(ctx, projectID)
	if err != nil {
		return fmt.Errorf("get edge list: %w", err)
	}

	symbols, err := e.store.ListSymbolsByProject(ctx, projectID)
	if err != nil {
		return fmt.Errorf("list symbols: %w", err)
	}

	if len(symbols) == 0 {
		return nil
	}

	e.logger.Info("computing clusters via label propagation",
		slog.Int("symbols", len(symbols)),
		slog.Int("edges", len(edges)))

	// Build adjacency list (undirected)
	neighbors := make(map[uuid.UUID][]uuid.UUID)
	for _, edge := range edges {
		neighbors[edge.SourceID] = append(neighbors[edge.SourceID], edge.TargetID)
		neighbors[edge.TargetID] = append(neighbors[edge.TargetID], edge.SourceID)
	}

	// Initialize: each node gets its own label (cluster ID)
	nodeIndex := make(map[uuid.UUID]int, len(symbols))
	labels := make([]int, len(symbols))
	for i, sym := range symbols {
		nodeIndex[sym.ID] = i
		labels[i] = i
	}

	// Label propagation iterations
	order := make([]int, len(symbols))
	for i := range order {
		order[i] = i
	}

	for iter := range maxLabelPropIterations {
		changed := false

		// Randomize processing order
		rand.Shuffle(len(order), func(i, j int) {
			order[i], order[j] = order[j], order[i]
		})

		for _, idx := range order {
			sym := symbols[idx]
			nbrs := neighbors[sym.ID]
			if len(nbrs) == 0 {
				continue
			}

			// Count neighbor labels
			labelCounts := make(map[int]int)
			for _, nbr := range nbrs {
				if nbrIdx, ok := nodeIndex[nbr]; ok {
					labelCounts[labels[nbrIdx]]++
				}
			}

			// Pick most frequent label
			bestLabel := labels[idx]
			bestCount := 0
			for label, count := range labelCounts {
				if count > bestCount || (count == bestCount && label < bestLabel) {
					bestLabel = label
					bestCount = count
				}
			}

			if labels[idx] != bestLabel {
				labels[idx] = bestLabel
				changed = true
			}
		}

		if !changed {
			e.logger.Info("label propagation converged", slog.Int("iterations", iter+1))
			break
		}
	}

	// Remap labels to sequential cluster IDs and filter small clusters
	labelToCluster := make(map[int]int)
	clusterSizes := make(map[int]int)
	for _, label := range labels {
		clusterSizes[label]++
	}

	nextCluster := 1
	for label, size := range clusterSizes {
		if size >= clusterMinSize {
			labelToCluster[label] = nextCluster
			nextCluster++
		}
	}

	// Persist cluster IDs
	clusterCount := 0
	for i, sym := range symbols {
		clusterID, inCluster := labelToCluster[labels[i]]
		if !inCluster {
			clusterID = 0 // unclustered
		}

		meta := map[string]any{"cluster_id": clusterID}
		metaJSON, err := json.Marshal(meta)
		if err != nil {
			continue
		}
		if err := e.store.UpdateSymbolMetadata(ctx, postgres.UpdateSymbolMetadataParams{
			AnalyticsJson: metaJSON,
			SymbolID:      sym.ID,
		}); err != nil {
			e.logger.Warn("failed to update cluster", slog.String("symbol_id", sym.ID.String()))
		}
		if inCluster {
			clusterCount++
		}
	}

	// Store cluster summaries in project_analytics
	for label, clusterID := range labelToCluster {
		size := clusterSizes[label]
		clusterAnalytics := map[string]any{
			"cluster_id": clusterID,
			"size":       size,
		}
		clusterJSON, _ := json.Marshal(clusterAnalytics)
		summary := fmt.Sprintf("Cluster %d: %d symbols", clusterID, size)

		if _, err := e.store.UpsertProjectAnalytics(ctx, postgres.UpsertProjectAnalyticsParams{
			ProjectID: projectID,
			Scope:     "cluster",
			ScopeID:   fmt.Sprintf("%d", clusterID),
			Analytics: clusterJSON,
			Summary:   &summary,
		}); err != nil {
			e.logger.Warn("failed to upsert cluster analytics")
		}
	}

	e.logger.Info("clusters computed",
		slog.Int("clusters", len(labelToCluster)),
		slog.Int("clustered_symbols", clusterCount))

	return nil
}
