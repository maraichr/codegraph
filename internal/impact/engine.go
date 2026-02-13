package impact

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/codegraph-labs/codegraph/internal/graph"
	"github.com/codegraph-labs/codegraph/internal/store"
)

// SymbolSummary is a lightweight representation of a symbol for impact results.
type SymbolSummary struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	QualifiedName string `json:"qualified_name"`
	Kind          string `json:"kind"`
	Language      string `json:"language"`
	FilePath      string `json:"file_path,omitempty"`
}

// ImpactNode represents a symbol affected by a change.
type ImpactNode struct {
	Symbol   SymbolSummary `json:"symbol"`
	Depth    int           `json:"depth"`
	Severity string        `json:"severity"` // critical, high, medium, low
	EdgeType string        `json:"edge_type"`
	Path     []string      `json:"path"`
}

// ImpactResult contains the full impact analysis for a symbol change.
type ImpactResult struct {
	Root             SymbolSummary `json:"root"`
	ChangeType       string        `json:"change_type"`
	DirectImpact     []ImpactNode  `json:"direct_impact"`
	TransitiveImpact []ImpactNode  `json:"transitive_impact"`
	TotalAffected    int           `json:"total_affected"`
}

// Engine performs impact analysis using Neo4j lineage data.
type Engine struct {
	graph  *graph.Client
	store  *store.Store
	logger *slog.Logger
}

// NewEngine creates a new impact analysis engine.
func NewEngine(g *graph.Client, s *store.Store, logger *slog.Logger) *Engine {
	return &Engine{graph: g, store: s, logger: logger}
}

// Analyze computes the downstream impact of changing a symbol.
func (e *Engine) Analyze(ctx context.Context, symbolID uuid.UUID, changeType string, maxDepth int) (*ImpactResult, error) {
	if e.graph == nil {
		return nil, fmt.Errorf("neo4j not configured")
	}

	if maxDepth <= 0 || maxDepth > 10 {
		maxDepth = 5
	}

	// Get the root symbol info
	sym, err := e.store.GetSymbol(ctx, symbolID)
	if err != nil {
		return nil, fmt.Errorf("get root symbol: %w", err)
	}

	root := SymbolSummary{
		ID:            sym.ID.String(),
		Name:          sym.Name,
		QualifiedName: sym.QualifiedName,
		Kind:          sym.Kind,
		Language:      sym.Language,
	}

	// Query downstream lineage from Neo4j
	lineageResult, err := e.graph.Lineage(ctx, symbolID, "downstream", maxDepth)
	if err != nil {
		return nil, fmt.Errorf("lineage query: %w", err)
	}

	// Build adjacency list for depth computation
	adjacency := make(map[string][]graph.LineageEdge)
	for _, edge := range lineageResult.Edges {
		adjacency[edge.SourceID] = append(adjacency[edge.SourceID], edge)
	}

	// Build node map for lookup
	nodeMap := make(map[string]graph.LineageNode)
	for _, n := range lineageResult.Nodes {
		nodeMap[n.ID] = n
	}

	// BFS to compute depth and paths from root
	type bfsEntry struct {
		id    string
		depth int
		path  []string
		edge  string
	}

	visited := make(map[string]bool)
	visited[symbolID.String()] = true
	queue := []bfsEntry{{id: symbolID.String(), depth: 0, path: []string{symbolID.String()}}}

	var direct, transitive []ImpactNode

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, edge := range adjacency[current.id] {
			if visited[edge.TargetID] {
				continue
			}
			visited[edge.TargetID] = true

			depth := current.depth + 1
			path := append(append([]string{}, current.path...), edge.TargetID)

			node, exists := nodeMap[edge.TargetID]
			if !exists {
				continue
			}

			severity := classifySeverity(depth, edge.EdgeType, changeType)
			impactNode := ImpactNode{
				Symbol: SymbolSummary{
					ID:            node.ID,
					Name:          node.Name,
					QualifiedName: node.QualifiedName,
					Kind:          node.Kind,
					Language:      node.Language,
				},
				Depth:    depth,
				Severity: severity,
				EdgeType: edge.EdgeType,
				Path:     path,
			}

			if depth == 1 {
				direct = append(direct, impactNode)
			} else {
				transitive = append(transitive, impactNode)
			}

			if depth < maxDepth {
				queue = append(queue, bfsEntry{id: edge.TargetID, depth: depth, path: path, edge: edge.EdgeType})
			}
		}
	}

	if direct == nil {
		direct = []ImpactNode{}
	}
	if transitive == nil {
		transitive = []ImpactNode{}
	}

	result := &ImpactResult{
		Root:             root,
		ChangeType:       changeType,
		DirectImpact:     direct,
		TransitiveImpact: transitive,
		TotalAffected:    len(direct) + len(transitive),
	}

	e.logger.Info("impact analysis complete",
		slog.String("symbol", sym.QualifiedName),
		slog.String("change_type", changeType),
		slog.Int("total_affected", result.TotalAffected))

	return result, nil
}

// classifySeverity determines the impact severity based on depth, edge type, and change type.
func classifySeverity(depth int, edgeType, changeType string) string {
	if depth == 1 {
		if changeType == "delete" {
			switch edgeType {
			case "writes_to", "reads_from", "calls":
				return "critical"
			}
			return "high"
		}
		switch edgeType {
		case "calls", "transforms_to":
			return "high"
		default:
			return "medium"
		}
	}
	if depth == 2 {
		return "medium"
	}
	return "low"
}
