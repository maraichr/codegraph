package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"

	"github.com/google/uuid"

	"github.com/codegraph-labs/codegraph/internal/mcp"
	"github.com/codegraph-labs/codegraph/internal/mcp/session"
	"github.com/codegraph-labs/codegraph/internal/store"
	"github.com/codegraph-labs/codegraph/internal/store/postgres"
)

// ExtractSubgraphParams are the parameters for the extract_subgraph tool.
type ExtractSubgraphParams struct {
	Project           string   `json:"project"`
	Topic             string   `json:"topic,omitempty"`
	SeedSymbols       []string `json:"seed_symbols,omitempty"`
	MaxDepth          int      `json:"max_depth,omitempty"`
	MaxNodes          int      `json:"max_nodes,omitempty"`
	CrossBoundary     bool     `json:"cross_boundary,omitempty"`
	Verbosity         string   `json:"verbosity,omitempty"`
	MaxResponseTokens int      `json:"max_response_tokens,omitempty"`
	SessionID         string   `json:"session_id,omitempty"`
	DryRun            bool     `json:"dry_run,omitempty"`
}

// ExtractSubgraphHandler implements the extract_subgraph MCP tool.
type ExtractSubgraphHandler struct {
	store   *store.Store
	session *session.Manager
	logger  *slog.Logger
}

// NewExtractSubgraphHandler creates a new handler.
func NewExtractSubgraphHandler(s *store.Store, sm *session.Manager, logger *slog.Logger) *ExtractSubgraphHandler {
	return &ExtractSubgraphHandler{store: s, session: sm, logger: logger}
}

// Handle executes the subgraph extraction: seed discovery → BFS → boundary → trim → format.
func (h *ExtractSubgraphHandler) Handle(ctx context.Context, params ExtractSubgraphParams) (string, error) {
	// Apply defaults
	if params.MaxDepth <= 0 {
		params.MaxDepth = 2
	}
	if params.MaxNodes <= 0 {
		params.MaxNodes = 50
	}
	if params.MaxResponseTokens <= 0 {
		params.MaxResponseTokens = 4000
	}

	verbosity := mcp.ParseVerbosity(params.Verbosity)

	// Load session
	var sess *session.Session
	if h.session != nil && params.SessionID != "" {
		var err error
		sess, err = h.session.Load(ctx, params.SessionID)
		if err != nil {
			h.logger.Warn("failed to load session", slog.String("error", err.Error()))
		}
	}

	// 1. Seed discovery
	seeds, err := h.discoverSeeds(ctx, params)
	if err != nil {
		return "", fmt.Errorf("seed discovery: %w", err)
	}

	if len(seeds) == 0 {
		return "No symbols found matching the topic. Try a different search term or provide seed_symbols.", nil
	}

	// 2. BFS expansion
	subgraph := h.expandBFS(ctx, seeds, params.MaxDepth, params.MaxNodes)

	// 3. Collect edges within the subgraph
	edges := h.collectEdges(ctx, subgraph)

	// Dry run: return counts only
	if params.DryRun {
		return mcp.FormatDryRun(mcp.DryRunResult{
			SymbolCount:     len(subgraph),
			EdgeCount:       len(edges),
			EstimatedTokens: estimateSubgraphTokens(subgraph, edges, verbosity),
			DepthReached:    params.MaxDepth,
		}), nil
	}

	// 4. Token-aware trimming
	subgraph = h.trimToTokenBudget(subgraph, params.MaxResponseTokens, verbosity)

	// 5. Format response
	rb := mcp.NewResponseBuilder(params.MaxResponseTokens)
	rb.AddHeader(fmt.Sprintf("**Subgraph: %s** (%d symbols, %d edges)", params.Topic, len(subgraph), len(edges)))

	// Identify core symbols (reached from multiple seeds)
	coreIDs := identifyCore(seeds, subgraph)

	// Add symbol cards
	returned := 0
	for _, sym := range subgraph {
		isCore := coreIDs[sym.ID]
		if sess != nil && sess.IsSeen(sym.ID) && !isCore {
			if !rb.AddSymbolStub(sym) {
				break
			}
		} else {
			if !rb.AddSymbolCard(sym, verbosity, sess) {
				break
			}
		}
		returned++
	}

	// Add edge summary
	if len(edges) > 0 {
		edgeSummary := formatEdgeSummary(edges, subgraph)
		rb.AddSection("Relationships", edgeSummary)
	}

	// Update session
	if sess != nil {
		for _, sym := range subgraph[:returned] {
			sess.MarkSeen(sym.ID)
		}
		if params.Topic != "" {
			sess.AddQuery("extract_subgraph: " + params.Topic)
			sess.AddRecap(fmt.Sprintf("Extracted subgraph '%s': %d symbols, %d edges", params.Topic, len(subgraph), len(edges)))
		}
		if h.session != nil {
			_ = h.session.Save(ctx, sess)
		}
	}

	// Navigation hints
	nav := mcp.NewNavigator(h.store.Queries)
	hints := nav.SuggestNextSteps("extract_subgraph", symbolsFromSubgraph(subgraph), sess)

	return rb.FinalizeWithHints(len(subgraph), returned, hints), nil
}

