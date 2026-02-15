package oracle

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/maraichr/codegraph/internal/graph"
	"github.com/maraichr/codegraph/internal/impact"
	"github.com/maraichr/codegraph/internal/store"
	"github.com/maraichr/codegraph/internal/store/postgres"
)

const maxResults = 20

// executeSearch finds symbols by name/description.
func executeSearch(ctx context.Context, s *store.Store, projectSlug string, params map[string]any) ([]Block, []SymbolItem, error) {
	query := stringParam(params, "query")
	if query == "" {
		query = stringParam(params, "symbol_name")
	}
	kinds := stringSliceParam(params, "kinds")
	languages := stringSliceParam(params, "languages")

	results, err := s.SearchSymbols(ctx, postgres.SearchSymbolsParams{
		ProjectSlug: projectSlug,
		Query:       &query,
		Kinds:       kinds,
		Languages:   languages,
		Lim:         maxResults,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("search symbols: %w", err)
	}

	if len(results) == 0 {
		return []Block{
			headerBlock("Search Results"),
			textBlock(fmt.Sprintf("No symbols found matching '%s'.", query)),
		}, nil, nil
	}

	items := symbolsToItems(results)
	shown := len(items)
	blocks := []Block{
		headerBlock(fmt.Sprintf("Search Results for \"%s\"", query)),
		symbolListBlock(items),
	}
	if shown < len(results) {
		blocks = append(blocks, truncationBlock(shown, len(results)))
	}
	return blocks, items, nil
}

// executeRanking finds the most important symbols.
func executeRanking(ctx context.Context, s *store.Store, projectSlug string, params map[string]any) ([]Block, []SymbolItem, error) {
	kinds := stringSliceParam(params, "kinds")
	metric := stringParam(params, "metric")
	if metric == "" {
		metric = "in_degree"
	}

	results, err := s.ListTopSymbolsByKind(ctx, postgres.ListTopSymbolsByKindParams{
		ProjectSlug: projectSlug,
		Kinds:       kinds,
		Languages:   []string{},
		Lim:         maxResults,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("list top symbols: %w", err)
	}

	if len(results) == 0 {
		return []Block{
			headerBlock("Top Symbols"),
			textBlock("No symbols found matching the criteria."),
		}, nil, nil
	}

	items := symbolsToItems(results)
	kindLabel := "symbols"
	if len(kinds) > 0 {
		kindLabel = kinds[0] + "s"
	}

	blocks := []Block{
		headerBlock(fmt.Sprintf("Top %s by %s", kindLabel, metric)),
		symbolListBlock(items),
	}
	return blocks, items, nil
}

// executeOverview returns project summary.
func executeOverview(ctx context.Context, s *store.Store, projectID uuid.UUID, projectName string) ([]Block, error) {
	analytics, err := s.GetProjectAnalytics(ctx, postgres.GetProjectAnalyticsParams{
		ProjectID: projectID,
		Scope:     "project",
		ScopeID:   "overview",
	})

	blocks := []Block{headerBlock(fmt.Sprintf("Project Overview: %s", projectName))}

	if err != nil {
		blocks = append(blocks, textBlock("No analytics computed yet. Run an indexing job first."))
		return blocks, nil
	}

	if analytics.Summary != nil {
		blocks = append(blocks, textBlock(*analytics.Summary))
	}

	// Add stats
	stats, err := s.GetProjectSymbolStats(ctx, projectID)
	if err == nil {
		data, _ := json.Marshal(stats)
		blocks = append(blocks, Block{Type: "table", Data: data})
	}

	// Add language breakdown
	langs, err := s.GetSymbolCountsByLanguage(ctx, projectID)
	if err == nil && len(langs) > 0 {
		headers := []string{"Language", "Count"}
		var rows [][]string
		for _, l := range langs {
			rows = append(rows, []string{l.Language, fmt.Sprintf("%d", l.Cnt)})
		}
		blocks = append(blocks, tableBlock(headers, rows))
	}

	return blocks, nil
}

// executeSubgraph extracts a connected module around a topic.
func executeSubgraph(ctx context.Context, s *store.Store, projectSlug string, params map[string]any) ([]Block, []SymbolItem, error) {
	topic := stringParam(params, "topic")
	kinds := stringSliceParam(params, "kinds")
	maxDepth := intParam(params, "max_depth", 2)

	// Seed discovery via search
	var seeds []postgres.Symbol
	if topic != "" {
		results, err := s.SearchSymbols(ctx, postgres.SearchSymbolsParams{
			ProjectSlug: projectSlug,
			Query:       &topic,
			Kinds:       kinds,
			Languages:   []string{},
			Lim:         5,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("seed search: %w", err)
		}
		seeds = results
	}

	if len(seeds) == 0 {
		return []Block{
			headerBlock("Subgraph Extraction"),
			textBlock(fmt.Sprintf("No symbols found for topic '%s'. Try a different search term.", topic)),
		}, nil, nil
	}

	// BFS expansion
	visited := make(map[uuid.UUID]bool)
	var subgraph []postgres.Symbol
	type bfsEntry struct {
		id    uuid.UUID
		depth int
	}

	maxNodes := 30
	queue := make([]bfsEntry, 0, len(seeds))
	for _, seed := range seeds {
		if !visited[seed.ID] {
			visited[seed.ID] = true
			subgraph = append(subgraph, seed)
			queue = append(queue, bfsEntry{id: seed.ID, depth: 0})
		}
	}

	for len(queue) > 0 && len(subgraph) < maxNodes {
		entry := queue[0]
		queue = queue[1:]
		if entry.depth >= maxDepth {
			continue
		}

		outEdges, err := s.GetOutgoingEdges(ctx, entry.id)
		if err != nil {
			continue
		}
		for _, edge := range outEdges {
			if visited[edge.TargetID] || len(subgraph) >= maxNodes {
				continue
			}
			sym, err := s.GetSymbol(ctx, edge.TargetID)
			if err != nil {
				continue
			}
			visited[sym.ID] = true
			subgraph = append(subgraph, sym)
			queue = append(queue, bfsEntry{id: sym.ID, depth: entry.depth + 1})
		}

		inEdges, err := s.GetIncomingEdges(ctx, entry.id)
		if err != nil {
			continue
		}
		for _, edge := range inEdges {
			if visited[edge.SourceID] || len(subgraph) >= maxNodes {
				continue
			}
			sym, err := s.GetSymbol(ctx, edge.SourceID)
			if err != nil {
				continue
			}
			visited[sym.ID] = true
			subgraph = append(subgraph, sym)
			queue = append(queue, bfsEntry{id: sym.ID, depth: entry.depth + 1})
		}
	}

	// Collect internal edges
	var gNodes []GraphNode
	var gEdges []GraphEdge
	for _, sym := range subgraph {
		gNodes = append(gNodes, GraphNode{
			ID:   sym.ID.String(),
			Name: sym.Name,
			Kind: sym.Kind,
		})
		outEdges, _ := s.GetOutgoingEdges(ctx, sym.ID)
		for _, e := range outEdges {
			if visited[e.TargetID] {
				gEdges = append(gEdges, GraphEdge{
					Source:   e.SourceID.String(),
					Target:   e.TargetID.String(),
					EdgeType: e.EdgeType,
				})
			}
		}
	}

	items := symbolsToItems(subgraph)
	blocks := []Block{
		headerBlock(fmt.Sprintf("Subgraph: %s (%d symbols, %d edges)", topic, len(subgraph), len(gEdges))),
		symbolListBlock(items),
		graphBlock(gNodes, gEdges),
	}
	return blocks, items, nil
}

// executeRelationships finds FK/join relationships between tables.
func executeRelationships(ctx context.Context, s *store.Store, projectSlug string, params map[string]any) ([]Block, []SymbolItem, error) {
	topic := stringParam(params, "topic")
	kinds := []string{"table"}

	// Use subgraph extraction with table kind
	return executeSubgraph(ctx, s, projectSlug, map[string]any{
		"topic":     topic,
		"kinds":     kinds,
		"max_depth": 1,
	})
}

// executeLineage traces data flow via Neo4j.
func executeLineage(ctx context.Context, s *store.Store, graphClient *graph.Client, projectSlug string, params map[string]any) ([]Block, []SymbolItem, error) {
	symbolName := stringParam(params, "symbol_name")
	direction := stringParam(params, "direction")
	if direction == "" {
		direction = "both"
	}

	if graphClient == nil {
		return []Block{
			headerBlock("Lineage"),
			textBlock("Lineage queries require Neo4j, which is not configured."),
		}, nil, nil
	}

	// Find the symbol first
	results, err := s.SearchSymbols(ctx, postgres.SearchSymbolsParams{
		ProjectSlug: projectSlug,
		Query:       &symbolName,
		Kinds:       []string{},
		Languages:   []string{},
		Lim:         1,
	})
	if err != nil || len(results) == 0 {
		return []Block{
			headerBlock("Lineage"),
			textBlock(fmt.Sprintf("Symbol '%s' not found.", symbolName)),
		}, nil, nil
	}

	sym := results[0]
	lineageResult, err := graphClient.Lineage(ctx, sym.ID, direction, 3)
	if err != nil {
		return nil, nil, fmt.Errorf("lineage query: %w", err)
	}

	// Convert to graph block
	var gNodes []GraphNode
	for _, n := range lineageResult.Nodes {
		gNodes = append(gNodes, GraphNode{
			ID:   n.ID,
			Name: n.Name,
			Kind: n.Kind,
		})
	}
	var gEdges []GraphEdge
	for _, e := range lineageResult.Edges {
		gEdges = append(gEdges, GraphEdge{
			Source:   e.SourceID,
			Target:   e.TargetID,
			EdgeType: e.EdgeType,
		})
	}

	blocks := []Block{
		headerBlock(fmt.Sprintf("Lineage for %s (%s)", sym.Name, direction)),
		graphBlock(gNodes, gEdges),
	}

	items := symbolsToItems(results)
	return blocks, items, nil
}

// executeImpact analyzes what breaks if a symbol changes.
func executeImpact(ctx context.Context, s *store.Store, impactEngine *impact.Engine, projectSlug string, params map[string]any) ([]Block, []SymbolItem, error) {
	symbolName := stringParam(params, "symbol_name")
	changeType := stringParam(params, "change_type")
	if changeType == "" {
		changeType = "modify"
	}

	if impactEngine == nil {
		return []Block{
			headerBlock("Impact Analysis"),
			textBlock("Impact analysis requires Neo4j, which is not configured."),
		}, nil, nil
	}

	// Find the symbol
	results, err := s.SearchSymbols(ctx, postgres.SearchSymbolsParams{
		ProjectSlug: projectSlug,
		Query:       &symbolName,
		Kinds:       []string{},
		Languages:   []string{},
		Lim:         1,
	})
	if err != nil || len(results) == 0 {
		return []Block{
			headerBlock("Impact Analysis"),
			textBlock(fmt.Sprintf("Symbol '%s' not found.", symbolName)),
		}, nil, nil
	}

	sym := results[0]
	result, err := impactEngine.Analyze(ctx, sym.ID, changeType, 5)
	if err != nil {
		return nil, nil, fmt.Errorf("impact analysis: %w", err)
	}

	blocks := []Block{
		headerBlock(fmt.Sprintf("Impact of %s on %s", changeType, sym.Name)),
		textBlock(fmt.Sprintf("**Total affected:** %d symbols", result.TotalAffected)),
	}

	if len(result.DirectImpact) > 0 {
		headers := []string{"Symbol", "Kind", "Severity", "Via"}
		var rows [][]string
		for _, n := range result.DirectImpact {
			rows = append(rows, []string{n.Symbol.Name, n.Symbol.Kind, n.Severity, n.EdgeType})
		}
		blocks = append(blocks, tableBlock(headers, rows))
	}

	if len(result.TransitiveImpact) > 0 {
		blocks = append(blocks, textBlock(fmt.Sprintf("**Transitive impact:** %d additional symbols affected", len(result.TransitiveImpact))))
	}

	items := symbolsToItems(results)
	return blocks, items, nil
}

// Param helpers

func stringParam(params map[string]any, key string) string {
	if v, ok := params[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func stringSliceParam(params map[string]any, key string) []string {
	if v, ok := params[key]; ok {
		if arr, ok := v.([]any); ok {
			result := make([]string, 0, len(arr))
			for _, item := range arr {
				if s, ok := item.(string); ok {
					result = append(result, s)
				}
			}
			return result
		}
	}
	return []string{}
}

func intParam(params map[string]any, key string, defaultVal int) int {
	if v, ok := params[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return defaultVal
}

func symbolsToItems(syms []postgres.Symbol) []SymbolItem {
	items := make([]SymbolItem, len(syms))
	for i, sym := range syms {
		item := SymbolItem{
			ID:            sym.ID.String(),
			Name:          sym.Name,
			QualifiedName: sym.QualifiedName,
			Kind:          sym.Kind,
			Language:      sym.Language,
			Signature:     sym.Signature,
		}
		// Extract metrics from metadata
		if len(sym.Metadata) > 0 {
			var meta map[string]any
			if err := json.Unmarshal(sym.Metadata, &meta); err == nil {
				if pr, ok := meta["pagerank"].(float64); ok {
					item.PageRank = pr
				}
				if id, ok := meta["in_degree"].(float64); ok {
					item.InDegree = int32(id)
				}
			}
		}
		items[i] = item
	}
	return items
}
