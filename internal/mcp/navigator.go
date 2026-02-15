package mcp

import (
	"fmt"

	"github.com/maraichr/lattice/internal/mcp/session"
	"github.com/maraichr/lattice/internal/store/postgres"
)

// NavigationHints suggests next tool calls based on current results.
type NavigationHints struct {
	Steps []NavigationStep `json:"steps"`
}

// NavigationStep is a suggested next MCP tool call.
type NavigationStep struct {
	Tool            string            `json:"tool"`
	Description     string            `json:"description"`
	Params          map[string]string `json:"params,omitempty"`
	EstimatedTokens int               `json:"estimated_tokens,omitempty"`
}

// Navigator generates context-aware navigation hints for MCP tool responses.
type Navigator struct {
	store *postgres.Queries
}

// NewNavigator creates a navigator with access to the store for edge counting.
func NewNavigator(store *postgres.Queries) *Navigator {
	return &Navigator{store: store}
}

// symbolKindCategory classifies symbol kinds for navigation routing.
type symbolKindCategory int

const (
	categoryData symbolKindCategory = iota
	categoryCode
	categoryContainer
	categoryOther
)

func classifyKind(kind string) symbolKindCategory {
	switch kind {
	case "table", "view", "column":
		return categoryData
	case "function", "method", "procedure", "trigger":
		return categoryCode
	case "class", "interface", "module", "package":
		return categoryContainer
	default:
		return categoryOther
	}
}

// SuggestNextSteps returns navigation hints based on the tool that was just called
// and the symbols it returned.
func (n *Navigator) SuggestNextSteps(toolName string, symbols []postgres.Symbol, sess *session.Session) *NavigationHints {
	if len(symbols) == 0 {
		return nil
	}

	hints := &NavigationHints{}

	switch toolName {
	case "search_symbols":
		hints.Steps = n.hintsAfterSearch(symbols)
	case "get_symbol_details":
		hints.Steps = n.hintsAfterDetails(symbols)
	case "get_dependencies":
		hints.Steps = n.hintsAfterDependencies(symbols)
	case "trace_lineage":
		hints.Steps = n.hintsAfterLineage(symbols)
	case "list_project_overview":
		hints.Steps = n.hintsAfterOverview(symbols)
	case "find_usages":
		hints.Steps = n.hintsAfterUsages(symbols)
	case "analyze_impact":
		hints.Steps = n.hintsAfterImpact(symbols)
	default:
		hints.Steps = n.defaultHints(symbols)
	}

	// Limit to top 3 hints
	if len(hints.Steps) > 3 {
		hints.Steps = hints.Steps[:3]
	}

	return hints
}

func (n *Navigator) hintsAfterSearch(symbols []postgres.Symbol) []NavigationStep {
	steps := make([]NavigationStep, 0, 3)

	if len(symbols) > 0 {
		top := symbols[0]
		steps = append(steps, NavigationStep{
			Tool:            "get_symbol_details",
			Description:     fmt.Sprintf("Deep-dive into %s", top.Name),
			Params:          map[string]string{"symbol_id": top.ID.String()},
			EstimatedTokens: estimateDetailTokens(top),
		})

		cat := classifyKind(top.Kind)
		if cat == categoryData {
			steps = append(steps, NavigationStep{
				Tool:            "trace_lineage",
				Description:     fmt.Sprintf("Trace data flow for %s", top.Name),
				Params:          map[string]string{"symbol_id": top.ID.String(), "direction": "both"},
				EstimatedTokens: 800,
			})
		} else if cat == categoryCode || cat == categoryContainer {
			steps = append(steps, NavigationStep{
				Tool:            "get_dependencies",
				Description:     fmt.Sprintf("Show what %s depends on / is depended by", top.Name),
				Params:          map[string]string{"symbol_id": top.ID.String()},
				EstimatedTokens: 600,
			})
		}
	}

	if len(symbols) > 3 {
		steps = append(steps, NavigationStep{
			Tool:            "extract_subgraph",
			Description:     "Extract topic subgraph around these results",
			EstimatedTokens: 1200,
		})
	}

	return steps
}

func (n *Navigator) hintsAfterDetails(symbols []postgres.Symbol) []NavigationStep {
	if len(symbols) == 0 {
		return nil
	}

	sym := symbols[0]
	steps := []NavigationStep{
		{
			Tool:            "get_dependencies",
			Description:     fmt.Sprintf("Explore dependencies of %s", sym.Name),
			Params:          map[string]string{"symbol_id": sym.ID.String()},
			EstimatedTokens: 600,
		},
		{
			Tool:            "find_usages",
			Description:     fmt.Sprintf("Find all references to %s", sym.Name),
			Params:          map[string]string{"symbol_id": sym.ID.String()},
			EstimatedTokens: 400,
		},
	}

	if classifyKind(sym.Kind) == categoryData {
		steps = append(steps, NavigationStep{
			Tool:            "trace_lineage",
			Description:     fmt.Sprintf("Trace lineage through %s", sym.Name),
			Params:          map[string]string{"symbol_id": sym.ID.String()},
			EstimatedTokens: 800,
		})
	} else {
		steps = append(steps, NavigationStep{
			Tool:            "analyze_impact",
			Description:     fmt.Sprintf("Analyze impact of changing %s", sym.Name),
			Params:          map[string]string{"symbol_id": sym.ID.String()},
			EstimatedTokens: 1000,
		})
	}

	return steps
}

