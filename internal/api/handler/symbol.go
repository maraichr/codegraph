package handler

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/maraichr/codegraph/internal/graph"
	"github.com/maraichr/codegraph/internal/impact"
	"github.com/maraichr/codegraph/internal/lineage"
	"github.com/maraichr/codegraph/internal/store"
	"github.com/maraichr/codegraph/internal/store/postgres"
	"github.com/maraichr/codegraph/pkg/apierr"
)

type SymbolHandler struct {
	logger  *slog.Logger
	store   *store.Store
	graph   *graph.Client
	lineage *lineage.Engine
	impact  *impact.Engine
}

func NewSymbolHandler(logger *slog.Logger, s *store.Store, g *graph.Client, lin *lineage.Engine, imp *impact.Engine) *SymbolHandler {
	return &SymbolHandler{logger: logger, store: s, graph: g, lineage: lin, impact: imp}
}

// Search finds symbols matching a query within a project.
// GET /projects/{slug}/symbols?q=...&kind=...&limit=...
func (h *SymbolHandler) Search(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	q := r.URL.Query().Get("q")
	if q == "" {
		writeAPIError(w, h.logger, apierr.New("QUERY_REQUIRED", http.StatusBadRequest, "Query parameter 'q' is required"))
		return
	}

	kinds := parseCSV(r.URL.Query().Get("kind"))
	if kinds == nil {
		kinds = []string{}
	}
	languages := parseCSV(r.URL.Query().Get("language"))
	if languages == nil {
		languages = []string{}
	}
	limit := intQuery(r, "limit", 20, 100)

	rows, err := h.store.SearchSymbols(r.Context(), postgres.SearchSymbolsParams{
		ProjectSlug: slug,
		Query:       &q,
		Kinds:       kinds,
		Languages:   languages,
		Lim:         int32(limit),
	})
	if err != nil {
		writeAPIError(w, h.logger, apierr.SearchFailed(err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"symbols": rows,
		"count":   len(rows),
	})
}

// Get returns a single symbol by ID.
// GET /symbols/{id}
func (h *SymbolHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeAPIError(w, h.logger, apierr.InvalidID("symbol"))
		return
	}

	sym, err := h.store.GetSymbol(r.Context(), id)
	if err != nil {
		if apierr.IsNotFound(err) {
			writeAPIError(w, h.logger, apierr.SymbolNotFound())
		} else {
			writeAPIError(w, h.logger, apierr.InternalError(err))
		}
		return
	}

	writeJSON(w, http.StatusOK, sym)
}

// References returns incoming/outgoing edges for a symbol.
// GET /symbols/{id}/references?direction=incoming|outgoing|both
func (h *SymbolHandler) References(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeAPIError(w, h.logger, apierr.InvalidID("symbol"))
		return
	}

	direction := r.URL.Query().Get("direction")
	if direction == "" {
		direction = "both"
	}

	var incoming, outgoing []postgres.SymbolEdge

	if direction == "incoming" || direction == "both" {
		incoming, err = h.store.GetIncomingEdges(r.Context(), id)
		if err != nil {
			writeAPIError(w, h.logger, apierr.InternalError(err))
			return
		}
	}
	if direction == "outgoing" || direction == "both" {
		outgoing, err = h.store.GetOutgoingEdges(r.Context(), id)
		if err != nil {
			writeAPIError(w, h.logger, apierr.InternalError(err))
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"incoming": incoming,
		"outgoing": outgoing,
	})
}

// Lineage returns the lineage graph for a symbol via Neo4j.
// GET /symbols/{id}/lineage?direction=upstream|downstream|both&max_depth=3
func (h *SymbolHandler) Lineage(w http.ResponseWriter, r *http.Request) {
	if h.graph == nil {
		writeAPIError(w, h.logger, apierr.NotImplemented("Lineage (Neo4j not configured)"))
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeAPIError(w, h.logger, apierr.InvalidID("symbol"))
		return
	}

	direction := r.URL.Query().Get("direction")
	if direction == "" {
		direction = "both"
	}
	maxDepth := intQuery(r, "max_depth", 3, 10)

	result, err := h.graph.Lineage(r.Context(), id, direction, maxDepth)
	if err != nil {
		writeAPIError(w, h.logger, apierr.LineageQueryFailed(err))
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// Impact returns downstream impact of changing a symbol.
// GET /symbols/{id}/impact?max_depth=5&change_type=modify
func (h *SymbolHandler) Impact(w http.ResponseWriter, r *http.Request) {
	if h.impact == nil {
		writeAPIError(w, h.logger, apierr.NotImplemented("Impact analysis (not configured)"))
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeAPIError(w, h.logger, apierr.InvalidID("symbol"))
		return
	}

	maxDepth := intQuery(r, "max_depth", 5, 10)
	changeType := r.URL.Query().Get("change_type")
	if changeType == "" {
		changeType = "modify"
	}

	result, err := h.impact.Analyze(r.Context(), id, changeType, maxDepth)
	if err != nil {
		writeAPIError(w, h.logger, apierr.LineageQueryFailed(err))
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// ColumnLineage returns column-level lineage for a symbol.
// GET /symbols/{id}/column-lineage?direction=both&max_depth=5
func (h *SymbolHandler) ColumnLineage(w http.ResponseWriter, r *http.Request) {
	if h.lineage == nil {
		writeAPIError(w, h.logger, apierr.NotImplemented("Column lineage (not configured)"))
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeAPIError(w, h.logger, apierr.InvalidID("symbol"))
		return
	}

	direction := r.URL.Query().Get("direction")
	if direction == "" {
		direction = "both"
	}
	maxDepth := intQuery(r, "max_depth", 5, 10)

	result, err := h.lineage.QueryColumnLineage(r.Context(), id, direction, maxDepth)
	if err != nil {
		writeAPIError(w, h.logger, apierr.LineageQueryFailed(err))
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// SearchGlobal finds symbols matching a query across all projects.
// GET /symbols/search?q=...&kind=...&language=...&limit=20
func (h *SymbolHandler) SearchGlobal(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeAPIError(w, h.logger, apierr.New("QUERY_REQUIRED", http.StatusBadRequest, "Query parameter 'q' is required"))
		return
	}

	kinds := parseCSV(r.URL.Query().Get("kind"))
	if kinds == nil {
		kinds = []string{}
	}
	languages := parseCSV(r.URL.Query().Get("language"))
	if languages == nil {
		languages = []string{}
	}
	limit := intQuery(r, "limit", 20, 100)

	rows, err := h.store.SearchSymbolsGlobal(r.Context(), postgres.SearchSymbolsGlobalParams{
		Query:     &q,
		Kinds:     kinds,
		Languages: languages,
		Lim:       int32(limit),
	})
	if err != nil {
		writeAPIError(w, h.logger, apierr.SearchFailed(err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"symbols": rows,
		"count":   len(rows),
	})
}

func parseCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, strings.ToLower(p))
		}
	}
	return result
}

func intQuery(r *http.Request, key string, defaultVal, maxVal int) int {
	v, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil || v <= 0 {
		return defaultVal
	}
	if v > maxVal {
		return maxVal
	}
	return v
}
