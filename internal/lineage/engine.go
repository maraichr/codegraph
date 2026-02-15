package lineage

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"github.com/maraichr/lattice/internal/graph"
	"github.com/maraichr/lattice/internal/parser"
	"github.com/maraichr/lattice/internal/store"
	"github.com/maraichr/lattice/internal/store/postgres"
)

// Engine handles column-level lineage building and querying.
type Engine struct {
	store  *store.Store
	graph  *graph.Client
	logger *slog.Logger
}

// NewEngine creates a new lineage engine.
func NewEngine(s *store.Store, g *graph.Client, logger *slog.Logger) *Engine {
	return &Engine{store: s, graph: g, logger: logger}
}

// BuildColumnLineage resolves column references to symbol IDs and creates edges.
// Returns the number of edges created.
func (e *Engine) BuildColumnLineage(ctx context.Context, projectID uuid.UUID, colRefs []parser.ColumnReference) (int, error) {
	// Load all column symbols for the project
	columns, err := e.store.ListColumnSymbolsByProject(ctx, projectID)
	if err != nil {
		return 0, fmt.Errorf("load column symbols: %w", err)
	}

	// Build lookup maps: qualified_name → symbol ID (case-insensitive)
	fqnMap := make(map[string]uuid.UUID, len(columns))
	for _, col := range columns {
		fqnMap[strings.ToLower(col.QualifiedName)] = col.ID
	}

	// Also load all symbols for name-based fallback resolution
	allSymbols, err := e.store.ListSymbolsByProject(ctx, projectID)
	if err != nil {
		return 0, fmt.Errorf("load symbols: %w", err)
	}
	symbolFQN := make(map[string]uuid.UUID, len(allSymbols))
	for _, sym := range allSymbols {
		symbolFQN[strings.ToLower(sym.QualifiedName)] = sym.ID
	}

	created := 0
	skipped := 0
	for _, ref := range colRefs {
		sourceID := resolveColumnID(ref.SourceColumn, fqnMap, symbolFQN)
		targetID := resolveColumnID(ref.TargetColumn, fqnMap, symbolFQN)

		// Fallback: if target is unresolvable (e.g. "proc.ColName"), try the
		// parent symbol (e.g. "proc"). This creates column→procedure edges
		// that let users trace which procedures read/write each column.
		if targetID == uuid.Nil && ref.TargetColumn != "" {
			if parts := strings.Split(ref.TargetColumn, "."); len(parts) > 1 {
				parent := strings.Join(parts[:len(parts)-1], ".")
				targetID = resolveColumnID(parent, fqnMap, symbolFQN)
			}
		}

		// Same fallback for source: "@ParamName" won't resolve, but the
		// procedure context might (e.g. UPDATE SET col = @param).
		if sourceID == uuid.Nil && ref.SourceColumn != "" && ref.Context != "" {
			sourceID = resolveColumnID(ref.Context, fqnMap, symbolFQN)
		}

		if sourceID == uuid.Nil || targetID == uuid.Nil || sourceID == targetID {
			skipped++
			continue
		}

		edgeType := mapDerivationToEdgeType(ref.DerivationType)
		confidence := derivationConfidence(ref.DerivationType)
		metadata := map[string]interface{}{
			"derivation_type": ref.DerivationType,
			"confidence":      confidence,
		}
		if ref.Expression != "" {
			metadata["expression"] = ref.Expression
		}
		metaJSON, _ := json.Marshal(metadata)

		_, err := e.store.CreateSymbolEdgeWithMetadata(ctx, postgres.CreateSymbolEdgeWithMetadataParams{
			ProjectID: projectID,
			SourceID:  sourceID,
			TargetID:  targetID,
			EdgeType:  edgeType,
			Metadata:  metaJSON,
		})
		if err != nil {
			continue
		}
		created++
	}

	e.logger.Info("column lineage built",
		slog.Int("edges_created", created),
		slog.Int("refs_skipped", skipped),
		slog.Int("column_refs_processed", len(colRefs)))

	return created, nil
}

// QueryColumnLineage queries Neo4j for column-level lineage.
func (e *Engine) QueryColumnLineage(ctx context.Context, symbolID uuid.UUID, direction string, maxDepth int) (*graph.ColumnLineageResult, error) {
	if e.graph == nil {
		return nil, fmt.Errorf("neo4j not configured")
	}

	return e.graph.ColumnLineage(ctx, symbolID, direction, maxDepth)
}

func resolveColumnID(name string, colMap, allMap map[string]uuid.UUID) uuid.UUID {
	lower := strings.ToLower(name)

	// Exact match
	if id, ok := colMap[lower]; ok {
		return id
	}
	if id, ok := allMap[lower]; ok {
		return id
	}

	// Suffix match: "table.col" matches "schema.table.col"
	suffix := "." + lower
	for fqn, id := range colMap {
		if strings.HasSuffix(fqn, suffix) {
			return id
		}
	}
	for fqn, id := range allMap {
		if strings.HasSuffix(fqn, suffix) {
			return id
		}
	}

	// Short name fallback (bare name without dots)
	if !strings.Contains(name, ".") {
		for fqn, id := range colMap {
			parts := strings.Split(fqn, ".")
			if strings.EqualFold(parts[len(parts)-1], name) {
				return id
			}
		}
	}

	return uuid.Nil
}

func mapDerivationToEdgeType(derivation string) string {
	switch derivation {
	case "direct_copy":
		return "direct_copy"
	case "transform", "aggregate", "conditional":
		return "transforms_to"
	case "filter", "join":
		return "uses_column"
	default:
		return "uses_column"
	}
}

// derivationConfidence returns a 0–1 score for lineage edge confidence (for filtering/down-ranking).
func derivationConfidence(derivation string) float64 {
	switch derivation {
	case "transform", "aggregate", "conditional":
		return 1.0
	case "direct_copy":
		return 0.9
	case "filter", "join":
		return 0.85
	default:
		return 0.7
	}
}
