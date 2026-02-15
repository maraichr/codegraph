package tools

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/maraichr/lattice/internal/auth"
	"github.com/maraichr/lattice/internal/mcp"
	"github.com/maraichr/lattice/internal/store"
	"github.com/maraichr/lattice/internal/store/postgres"
)

// GetProjectAnalyticsParams are the parameters for the get_project_analytics tool.
type GetProjectAnalyticsParams struct {
	Project string `json:"project"`
	Scope   string `json:"scope,omitempty"` // summary, languages, kinds, layers, bridges
}

// GetProjectAnalyticsHandler implements the get_project_analytics MCP tool.
type GetProjectAnalyticsHandler struct {
	store  *store.Store
	logger *slog.Logger
}

// NewGetProjectAnalyticsHandler creates a new handler.
func NewGetProjectAnalyticsHandler(s *store.Store, logger *slog.Logger) *GetProjectAnalyticsHandler {
	return &GetProjectAnalyticsHandler{store: s, logger: logger}
}

// Handle returns project analytics for the requested scope.
func (h *GetProjectAnalyticsHandler) Handle(ctx context.Context, params GetProjectAnalyticsParams) (string, error) {
	if params.Scope == "" {
		params.Scope = "summary"
	}

	project, err := h.store.GetProject(ctx, params.Project)
	if err != nil {
		return "", WrapProjectError(err)
	}
	if p, ok := auth.PrincipalFrom(ctx); ok && !p.IsAdmin() && project.TenantID != p.TenantID {
		return "", fmt.Errorf("access denied to project %s", params.Project)
	}

	rb := mcp.NewResponseBuilder(4000)

	switch params.Scope {
	case "summary":
		return h.handleSummary(ctx, project, rb)
	case "languages":
		return h.handleLanguages(ctx, project, rb)
	case "kinds":
		return h.handleKinds(ctx, project, rb)
	case "layers":
		return h.handleLayers(ctx, project, rb)
	case "bridges":
		return h.handleBridges(ctx, project, rb)
	default:
		return "", fmt.Errorf("unknown scope: %s (valid: summary, languages, kinds, layers, bridges)", params.Scope)
	}
}

func (h *GetProjectAnalyticsHandler) handleSummary(ctx context.Context, project postgres.Project, rb *mcp.ResponseBuilder) (string, error) {
	rb.AddHeader(fmt.Sprintf("**Project Analytics: %s** (summary)", project.Name))

	stats, err := h.store.GetProjectSymbolStats(ctx, project.ID)
	if err != nil {
		rb.AddLine("No analytics data available. Run an indexing job first.")
		return rb.Finalize(0, 0), nil
	}

	rb.AddLine(fmt.Sprintf("- **Total symbols:** %d", stats.TotalSymbols))
	rb.AddLine(fmt.Sprintf("- **Languages:** %d", stats.LanguageCount))
	rb.AddLine(fmt.Sprintf("- **Symbol kinds:** %d", stats.KindCount))
	rb.AddLine(fmt.Sprintf("- **Files:** %d", stats.FileCount))

	// Try to get stored analytics summary
	analytics, err := h.store.GetProjectAnalytics(ctx, postgres.GetProjectAnalyticsParams{
		ProjectID: project.ID,
		Scope:     "project",
		ScopeID:   "overview",
	})
	if err == nil && analytics.Summary != nil {
		rb.AddLine("")
		rb.AddLine(*analytics.Summary)
	}

	return rb.Finalize(1, 1), nil
}

func (h *GetProjectAnalyticsHandler) handleLanguages(ctx context.Context, project postgres.Project, rb *mcp.ResponseBuilder) (string, error) {
	rb.AddHeader(fmt.Sprintf("**Project Analytics: %s** (languages)", project.Name))

	rows, err := h.store.GetSymbolCountsByLanguage(ctx, project.ID)
	if err != nil {
		return "", fmt.Errorf("get language counts: %w", err)
	}

	if len(rows) == 0 {
		rb.AddLine("No language data available.")
		return rb.Finalize(0, 0), nil
	}

	for _, r := range rows {
		rb.AddLine(fmt.Sprintf("- **%s:** %d symbols", r.Language, r.Cnt))
	}

	return rb.Finalize(len(rows), len(rows)), nil
}

func (h *GetProjectAnalyticsHandler) handleKinds(ctx context.Context, project postgres.Project, rb *mcp.ResponseBuilder) (string, error) {
	rb.AddHeader(fmt.Sprintf("**Project Analytics: %s** (kinds)", project.Name))

	rows, err := h.store.GetSymbolCountsByKind(ctx, project.ID)
	if err != nil {
		return "", fmt.Errorf("get kind counts: %w", err)
	}

	if len(rows) == 0 {
		rb.AddLine("No kind data available.")
		return rb.Finalize(0, 0), nil
	}

	for _, r := range rows {
		rb.AddLine(fmt.Sprintf("- **%s:** %d", r.Kind, r.Cnt))
	}

	return rb.Finalize(len(rows), len(rows)), nil
}

func (h *GetProjectAnalyticsHandler) handleLayers(ctx context.Context, project postgres.Project, rb *mcp.ResponseBuilder) (string, error) {
	rb.AddHeader(fmt.Sprintf("**Project Analytics: %s** (layers)", project.Name))

	rows, err := h.store.CountSymbolsByLayer(ctx, project.ID)
	if err != nil {
		return "", fmt.Errorf("get layer counts: %w", err)
	}

	if len(rows) == 0 {
		rb.AddLine("No layer data available. Run analytics pipeline first.")
		return rb.Finalize(0, 0), nil
	}

	for _, r := range rows {
		rb.AddLine(fmt.Sprintf("- **%v:** %d symbols", r.Layer, r.Cnt))
	}

	return rb.Finalize(len(rows), len(rows)), nil
}

func (h *GetProjectAnalyticsHandler) handleBridges(ctx context.Context, project postgres.Project, rb *mcp.ResponseBuilder) (string, error) {
	rb.AddHeader(fmt.Sprintf("**Project Analytics: %s** (cross-language bridges)", project.Name))

	rows, err := h.store.GetCrossLanguageBridges(ctx, project.ID)
	if err != nil {
		return "", fmt.Errorf("get bridges: %w", err)
	}

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
