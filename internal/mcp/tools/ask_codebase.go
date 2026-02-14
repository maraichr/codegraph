package tools

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/codegraph-labs/codegraph/internal/mcp"
	"github.com/codegraph-labs/codegraph/internal/mcp/session"
	"github.com/codegraph-labs/codegraph/internal/store"
	"github.com/codegraph-labs/codegraph/internal/store/postgres"
)

// AskCodebaseParams are the parameters for the ask_codebase meta-tool.
type AskCodebaseParams struct {
	Project           string `json:"project"`
	Question          string `json:"question"`
	MaxResponseTokens int    `json:"max_response_tokens,omitempty"`
	SessionID         string `json:"session_id,omitempty"`
	Verbosity         string `json:"verbosity,omitempty"`
}

// AskCodebaseHandler routes natural language questions to appropriate tool chains.
type AskCodebaseHandler struct {
	store    *store.Store
	session  *session.Manager
	subgraph *ExtractSubgraphHandler
	logger   *slog.Logger
}

// NewAskCodebaseHandler creates a new intent router handler.
func NewAskCodebaseHandler(s *store.Store, sm *session.Manager, logger *slog.Logger) *AskCodebaseHandler {
	return &AskCodebaseHandler{
		store:    s,
		session:  sm,
		subgraph: NewExtractSubgraphHandler(s, sm, logger),
		logger:   logger,
	}
}

// Intent represents a classified question intent.
type Intent string

const (
	IntentSearch    Intent = "search"
	IntentImpact    Intent = "impact"
	IntentLineage   Intent = "lineage"
	IntentOverview  Intent = "overview"
	IntentSubgraph  Intent = "subgraph"
	IntentDeps      Intent = "dependencies"
)

// Handle classifies the question intent and routes to the appropriate tool chain.
func (h *AskCodebaseHandler) Handle(ctx context.Context, params AskCodebaseParams) (string, error) {
	if params.MaxResponseTokens <= 0 {
		params.MaxResponseTokens = 4000
	}

	intent := classifyIntent(params.Question)
	h.logger.Info("classified intent",
		slog.String("question", params.Question),
		slog.String("intent", string(intent)))

	switch intent {
	case IntentOverview:
		return h.handleOverview(ctx, params)
	case IntentImpact:
		return h.handleImpact(ctx, params)
	case IntentLineage:
		return h.handleLineage(ctx, params)
	case IntentSubgraph:
		return h.handleSubgraph(ctx, params)
	case IntentDeps:
		return h.handleDependencies(ctx, params)
	default:
		return h.handleSearch(ctx, params)
	}
}

func classifyIntent(question string) Intent {
	q := strings.ToLower(question)

	// Impact patterns
	impactPatterns := []string{
		"what breaks", "what happens if", "impact", "blast radius",
		"change", "rename", "delete", "remove", "modify", "affected",
	}
	for _, p := range impactPatterns {
		if strings.Contains(q, p) {
			return IntentImpact
		}
	}

	// Lineage patterns
	lineagePatterns := []string{
		"data flow", "lineage", "where does", "data come from",
		"written to", "read from", "transforms", "populates",
	}
	for _, p := range lineagePatterns {
		if strings.Contains(q, p) {
			return IntentLineage
		}
	}

	// Overview patterns
	overviewPatterns := []string{
		"overview", "what is this", "describe", "summary",
		"architecture", "structure", "languages", "how big",
	}
	for _, p := range overviewPatterns {
		if strings.Contains(q, p) {
			return IntentOverview
		}
	}

	// Dependency patterns
	depPatterns := []string{
		"depends on", "dependency", "dependencies", "uses",
		"calls", "imports", "references",
	}
	for _, p := range depPatterns {
		if strings.Contains(q, p) {
			return IntentDeps
		}
	}

	// Subgraph patterns (topic exploration)
	subgraphPatterns := []string{
		"everything about", "all related", "module", "system",
		"pipeline", "workflow", "process",
	}
	for _, p := range subgraphPatterns {
		if strings.Contains(q, p) {
			return IntentSubgraph
		}
	}

	return IntentSearch
}