func (h *ExtractSubgraphHandler) discoverSeeds(ctx context.Context, params ExtractSubgraphParams) ([]postgres.Symbol, error) {
	var seeds []postgres.Symbol

	// Use explicit seed symbols if provided
	if len(params.SeedSymbols) > 0 {
		for _, idStr := range params.SeedSymbols {
			id, err := uuid.Parse(idStr)
			if err != nil {
				continue
			}
			sym, err := h.store.GetSymbol(ctx, id)
			if err != nil {
				continue
			}
			seeds = append(seeds, sym)
		}
		return seeds, nil
	}

	// Fall back to text search for the topic
	if params.Topic != "" {
		project, err := h.store.GetProject(ctx, params.Project)
		if err != nil {
			return nil, fmt.Errorf("get project: %w", err)
		}

		topic := params.Topic
		results, err := h.store.SearchSymbols(ctx, postgres.SearchSymbolsParams{
			ProjectSlug: project.Slug,
			Query:       &topic,
			Kinds:       []string{},
			Languages:   []string{},
			Lim:         5,
		})
		if err != nil {
			return nil, fmt.Errorf("search symbols: %w", err)
		}
		seeds = results
	}

	return seeds, nil
}

func (h *ExtractSubgraphHandler) expandBFS(ctx context.Context, seeds []postgres.Symbol, maxDepth, maxNodes int) []postgres.Symbol {
	visited := make(map[uuid.UUID]bool)
	var result []postgres.Symbol

	// Seed the BFS
	queue := make([]bfsEntry, 0, len(seeds))
	for _, s := range seeds {
		if !visited[s.ID] {
			visited[s.ID] = true
			result = append(result, s)
			queue = append(queue, bfsEntry{id: s.ID, depth: 0})
		}
	}

	// BFS expansion
	for len(queue) > 0 && len(result) < maxNodes {
		entry := queue[0]
		queue = queue[1:]

		if entry.depth >= maxDepth {
			continue
		}

		// Get outgoing edges
		outEdges, err := h.store.GetOutgoingEdges(ctx, entry.id)
		if err != nil {
			continue
		}
		for _, edge := range outEdges {
			if visited[edge.TargetID] || len(result) >= maxNodes {
				continue
			}
			sym, err := h.store.GetSymbol(ctx, edge.TargetID)
			if err != nil {
				continue
			}

			// Boundary detection: skip low-PageRank symbols at deeper levels
			if entry.depth > 0 && isLowValue(sym) {
				continue
			}

			visited[sym.ID] = true
			result = append(result, sym)
			queue = append(queue, bfsEntry{id: sym.ID, depth: entry.depth + 1})
		}

		// Get incoming edges
		inEdges, err := h.store.GetIncomingEdges(ctx, entry.id)
		if err != nil {
			continue
		}
		for _, edge := range inEdges {
			if visited[edge.SourceID] || len(result) >= maxNodes {
				continue
			}
			sym, err := h.store.GetSymbol(ctx, edge.SourceID)
			if err != nil {
				continue
			}

			if entry.depth > 0 && isLowValue(sym) {
				continue
			}

			visited[sym.ID] = true
			result = append(result, sym)
			queue = append(queue, bfsEntry{id: sym.ID, depth: entry.depth + 1})
		}
	}

	return result
}

