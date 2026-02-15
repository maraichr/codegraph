package handler

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/maraichr/codegraph/internal/store"
	"github.com/maraichr/codegraph/internal/store/postgres"
	"github.com/maraichr/codegraph/pkg/apierr"
)

// AnalyticsHandler serves project analytics endpoints.
type AnalyticsHandler struct {
	logger *slog.Logger
	store  *store.Store
}

func NewAnalyticsHandler(logger *slog.Logger, s *store.Store) *AnalyticsHandler {
	return &AnalyticsHandler{logger: logger, store: s}
}

// Summary returns the full project analytics JSON + summary text.
// GET /projects/{slug}/analytics/summary
func (h *AnalyticsHandler) Summary(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	project, ok := getProjectOr404(w, r, h.logger, h.store, slug)
	if !ok {
		return
	}
	if !checkTenantAccess(w, r, h.logger, project) {
		return
	}

	analytics, err := h.store.GetProjectAnalytics(r.Context(), postgres.GetProjectAnalyticsParams{
		ProjectID: project.ID,
		Scope:     "project",
		ScopeID:   project.ID.String(),
	})
	if err != nil {
		if apierr.IsNotFound(err) {
			writeJSON(w, http.StatusOK, map[string]any{
				"analytics": nil,
				"summary":   nil,
			})
			return
		}
		writeAPIError(w, h.logger, apierr.AnalyticsFailed(err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"analytics": analytics.Analytics,
		"summary":   analytics.Summary,
	})
}

// Stats returns aggregate symbol/file/language/kind counts.
// GET /projects/{slug}/analytics/stats
func (h *AnalyticsHandler) Stats(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	project, ok := getProjectOr404(w, r, h.logger, h.store, slug)
	if !ok {
		return
	}
	if !checkTenantAccess(w, r, h.logger, project) {
		return
	}

	stats, err := h.store.GetProjectSymbolStats(r.Context(), project.ID)
	if err != nil {
		writeAPIError(w, h.logger, apierr.AnalyticsFailed(err))
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

// Languages returns symbol counts grouped by language.
// GET /projects/{slug}/analytics/languages
func (h *AnalyticsHandler) Languages(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	project, ok := getProjectOr404(w, r, h.logger, h.store, slug)
	if !ok {
		return
	}
	if !checkTenantAccess(w, r, h.logger, project) {
		return
	}

	rows, err := h.store.GetSymbolCountsByLanguage(r.Context(), project.ID)
	if err != nil {
		writeAPIError(w, h.logger, apierr.AnalyticsFailed(err))
		return
	}

	writeJSON(w, http.StatusOK, rows)
}

// Kinds returns symbol counts grouped by kind.
// GET /projects/{slug}/analytics/kinds
func (h *AnalyticsHandler) Kinds(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	project, ok := getProjectOr404(w, r, h.logger, h.store, slug)
	if !ok {
		return
	}
	if !checkTenantAccess(w, r, h.logger, project) {
		return
	}

	rows, err := h.store.GetSymbolCountsByKind(r.Context(), project.ID)
	if err != nil {
		writeAPIError(w, h.logger, apierr.AnalyticsFailed(err))
		return
	}

	writeJSON(w, http.StatusOK, rows)
}

// Layers returns symbol counts grouped by architectural layer.
// GET /projects/{slug}/analytics/layers
func (h *AnalyticsHandler) Layers(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	project, ok := getProjectOr404(w, r, h.logger, h.store, slug)
	if !ok {
		return
	}
	if !checkTenantAccess(w, r, h.logger, project) {
		return
	}

	rows, err := h.store.CountSymbolsByLayer(r.Context(), project.ID)
	if err != nil {
		writeAPIError(w, h.logger, apierr.AnalyticsFailed(err))
		return
	}

	writeJSON(w, http.StatusOK, rows)
}

// LayerSymbols returns paginated symbols for a specific layer.
// GET /projects/{slug}/analytics/layers/{layer}?limit=20&offset=0
func (h *AnalyticsHandler) LayerSymbols(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	layer := chi.URLParam(r, "layer")
	project, ok := getProjectOr404(w, r, h.logger, h.store, slug)
	if !ok {
		return
	}
	if !checkTenantAccess(w, r, h.logger, project) {
		return
	}

	limit := intQuery(r, "limit", 20, 100)
	offset := intQuery(r, "offset", 0, 10000)

	rows, err := h.store.GetSymbolsByLayer(r.Context(), postgres.GetSymbolsByLayerParams{
		ProjectID: project.ID,
		Metadata:  []byte(layer),
		Limit:     int32(limit),
		Offset:    int32(offset),
	})
	if err != nil {
		writeAPIError(w, h.logger, apierr.AnalyticsFailed(err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"symbols": rows,
		"count":   len(rows),
	})
}

// TopByInDegree returns the top-N most depended-upon symbols.
// GET /projects/{slug}/analytics/top/in-degree?limit=10
func (h *AnalyticsHandler) TopByInDegree(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	project, ok := getProjectOr404(w, r, h.logger, h.store, slug)
	if !ok {
		return
	}
	if !checkTenantAccess(w, r, h.logger, project) {
		return
	}

	limit := intQuery(r, "limit", 10, 100)

	rows, err := h.store.TopSymbolsByInDegree(r.Context(), postgres.TopSymbolsByInDegreeParams{
		ProjectID: project.ID,
		Limit:     int32(limit),
	})
	if err != nil {
		writeAPIError(w, h.logger, apierr.AnalyticsFailed(err))
		return
	}

	writeJSON(w, http.StatusOK, rows)
}

// TopByPageRank returns the top-N highest centrality symbols.
// GET /projects/{slug}/analytics/top/pagerank?limit=10
func (h *AnalyticsHandler) TopByPageRank(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	project, ok := getProjectOr404(w, r, h.logger, h.store, slug)
	if !ok {
		return
	}
	if !checkTenantAccess(w, r, h.logger, project) {
		return
	}

	limit := intQuery(r, "limit", 10, 100)

	rows, err := h.store.TopSymbolsByPageRank(r.Context(), postgres.TopSymbolsByPageRankParams{
		ProjectID: project.ID,
		Limit:     int32(limit),
	})
	if err != nil {
		writeAPIError(w, h.logger, apierr.AnalyticsFailed(err))
		return
	}

	writeJSON(w, http.StatusOK, rows)
}

// Bridges returns cross-language edge summary.
// GET /projects/{slug}/analytics/bridges
func (h *AnalyticsHandler) Bridges(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	project, ok := getProjectOr404(w, r, h.logger, h.store, slug)
	if !ok {
		return
	}
	if !checkTenantAccess(w, r, h.logger, project) {
		return
	}

	rows, err := h.store.GetCrossLanguageBridges(r.Context(), project.ID)
	if err != nil {
		writeAPIError(w, h.logger, apierr.AnalyticsFailed(err))
		return
	}

	writeJSON(w, http.StatusOK, rows)
}

// Sources returns per-source symbol stats.
// GET /projects/{slug}/analytics/sources
func (h *AnalyticsHandler) Sources(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	project, ok := getProjectOr404(w, r, h.logger, h.store, slug)
	if !ok {
		return
	}
	if !checkTenantAccess(w, r, h.logger, project) {
		return
	}

	rows, err := h.store.GetSourceSymbolStats(r.Context(), project.ID)
	if err != nil {
		writeAPIError(w, h.logger, apierr.AnalyticsFailed(err))
		return
	}

	writeJSON(w, http.StatusOK, rows)
}

// Coverage returns parser coverage per source (total files vs parsed files).
// GET /projects/{slug}/analytics/coverage
func (h *AnalyticsHandler) Coverage(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	project, ok := getProjectOr404(w, r, h.logger, h.store, slug)
	if !ok {
		return
	}
	if !checkTenantAccess(w, r, h.logger, project) {
		return
	}

	rows, err := h.store.GetParserCoverage(r.Context(), project.ID)
	if err != nil {
		writeAPIError(w, h.logger, apierr.AnalyticsFailed(err))
		return
	}

	writeJSON(w, http.StatusOK, rows)
}