func (h *AskCodebaseHandler) handleOverview(ctx context.Context, params AskCodebaseParams) (string, error) {
	project, err := h.store.GetProject(ctx, params.Project)
	if err != nil {
		return "", fmt.Errorf("get project: %w", err)
	}

	analytics, err := h.store.GetProjectAnalytics(ctx, postgres.GetProjectAnalyticsParams{
		ProjectID: project.ID,
		Scope:     "project",
		ScopeID:   "overview",
	})
	if err != nil {
		return fmt.Sprintf("Project '%s' found but no analytics computed yet. Run an indexing job first.", params.Project), nil
	}

	rb := mcp.NewResponseBuilder(params.MaxResponseTokens)
	rb.AddHeader(fmt.Sprintf("**Project Overview: %s**", project.Name))

	if analytics.Summary != nil {
		rb.AddLine(*analytics.Summary)
		rb.AddLine("")
	}

	// Add layer distribution if available
	layers, err := h.store.GetProjectAnalytics(ctx, postgres.GetProjectAnalyticsParams{
		ProjectID: project.ID,
		Scope:     "project",
		ScopeID:   "layers",
	})
	if err == nil && layers.Summary != nil {
		rb.AddLine(*layers.Summary)
	}

	// Add bridge info
	bridges, err := h.store.ListProjectAnalyticsByScope(ctx, postgres.ListProjectAnalyticsByScopeParams{
		ProjectID: project.ID,
		Scope:     "bridge",
	})
	if err == nil && len(bridges) > 0 {
		rb.AddLine("")
		rb.AddLine("**Cross-language bridges:**")
		for _, b := range bridges {
			if b.Summary != nil {
				rb.AddLine(fmt.Sprintf("- %s", *b.Summary))
			}
		}
	}

	nav := mcp.NewNavigator(h.store.Queries)
	hints := nav.SuggestNextSteps("list_project_overview", nil, nil)
	return rb.FinalizeWithHints(1, 1, hints), nil
}

func (h *AskCodebaseHandler) handleSearch(ctx context.Context, params AskCodebaseParams) (string, error) {
	project, err := h.store.GetProject(ctx, params.Project)
	if err != nil {
		return "", fmt.Errorf("get project: %w", err)
	}

	searchTerms := extractSearchTerms(params.Question)
	results, err := h.store.SearchSymbols(ctx, postgres.SearchSymbolsParams{
		ProjectSlug: project.Slug,
		Query:       &searchTerms,
		Kinds:       []string{},
		Languages:   []string{},
		Lim:         20,
	})
	if err != nil {
		return "", fmt.Errorf("search symbols: %w", err)
	}

	if len(results) == 0 {
		return fmt.Sprintf("No symbols found matching '%s'.", params.Question), nil
	}

	// Load session for ranking
	var sess *session.Session
	if h.session != nil && params.SessionID != "" {
		sess, _ = h.session.Load(ctx, params.SessionID)
	}

	verbosity := mcp.ParseVerbosity(params.Verbosity)
	ranked := mcp.RankSymbols(results, extractSearchTerms(params.Question), mcp.DefaultRankConfig(), sess)

	rb := mcp.NewResponseBuilder(params.MaxResponseTokens)
	rb.AddHeader(fmt.Sprintf("**Search results for: %s**", params.Question))

	returned := 0
	for _, r := range ranked {
		if !rb.AddSymbolCard(r.Symbol, verbosity, sess) {
			break
		}
		returned++
	}

	nav := mcp.NewNavigator(h.store.Queries)
	symbols := make([]postgres.Symbol, 0, len(ranked))
	for _, r := range ranked {
		symbols = append(symbols, r.Symbol)
	}
	hints := nav.SuggestNextSteps("search_symbols", symbols, sess)

	return rb.FinalizeWithHints(len(results), returned, hints), nil
}

func (h *AskCodebaseHandler) handleImpact(ctx context.Context, params AskCodebaseParams) (string, error) {
	// Extract the target symbol name from the question and search for it
	return h.handleSearch(ctx, params)
}

func (h *AskCodebaseHandler) handleLineage(ctx context.Context, params AskCodebaseParams) (string, error) {
	return h.handleSearch(ctx, params)
}

func (h *AskCodebaseHandler) handleSubgraph(ctx context.Context, params AskCodebaseParams) (string, error) {
	return h.subgraph.Handle(ctx, ExtractSubgraphParams{
		Project:           params.Project,
		Topic:             extractSearchTerms(params.Question),
		MaxDepth:          2,
		MaxNodes:          30,
		MaxResponseTokens: params.MaxResponseTokens,
		SessionID:         params.SessionID,
		Verbosity:         params.Verbosity,
	})
}

func (h *AskCodebaseHandler) handleDependencies(ctx context.Context, params AskCodebaseParams) (string, error) {
	return h.handleSearch(ctx, params)
}

// extractSearchTerms removes common question words to get the core search terms.
func extractSearchTerms(question string) string {
	stopWords := map[string]bool{
		"what": true, "where": true, "how": true, "does": true, "is": true,
		"the": true, "a": true, "an": true, "are": true, "can": true,
		"do": true, "if": true, "i": true, "to": true, "of": true,
		"in": true, "for": true, "it": true, "this": true, "that": true,
		"about": true, "show": true, "me": true, "find": true, "get": true,
		"tell": true, "breaks": true, "happens": true, "everything": true,
	}

	words := strings.Fields(strings.ToLower(question))
	var terms []string
	for _, w := range words {
		w = strings.Trim(w, "?.,!\"'")
		if !stopWords[w] && len(w) > 1 {
			terms = append(terms, w)
		}
	}

	if len(terms) == 0 {
		return question
	}
	return strings.Join(terms, " ")
}
