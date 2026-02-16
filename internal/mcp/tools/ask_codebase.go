package tools

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/maraichr/lattice/internal/auth"
	"github.com/maraichr/lattice/internal/embedding"
	"github.com/maraichr/lattice/internal/mcp"
	"github.com/maraichr/lattice/internal/mcp/session"
	"github.com/maraichr/lattice/internal/store"
	"github.com/maraichr/lattice/internal/store/postgres"
)

// AskCodebaseParams are the parameters for the ask_codebase meta-tool.
type AskCodebaseParams struct {
	Project           string   `json:"project"`
	Question          string   `json:"question"`
	Kinds             []string `json:"kinds,omitempty"`
	Languages         []string `json:"languages,omitempty"`
	MaxResponseTokens int      `json:"max_response_tokens,omitempty"`
	SessionID         string   `json:"session_id,omitempty"`
	Verbosity         string   `json:"verbosity,omitempty"`
}

// AskCodebaseHandler routes natural language questions to appropriate tool chains.
type AskCodebaseHandler struct {
	store    *store.Store
	session  *session.Manager
	subgraph *ExtractSubgraphHandler
	impact   *AnalyzeImpactHandler
	lineage  *GetLineageHandler
	trace    *TraceCrossLanguageHandler
	logger   *slog.Logger
}

// NewAskCodebaseHandler creates a new intent router handler.
func NewAskCodebaseHandler(s *store.Store, sm *session.Manager, embedder embedding.Embedder, logger *slog.Logger) *AskCodebaseHandler {
	return &AskCodebaseHandler{
		store:    s,
		session:  sm,
		subgraph: NewExtractSubgraphHandler(s, sm, embedder, logger),
		impact:   NewAnalyzeImpactHandler(s, logger),
		lineage:  NewGetLineageHandler(s, logger),
		trace:    NewTraceCrossLanguageHandler(s, logger),
		logger:   logger,
	}
}

// Intent represents a classified question intent.
type Intent string

const (
	IntentSearch        Intent = "search"
	IntentImpact        Intent = "impact"
	IntentLineage       Intent = "lineage"
	IntentOverview      Intent = "overview"
	IntentSubgraph      Intent = "subgraph"
	IntentDeps          Intent = "dependencies"
	IntentRanking       Intent = "ranking"
	IntentRelationships Intent = "relationships"
	IntentBridges       Intent = "bridges"
	IntentAnalytics     Intent = "analytics"
	IntentCrossLanguage Intent = "cross_language"
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
	case IntentRanking:
		return h.handleRanking(ctx, params)
	case IntentImpact:
		return h.handleImpact(ctx, params)
	case IntentLineage:
		return h.handleLineage(ctx, params)
	case IntentSubgraph:
		return h.handleSubgraph(ctx, params)
	case IntentDeps:
		return h.handleDependencies(ctx, params)
	case IntentRelationships:
		return h.handleRelationships(ctx, params)
	case IntentBridges:
		return h.handleBridges(ctx, params)
	case IntentAnalytics:
		return h.handleAnalytics(ctx, params)
	case IntentCrossLanguage:
		return h.handleCrossLanguage(ctx, params)
	default:
		return h.handleSearch(ctx, params)
	}
}

