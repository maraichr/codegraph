package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"github.com/codegraph-labs/codegraph/internal/store/postgres"
)

// Layer represents an architectural layer classification.
type Layer string

const (
	LayerData           Layer = "data"
	LayerBusiness       Layer = "business"
	LayerAPI            Layer = "api"
	LayerInfrastructure Layer = "infrastructure"
	LayerCrossCutting   Layer = "cross-cutting"
	LayerUnknown        Layer = "unknown"
)

// dataKinds are symbol kinds inherently in the data layer.
var dataKinds = map[string]bool{
	"table": true, "view": true, "column": true, "procedure": true, "trigger": true,
}

// dataNamespacePatterns match data-layer namespaces.
var dataNamespacePatterns = []string{
	"repository", "repositories", "dal", "data", "dao",
	"persistence", "storage", "database", "db", "store",
	"dbo", "schema",
}

// businessNamespacePatterns match business-layer namespaces.
var businessNamespacePatterns = []string{
	"service", "services", "domain", "core", "business",
	"usecase", "usecases", "logic", "engine", "manager",
}

// apiNamespacePatterns match API-layer namespaces.
var apiNamespacePatterns = []string{
	"controller", "controllers", "handler", "handlers",
	"api", "endpoint", "endpoints", "rest", "graphql",
	"route", "routes", "web", "servlet",
}

// infraNamespacePatterns match infrastructure-layer namespaces.
var infraNamespacePatterns = []string{
	"config", "configuration", "startup", "infrastructure",
	"infra", "bootstrap", "setup", "middleware", "filter",
	"interceptor", "logging", "monitoring",
}

// ComputeLayers classifies symbols into architectural layers and persists as metadata.
func (e *Engine) ComputeLayers(ctx context.Context, projectID uuid.UUID) error {
	symbols, err := e.store.ListSymbolsByProject(ctx, projectID)
	if err != nil {
		return fmt.Errorf("list symbols: %w", err)
	}

	e.logger.Info("computing architectural layers", slog.Int("symbols", len(symbols)))

	counts := map[Layer]int{
		LayerData:           0,
		LayerBusiness:       0,
		LayerAPI:            0,
		LayerInfrastructure: 0,
		LayerCrossCutting:   0,
		LayerUnknown:        0,
	}

	for _, sym := range symbols {
		layer := classifyLayer(sym)
		counts[layer]++

		meta := map[string]any{"layer": string(layer)}
		metaJSON, err := json.Marshal(meta)
		if err != nil {
			continue
		}

		if err := e.store.UpdateSymbolMetadata(ctx, postgres.UpdateSymbolMetadataParams{
			AnalyticsJson: metaJSON,
			SymbolID:      sym.ID,
		}); err != nil {
			e.logger.Warn("failed to update layer",
				slog.String("symbol_id", sym.ID.String()),
				slog.String("error", err.Error()))
		}
	}

	// Store layer distribution in project_analytics
	layerAnalytics := map[string]any{"layer_distribution": counts}
	layerJSON, _ := json.Marshal(layerAnalytics)
	summary := fmt.Sprintf("Layer distribution: data=%d, business=%d, api=%d, infra=%d, cross-cutting=%d, unknown=%d",
		counts[LayerData], counts[LayerBusiness], counts[LayerAPI],
		counts[LayerInfrastructure], counts[LayerCrossCutting], counts[LayerUnknown])

	if _, err := e.store.UpsertProjectAnalytics(ctx, postgres.UpsertProjectAnalyticsParams{
		ProjectID: projectID,
		Scope:     "project",
		ScopeID:   "layers",
		Analytics: layerJSON,
		Summary:   &summary,
	}); err != nil {
		e.logger.Warn("failed to upsert layer analytics", slog.String("error", err.Error()))
	}

	e.logger.Info("layers computed",
		slog.Int("data", counts[LayerData]),
		slog.Int("business", counts[LayerBusiness]),
		slog.Int("api", counts[LayerAPI]),
		slog.Int("infra", counts[LayerInfrastructure]))

	return nil
}

func classifyLayer(sym postgres.Symbol) Layer {
	kind := strings.ToLower(sym.Kind)
	fqn := strings.ToLower(sym.QualifiedName)

	// 1. Kind-based classification (highest priority for SQL objects)
	if dataKinds[kind] {
		return LayerData
	}

	// 2. Namespace-based classification
	if matchesAnyPattern(fqn, apiNamespacePatterns) {
		return LayerAPI
	}
	if matchesAnyPattern(fqn, dataNamespacePatterns) {
		return LayerData
	}
	if matchesAnyPattern(fqn, businessNamespacePatterns) {
		return LayerBusiness
	}
	if matchesAnyPattern(fqn, infraNamespacePatterns) {
		return LayerInfrastructure
	}

	// 3. Kind-based hints for app code
	switch kind {
	case "interface":
		// Interfaces are often cross-cutting
		return LayerCrossCutting
	case "enum", "constant":
		return LayerCrossCutting
	}

	return LayerUnknown
}

func matchesAnyPattern(fqn string, patterns []string) bool {
	// Split FQN into segments by common delimiters
	segments := splitFQN(fqn)
	for _, segment := range segments {
		for _, pattern := range patterns {
			if segment == pattern {
				return true
			}
		}
	}
	return false
}

func splitFQN(fqn string) []string {
	// Split by common namespace delimiters: dot, slash, backslash
	var segments []string
	current := strings.Builder{}
	for _, r := range fqn {
		switch r {
		case '.', '/', '\\':
			if current.Len() > 0 {
				segments = append(segments, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		segments = append(segments, current.String())
	}
	return segments
}
