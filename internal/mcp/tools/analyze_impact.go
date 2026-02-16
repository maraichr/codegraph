package tools

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/maraichr/lattice/internal/auth"
	"github.com/maraichr/lattice/internal/mcp"
	"github.com/maraichr/lattice/internal/store"
	"github.com/maraichr/lattice/internal/store/postgres"
)

// AnalyzeImpactParams are the parameters for the analyze_impact tool.
type AnalyzeImpactParams struct {
	Project    string `json:"project"`
	SymbolID   string `json:"symbol_id,omitempty"`
	SymbolName string `json:"symbol_name,omitempty"`
	ChangeType string `json:"change_type,omitempty"` // modify, delete, rename
	MaxDepth   int    `json:"max_depth,omitempty"`
}

// AnalyzeImpactHandler implements the analyze_impact MCP tool.
type AnalyzeImpactHandler struct {
	store  *store.Store
	logger *slog.Logger
}

// NewAnalyzeImpactHandler creates a new handler.
func NewAnalyzeImpactHandler(s *store.Store, logger *slog.Logger) *AnalyzeImpactHandler {
	return &AnalyzeImpactHandler{store: s, logger: logger}
}

// Handle performs downstream impact analysis from a symbol.
func (h *AnalyzeImpactHandler) Handle(ctx context.Context, params AnalyzeImpactParams) (string, error) {
	if params.SymbolID == "" && params.SymbolName == "" {
		return "", fmt.Errorf("symbol_id or symbol_name is required")
	}
	if params.MaxDepth <= 0 {
		params.MaxDepth = 3
	}
	if params.ChangeType == "" {
		params.ChangeType = "modify"
	}

	project, err := h.store.GetProject(ctx, params.Project)
	if err != nil {
		return "", WrapProjectError(err)
	}
	if p, ok := auth.PrincipalFrom(ctx); ok && !p.IsAdmin() && project.TenantID != p.TenantID {
		return "", fmt.Errorf("access denied to project %s", params.Project)
	}

	// Resolve seed symbol (reuse lineage's resolveSeed pattern)
	seed, err := h.resolveSeed(ctx, project, params)
	if err != nil {
		return "", err
	}

	// BFS downstream to find all affected symbols
	type impactNode struct {
		Symbol     postgres.Symbol
		Depth      int
		EdgeType   string
		Confidence float64
	}

	visited := map[uuid.UUID]bool{seed.ID: true}
	var direct, transitive []impactNode

	queue := []impactNode{{Symbol: seed, Depth: 0}}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur.Depth >= params.MaxDepth {
			continue
		}

		edges, err := h.store.GetOutgoingEdges(ctx, cur.Symbol.ID)
		if err != nil {
			continue
		}
		for _, e := range edges {
			if visited[e.TargetID] {
				continue
			}
			visited[e.TargetID] = true
			sym, err := h.store.GetSymbol(ctx, e.TargetID)
			if err != nil {
				continue
			}
			node := impactNode{Symbol: sym, Depth: cur.Depth + 1, EdgeType: e.EdgeType, Confidence: extractEdgeConfidence(e.Metadata)}
			if cur.Depth == 0 {
				direct = append(direct, node)
			} else {
				transitive = append(transitive, node)
			}
			queue = append(queue, node)
		}
	}

	// Also check incoming edges for "who references this" (reverse impact)
	inEdges, _ := h.store.GetIncomingEdges(ctx, seed.ID)
	var callers []impactNode
	for _, e := range inEdges {
		if visited[e.SourceID] {
			continue
		}
		sym, err := h.store.GetSymbol(ctx, e.SourceID)
		if err != nil {
			continue
		}
		callers = append(callers, impactNode{Symbol: sym, Depth: 1, EdgeType: e.EdgeType, Confidence: extractEdgeConfidence(e.Metadata)})
	}

	// Format response
	rb := mcp.NewResponseBuilder(4000)
	rb.AddHeader(fmt.Sprintf("**Impact Analysis: %s %s**", params.ChangeType, seed.Name))
	rb.AddLine(fmt.Sprintf("Symbol: `%s` (%s, %s)", seed.QualifiedName, seed.Kind, seed.Language))
	total := len(direct) + len(transitive) + len(callers)
	rb.AddLine(fmt.Sprintf("Total affected: %d direct, %d transitive, %d callers/references",
		len(direct), len(transitive), len(callers)))
	rb.AddLine("")

	if len(direct) > 0 {
		rb.AddLine("### Direct Impact")
		for _, n := range direct {
			severity := classifyImpactSeverity(params.ChangeType, n.EdgeType)
			confStr := ""
			if n.Confidence > 0 {
				confStr = fmt.Sprintf(", confidence: %.2f", n.Confidence)
			}
			rb.AddLine(fmt.Sprintf("- %s `%s` [%s] via %s%s — **%s**",
				n.Symbol.Kind, n.Symbol.Name, n.Symbol.Language, n.EdgeType, confStr, severity))
		}
		rb.AddLine("")
	}

	if len(transitive) > 0 {
		rb.AddLine("### Transitive Impact")
		for _, n := range transitive {
			confStr := ""
			if n.Confidence > 0 {
				confStr = fmt.Sprintf(", confidence: %.2f", n.Confidence)
			}
			rb.AddLine(fmt.Sprintf("- %s `%s` [%s] (depth %d, via %s%s)",
				n.Symbol.Kind, n.Symbol.Name, n.Symbol.Language, n.Depth, n.EdgeType, confStr))
		}
		rb.AddLine("")
	}

	if len(callers) > 0 {
		rb.AddLine("### Callers / References (will need updating)")
		for _, n := range callers {
			confStr := ""
			if n.Confidence > 0 {
				confStr = fmt.Sprintf(", confidence: %.2f", n.Confidence)
			}
			rb.AddLine(fmt.Sprintf("- %s `%s` [%s] via %s%s",
				n.Symbol.Kind, n.Symbol.Name, n.Symbol.Language, n.EdgeType, confStr))
		}
	}

	if len(direct) == 0 && len(transitive) == 0 && len(callers) == 0 {
		rb.AddLine("No downstream impact found. This symbol appears to be a leaf node.")
	}

	return rb.Finalize(total, total), nil
}

func classifyImpactSeverity(changeType, edgeType string) string {
	switch changeType {
	case "delete":
		switch edgeType {
		case "calls", "references", "inherits", "implements":
			return "BREAKING"
		default:
			return "HIGH"
		}
	case "rename":
		switch edgeType {
		case "calls", "references":
			return "BREAKING"
		default:
			return "MEDIUM"
		}
	default: // modify
		switch edgeType {
		case "calls", "inherits", "implements":
			return "HIGH"
		default:
			return "LOW"
		}
	}
}

func (h *AnalyzeImpactHandler) resolveSeed(ctx context.Context, project postgres.Project, params AnalyzeImpactParams) (postgres.Symbol, error) {
	if params.SymbolID != "" {
		id, err := uuid.Parse(params.SymbolID)
		if err != nil {
			return postgres.Symbol{}, fmt.Errorf("invalid symbol_id: %w", err)
		}
		sym, err := h.store.GetSymbol(ctx, id)
		if err != nil {
			return postgres.Symbol{}, WrapSymbolError(err)
		}
		return sym, nil
	}

	// Search by name with ranking
	return ResolveSymbolByName(ctx, h.store, project.Slug, params.SymbolName)
}