func classifyIntent(question string) Intent {
	q := strings.ToLower(question)

	// Ranking patterns (check early — "most used", "top", "busiest", "most important")
	rankingPatterns := []string{
		"most used", "most important", "most referenced", "most connected",
		"top ", "busiest", "highest", "largest", "most common",
		"most frequent", "most popular", "heavily used",
	}
	for _, p := range rankingPatterns {
		if strings.Contains(q, p) {
			return IntentRanking
		}
	}

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

	// Cross-language trace patterns (check before bridges)
	crossLangPatterns := []string{
		"what tables does", "tables does this endpoint",
		"full stack", "stack trace", "stack slice", "end to end",
		"calls this stored proc", "calls this procedure",
		"from app code", "from the frontend", "from the api",
		"what touches", "who calls",
		"cross-language trace", "cross language trace",
	}
	for _, p := range crossLangPatterns {
		if strings.Contains(q, p) {
			return IntentCrossLanguage
		}
	}

	// Bridge patterns (cross-language)
	bridgePatterns := []string{
		"cross-language", "bridge", "bridges", "between languages",
		"polyglot", "multi-language",
	}
	for _, p := range bridgePatterns {
		if strings.Contains(q, p) {
			return IntentBridges
		}
	}

	// Analytics patterns
	analyticsPatterns := []string{
		"statistics", "stats", "distribution", "breakdown",
		"how many", "count", "metrics", "layer", "layers",
	}
	for _, p := range analyticsPatterns {
		if strings.Contains(q, p) {
			return IntentAnalytics
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

	// Relationship / FK patterns
	relPatterns := []string{
		"foreign key", "foreign keys", "relationship", "relationships",
		"related to", "joins", "references between", "missing fk",
		"data access pattern",
	}
	for _, p := range relPatterns {
		if strings.Contains(q, p) {
			return IntentRelationships
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
		return "", WrapProjectError(err)
	}
	if p, ok := auth.PrincipalFrom(ctx); ok && !p.IsAdmin() && project.TenantID != p.TenantID {
		return "", fmt.Errorf("access denied to project %s", params.Project)
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

func (h *AskCodebaseHandler) handleRanking(ctx context.Context, params AskCodebaseParams) (string, error) {
	project, err := h.store.GetProject(ctx, params.Project)
	if err != nil {
		return "", WrapProjectError(err)
	}
	if p, ok := auth.PrincipalFrom(ctx); ok && !p.IsAdmin() && project.TenantID != p.TenantID {
		return "", fmt.Errorf("access denied to project %s", params.Project)
	}

	// Extract kinds from the question if not explicitly provided
	kinds := params.Kinds
	if len(kinds) == 0 {
		kinds = extractKindsFromQuestion(params.Question)
	}

	results, err := h.store.ListTopSymbolsByKind(ctx, postgres.ListTopSymbolsByKindParams{
		ProjectSlug: project.Slug,
		Kinds:       kinds,
		Languages:   params.Languages,
		Lim:         10,
	})
	if err != nil {
		return "", fmt.Errorf("list top symbols: %w", err)
	}

	if len(results) == 0 {
		return fmt.Sprintf("No symbols found matching the criteria (kinds=%v).", kinds), nil
	}

	verbosity := mcp.ParseVerbosity(params.Verbosity)
	rb := mcp.NewResponseBuilder(params.MaxResponseTokens)

	kindLabel := "symbols"
	if len(kinds) > 0 {
		kindLabel = strings.Join(kinds, "/") + "s"
	}
	rb.AddHeader(fmt.Sprintf("**Top %s by usage (in-degree)**", kindLabel))

	var sess *session.Session
	if h.session != nil && params.SessionID != "" {
		sess, _ = h.session.Load(ctx, params.SessionID)
	}

	returned := 0
	for _, sym := range results {
		if !rb.AddSymbolCard(sym, verbosity, sess) {
			break
		}
		returned++
	}

	nav := mcp.NewNavigator(h.store.Queries)
	hints := nav.SuggestNextSteps("search_symbols", results, sess)
	return rb.FinalizeWithHints(len(results), returned, hints), nil
}

func (h *AskCodebaseHandler) handleSearch(ctx context.Context, params AskCodebaseParams) (string, error) {
	project, err := h.store.GetProject(ctx, params.Project)
	if err != nil {
		return "", WrapProjectError(err)
	}
	if p, ok := auth.PrincipalFrom(ctx); ok && !p.IsAdmin() && project.TenantID != p.TenantID {
		return "", fmt.Errorf("access denied to project %s", params.Project)
	}

	searchTerms := extractSearchTerms(params.Question)
	kinds := params.Kinds
	if kinds == nil {
		kinds = []string{}
	}
	results, err := h.store.SearchSymbols(ctx, postgres.SearchSymbolsParams{
		ProjectSlug: project.Slug,
		Query:       &searchTerms,
		Kinds:       kinds,
		Languages:   params.Languages,
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
	symbolName := extractSearchTerms(params.Question)
	changeType := "modify"
	q := strings.ToLower(params.Question)
	if strings.Contains(q, "delete") || strings.Contains(q, "remove") || strings.Contains(q, "drop") {
		changeType = "delete"
	} else if strings.Contains(q, "rename") {
		changeType = "rename"
	}
	return h.impact.Handle(ctx, AnalyzeImpactParams{
		Project:    params.Project,
		SymbolName: symbolName,
		ChangeType: changeType,
		MaxDepth:   3,
	})
}

func (h *AskCodebaseHandler) handleLineage(ctx context.Context, params AskCodebaseParams) (string, error) {
	symbolName := extractSearchTerms(params.Question)
	direction := "both"
	q := strings.ToLower(params.Question)
	if strings.Contains(q, "come from") || strings.Contains(q, "upstream") || strings.Contains(q, "data source") {
		direction = "upstream"
	} else if strings.Contains(q, "written to") || strings.Contains(q, "downstream") || strings.Contains(q, "populates") {
		direction = "downstream"
	}
	return h.lineage.Handle(ctx, GetLineageParams{
		Project:    params.Project,
		SymbolName: symbolName,
		Direction:  direction,
		MaxDepth:   5,
	})
}

func (h *AskCodebaseHandler) handleCrossLanguage(ctx context.Context, params AskCodebaseParams) (string, error) {
	symbolName := extractSearchTerms(params.Question)
	return h.trace.Handle(ctx, TraceCrossLanguageParams{
		Project:    params.Project,
		SymbolName: symbolName,
		Direction:  "full",
		MaxDepth:   5,
		SessionID:  params.SessionID,
	})
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

func (h *AskCodebaseHandler) handleRelationships(ctx context.Context, params AskCodebaseParams) (string, error) {
	// Use extract_subgraph with kinds=["table"] to get all tables and their edges in one shot
	kinds := params.Kinds
	if len(kinds) == 0 {
		kinds = extractKindsFromQuestion(params.Question)
	}
	if len(kinds) == 0 {
		kinds = []string{"table"}
	}
	return h.subgraph.Handle(ctx, ExtractSubgraphParams{
		Project:           params.Project,
		Kinds:             kinds,
		MaxDepth:          1,
		MaxNodes:          100,
		MaxResponseTokens: params.MaxResponseTokens,
		SessionID:         params.SessionID,
		Verbosity:         "summary",
	})
}

func (h *AskCodebaseHandler) handleDependencies(ctx context.Context, params AskCodebaseParams) (string, error) {
	return h.handleSearch(ctx, params)
}

func (h *AskCodebaseHandler) handleBridges(ctx context.Context, params AskCodebaseParams) (string, error) {
	project, err := h.store.GetProject(ctx, params.Project)
	if err != nil {
		return "", WrapProjectError(err)
	}
	if p, ok := auth.PrincipalFrom(ctx); ok && !p.IsAdmin() && project.TenantID != p.TenantID {
		return "", fmt.Errorf("access denied to project %s", params.Project)
	}

	rows, err := h.store.GetCrossLanguageBridges(ctx, project.ID)
	if err != nil {
		return "", fmt.Errorf("get bridges: %w", err)
	}

	rb := mcp.NewResponseBuilder(params.MaxResponseTokens)
	rb.AddHeader(fmt.Sprintf("**Cross-Language Bridges: %s**", project.Name))

	if len(rows) == 0 {
		rb.AddLine("No cross-language bridges found.")
		return rb.Finalize(0, 0), nil
	}

	for _, r := range rows {
		rb.AddLine(fmt.Sprintf("- **%s → %s** via `%s`: %d edges",
			r.SourceLanguage, r.TargetLanguage, r.EdgeType, r.EdgeCount))
	}

	return rb.Finalize(len(rows), len(rows)), nil
}

func (h *AskCodebaseHandler) handleAnalytics(ctx context.Context, params AskCodebaseParams) (string, error) {
	q := strings.ToLower(params.Question)

	// Determine the best analytics scope from the question
	scope := "summary"
	if strings.Contains(q, "layer") || strings.Contains(q, "layers") {
		scope = "layers"
	} else if strings.Contains(q, "language") || strings.Contains(q, "languages") {
		scope = "languages"
	} else if strings.Contains(q, "kind") || strings.Contains(q, "kinds") || strings.Contains(q, "type") {
		scope = "kinds"
	}

	handler := NewGetProjectAnalyticsHandler(h.store, h.logger)
	return handler.Handle(ctx, GetProjectAnalyticsParams{
		Project: params.Project,
		Scope:   scope,
	})
}

// extractKindsFromQuestion infers symbol kinds from natural language question text.
func extractKindsFromQuestion(question string) []string {
	q := strings.ToLower(question)
	kindMap := map[string]string{
		"table":     "table",
		"tables":    "table",
		"procedure": "procedure",
		"proc":      "procedure",
		"procs":     "procedure",
		"class":     "class",
		"classes":   "class",
		"method":    "method",
		"methods":   "method",
		"function":  "function",
		"functions": "function",
		"column":    "column",
		"columns":   "column",
		"interface": "interface",
		"field":     "field",
		"property":  "property",
		"enum":      "enum",
	}
	seen := make(map[string]bool)
	var kinds []string
	for word, kind := range kindMap {
		if strings.Contains(q, word) && !seen[kind] {
			seen[kind] = true
			kinds = append(kinds, kind)
		}
	}
	return kinds
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