func (n *Navigator) hintsAfterDependencies(symbols []postgres.Symbol) []NavigationStep {
	steps := make([]NavigationStep, 0, 3)

	// Find unexplored high-value symbols
	for _, sym := range symbols {
		if classifyKind(sym.Kind) == categoryContainer || classifyKind(sym.Kind) == categoryData {
			steps = append(steps, NavigationStep{
				Tool:            "get_symbol_details",
				Description:     fmt.Sprintf("Examine %s (%s)", sym.Name, sym.Kind),
				Params:          map[string]string{"symbol_id": sym.ID.String()},
				EstimatedTokens: estimateDetailTokens(sym),
			})
			if len(steps) >= 2 {
				break
			}
		}
	}

	steps = append(steps, NavigationStep{
		Tool:            "extract_subgraph",
		Description:     "Extract the full topic subgraph",
		EstimatedTokens: 1200,
	})

	return steps
}

func (n *Navigator) hintsAfterLineage(symbols []postgres.Symbol) []NavigationStep {
	steps := make([]NavigationStep, 0, 3)

	for _, sym := range symbols {
		if classifyKind(sym.Kind) == categoryCode {
			steps = append(steps, NavigationStep{
				Tool:            "get_symbol_details",
				Description:     fmt.Sprintf("Examine transformer %s", sym.Name),
				Params:          map[string]string{"symbol_id": sym.ID.String()},
				EstimatedTokens: estimateDetailTokens(sym),
			})
			break
		}
	}

	steps = append(steps, NavigationStep{
		Tool:            "analyze_impact",
		Description:     "Assess blast radius of changes to this data flow",
		EstimatedTokens: 1000,
	})

	return steps
}

func (n *Navigator) hintsAfterOverview(_ []postgres.Symbol) []NavigationStep {
	return []NavigationStep{
		{
			Tool:            "search_symbols",
			Description:     "Search for specific symbols by name or kind",
			EstimatedTokens: 400,
		},
		{
			Tool:            "extract_subgraph",
			Description:     "Extract a topic subgraph (e.g., 'order processing')",
			EstimatedTokens: 1200,
		},
	}
}

func (n *Navigator) hintsAfterUsages(symbols []postgres.Symbol) []NavigationStep {
	steps := make([]NavigationStep, 0, 2)

	for _, sym := range symbols {
		steps = append(steps, NavigationStep{
			Tool:            "get_symbol_details",
			Description:     fmt.Sprintf("Examine caller %s", sym.Name),
			Params:          map[string]string{"symbol_id": sym.ID.String()},
			EstimatedTokens: estimateDetailTokens(sym),
		})
		if len(steps) >= 2 {
			break
		}
	}

	return steps
}

func (n *Navigator) hintsAfterImpact(symbols []postgres.Symbol) []NavigationStep {
	steps := make([]NavigationStep, 0, 2)

	for _, sym := range symbols {
		if classifyKind(sym.Kind) == categoryData || classifyKind(sym.Kind) == categoryContainer {
			steps = append(steps, NavigationStep{
				Tool:            "get_symbol_details",
				Description:     fmt.Sprintf("Examine impacted %s (%s)", sym.Name, sym.Kind),
				Params:          map[string]string{"symbol_id": sym.ID.String()},
				EstimatedTokens: estimateDetailTokens(sym),
			})
			if len(steps) >= 2 {
				break
			}
		}
	}

	return steps
}

func (n *Navigator) defaultHints(symbols []postgres.Symbol) []NavigationStep {
	if len(symbols) == 0 {
		return nil
	}
	sym := symbols[0]
	return []NavigationStep{
		{
			Tool:            "get_symbol_details",
			Description:     fmt.Sprintf("Examine %s", sym.Name),
			Params:          map[string]string{"symbol_id": sym.ID.String()},
			EstimatedTokens: estimateDetailTokens(sym),
		},
	}
}

func estimateDetailTokens(sym postgres.Symbol) int {
	base := 200
	if sym.DocComment != nil {
		base += len(*sym.DocComment) / 4
	}
	if sym.Signature != nil {
		base += len(*sym.Signature) / 4
	}
	return base
}