func (h *ExtractSubgraphHandler) collectEdges(ctx context.Context, symbols []postgres.Symbol) []subgraphEdge {
	symbolSet := make(map[uuid.UUID]bool)
	for _, s := range symbols {
		symbolSet[s.ID] = true
	}

	var edges []subgraphEdge
	seen := make(map[string]bool)

	for _, sym := range symbols {
		outEdges, err := h.store.GetOutgoingEdges(ctx, sym.ID)
		if err != nil {
			continue
		}
		for _, e := range outEdges {
			if !symbolSet[e.TargetID] {
				continue
			}
			key := fmt.Sprintf("%s-%s-%s", e.SourceID, e.TargetID, e.EdgeType)
			if seen[key] {
				continue
			}
			seen[key] = true
			edges = append(edges, subgraphEdge{
				SourceID: e.SourceID,
				TargetID: e.TargetID,
				EdgeType: e.EdgeType,
			})
		}
	}

	return edges
}

func (h *ExtractSubgraphHandler) trimToTokenBudget(symbols []postgres.Symbol, maxTokens int, verbosity mcp.Verbosity) []postgres.Symbol {
	// Sort by PageRank descending to keep highest-value symbols
	sort.Slice(symbols, func(i, j int) bool {
		return getPageRank(symbols[i]) > getPageRank(symbols[j])
	})

	estimated := 0
	tokensPerSymbol := symbolTokenEstimate(verbosity)
	var result []postgres.Symbol

	for _, sym := range symbols {
		cost := tokensPerSymbol
		if sym.Signature != nil {
			cost += len(*sym.Signature) / 4
		}
		if estimated+cost > maxTokens {
			break
		}
		estimated += cost
		result = append(result, sym)
	}

	return result
}

type bfsEntry struct {
	id    uuid.UUID
	depth int
}

type subgraphEdge struct {
	SourceID uuid.UUID
	TargetID uuid.UUID
	EdgeType string
}

func isLowValue(sym postgres.Symbol) bool {
	pr := getPageRank(sym)
	return pr < 0.0001 && sym.Kind == "column"
}

func getPageRank(sym postgres.Symbol) float64 {
	if len(sym.Metadata) == 0 {
		return 0
	}
	var meta map[string]any
	if err := json.Unmarshal(sym.Metadata, &meta); err != nil {
		return 0
	}
	if pr, ok := meta["pagerank"].(float64); ok {
		return pr
	}
	return 0
}

func symbolTokenEstimate(verbosity mcp.Verbosity) int {
	switch verbosity {
	case mcp.VerbositySummary:
		return 30
	case mcp.VerbosityFull:
		return 120
	default:
		return 60
	}
}

func estimateSubgraphTokens(symbols []postgres.Symbol, edges []subgraphEdge, verbosity mcp.Verbosity) int {
	perSym := symbolTokenEstimate(verbosity)
	return len(symbols)*perSym + len(edges)*15 + 100 // header + edge summary
}

func identifyCore(seeds []postgres.Symbol, subgraph []postgres.Symbol) map[uuid.UUID]bool {
	core := make(map[uuid.UUID]bool)
	for _, seed := range seeds {
		core[seed.ID] = true
	}
	return core
}

func formatEdgeSummary(edges []subgraphEdge, symbols []postgres.Symbol) string {
	nameMap := make(map[uuid.UUID]string)
	for _, s := range symbols {
		nameMap[s.ID] = s.Name
	}

	// Group by edge type
	byType := make(map[string]int)
	for _, e := range edges {
		byType[e.EdgeType]++
	}

	var summary string
	for edgeType, count := range byType {
		summary += fmt.Sprintf("- %s: %d edges\n", edgeType, count)
	}

	// Show first few edges as examples
	if len(edges) > 0 {
		summary += "\nKey connections:\n"
		shown := 0
		for _, e := range edges {
			if shown >= 10 {
				summary += fmt.Sprintf("  ... and %d more\n", len(edges)-10)
				break
			}
			src := nameMap[e.SourceID]
			tgt := nameMap[e.TargetID]
			if src == "" {
				src = e.SourceID.String()[:8]
			}
			if tgt == "" {
				tgt = e.TargetID.String()[:8]
			}
			summary += fmt.Sprintf("  %s -[%s]-> %s\n", src, e.EdgeType, tgt)
			shown++
		}
	}

	return summary
}

func symbolsFromSubgraph(symbols []postgres.Symbol) []postgres.Symbol {
	return symbols
}
