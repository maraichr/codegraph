package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/maraichr/lattice/internal/oracle"
	"github.com/maraichr/lattice/pkg/apierr"
)

// OracleHandler serves the Oracle chat endpoint.
type OracleHandler struct {
	logger *slog.Logger
	engine *oracle.Engine
}

// NewOracleHandler creates a new OracleHandler.
func NewOracleHandler(logger *slog.Logger, engine *oracle.Engine) *OracleHandler {
	return &OracleHandler{logger: logger, engine: engine}
}

// Ask handles POST /projects/{slug}/oracle
func (h *OracleHandler) Ask(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	project, ok := getProjectOr404(w, r, h.logger, h.engine.Store(), slug)
	if !ok {
		return
	}
	if !checkTenantAccess(w, r, h.logger, project) {
		return
	}

	var req oracle.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, h.logger, apierr.InvalidRequestBody())
		return
	}

	if req.Question == "" {
		writeAPIError(w, h.logger, apierr.New("QUESTION_REQUIRED", http.StatusBadRequest, "Question is required"))
		return
	}

	resp, err := h.engine.Ask(r.Context(), project, req)
	if err != nil {
		h.logger.Error("oracle ask failed", slog.String("error", err.Error()), slog.String("project", slug))
		writeAPIError(w, h.logger, apierr.Wrap("ORACLE_FAILED", http.StatusInternalServerError, "Oracle query failed", err))
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
