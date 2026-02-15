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

// GetLineageParams are the parameters for the get_lineage tool.
type GetLineageParams struct {
	Project    string `json:"project"`
	SymbolID   string `json:"symbol_id,omitempty"`
	SymbolName string `json:"symbol_name,omitempty"`
	Direction  string `json:"direction,omitempty"` // upstream, downstream, both
	MaxDepth   int    `json:"max_depth,omitempty"`
}

// GetLineageHandler implements the get_lineage MCP tool.
type GetLineageHandler struct {
	store  *store.Store
	logger *slog.Logger
}

// NewGetLineageHandler creates a new handler.
func NewGetLineageHandler(s *store.Store, logger *slog.Logger) *GetLineageHandler {
	return &GetLineageHandler{store: s, logger: logger}
}

// Handle traces upstream or downstream lineage from a symbol.
func (h *GetLineageHandler) Handle(ctx context.Context, params GetLineageParams) (string, error) {
	if params.SymbolID == "" && params.SymbolName == "" {
		return "", fmt.Errorf("symbol_id or symbol_name is required")
	}
	if params.MaxDepth <= 0 {
		params.MaxDepth = 3
	}
	if params.Direction == "" {
		params.Direction = "both"
	}

	project, err := h.store.GetProject(ctx, params.Project)
	if err != nil {
		return "", WrapProjectError(err)
	}
	if p, ok := auth.PrincipalFrom(ctx); ok && !p.IsAdmin() && project.TenantID != p.TenantID {
		return "", fmt.Errorf("access denied to project %s", params.Project)
	}

	// Resolve the seed symbol
	seed, err := h.resolveSeed(ctx, project, params)
	if err != nil {
		return "", err
	}

	// BFS lineage traversal
	type lineageNode struct {
		Symbol postgres.Symbol
		Depth  int
		Via    string // edge type that led here
	}

	visited := map[uuid.UUID]bool{seed.ID: true}
	var upstream, downstream []lineageNode

	// Upstream: follow incoming edges
	if params.Direction == "upstream" || params.Direction == "both" {
		queue := []lineageNode{{Symbol: seed, Depth: 0}}
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			if cur.Depth >= params.MaxDepth {
				continue
			}
			edges, err := h.store.GetIncomingEdges(ctx, cur.Symbol.ID)
			if err != nil {
				continue
			}
			for _, e := range edges {
				if visited[e.SourceID] {
					continue
				}
				visited[e.SourceID] = true
				sym, err := h.store.GetSymbol(ctx, e.SourceID)
				if err != nil {
					continue
				}
				node := lineageNode{Symbol: sym, Depth: cur.Depth + 1, Via: e.EdgeType}
				upstream = append(upstream, node)
				queue = append(queue, node)
			}
		}
	}

	// Downstream: follow outgoing edges
	if params.Direction == "downstream" || params.Direction == "both" {
		// Reset visited for downstream except seed
		if params.Direction == "both" {
			visited = map[uuid.UUID]bool{seed.ID: true}
		}
		queue := []lineageNode{{Symbol: seed, Depth: 0}}
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
				node := lineageNode{Symbol: sym, Depth: cur.Depth + 1, Via: e.EdgeType}
				downstream = append(downstream, node)
				queue = append(queue, node)
			}
		}
	}

	// Format response
	rb := mcp.NewResponseBuilder(4000)
	rb.AddHeader(fmt.Sprintf("**Lineage for: %s** (%s)", seed.Name, params.Direction))

	if len(upstream) > 0 {
		rb.AddLine("### Upstream (data sources / callers)")
		for _, n := range upstream {
			indent := ""
			for i := 0; i < n.Depth; i++ {
				indent += "  "
			}
			rb.AddLine(fmt.Sprintf("%s- %s `%s` [%s] (via %s)", indent, n.Symbol.Kind, n.Symbol.Name, n.Symbol.Language, n.Via))
		}
		rb.AddLine("")
	}

	if len(downstream) > 0 {
		rb.AddLine("### Downstream (consumers / dependents)")
		for _, n := range downstream {
			indent := ""
			for i := 0; i < n.Depth; i++ {
				indent += "  "
			}
			rb.AddLine(fmt.Sprintf("%s- %s `%s` [%s] (via %s)", indent, n.Symbol.Kind, n.Symbol.Name, n.Symbol.Language, n.Via))
		}
		rb.AddLine("")
	}

	if len(upstream) == 0 && len(downstream) == 0 {
		rb.AddLine("No lineage connections found for this symbol.")
	}

	return rb.Finalize(len(upstream)+len(downstream), len(upstream)+len(downstream)), nil
}

func (h *GetLineageHandler) resolveSeed(ctx context.Context, project postgres.Project, params GetLineageParams) (postgres.Symbol, error) {
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
