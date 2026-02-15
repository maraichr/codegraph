package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/maraichr/codegraph/internal/store"
	"github.com/maraichr/codegraph/internal/store/postgres"
	"github.com/maraichr/codegraph/pkg/apierr"
)

type SourceHandler struct {
	logger *slog.Logger
	store  *store.Store
}

func NewSourceHandler(logger *slog.Logger, s *store.Store) *SourceHandler {
	return &SourceHandler{logger: logger, store: s}
}

func (h *SourceHandler) List(w http.ResponseWriter, r *http.Request) {
	projectSlug := chi.URLParam(r, "slug")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	sources, err := h.store.ListSourcesByProject(r.Context(), postgres.ListSourcesByProjectParams{
		Slug:   projectSlug,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		writeAPIError(w, h.logger, apierr.SourceListFailed(err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"sources": sources,
		"total":   len(sources),
	})
}

func (h *SourceHandler) Get(w http.ResponseWriter, r *http.Request) {
	sourceID, err := uuid.Parse(chi.URLParam(r, "sourceID"))
	if err != nil {
		writeAPIError(w, h.logger, apierr.InvalidSourceID())
		return
	}

	source, ok := getSourceOr404(w, r, h.logger, h.store, sourceID)
	if !ok {
		return
	}

	writeJSON(w, http.StatusOK, source)
}

func (h *SourceHandler) Create(w http.ResponseWriter, r *http.Request) {
	projectSlug := chi.URLParam(r, "slug")

	var req struct {
		Name          string          `json:"name"`
		SourceType    string          `json:"source_type"`
		ConnectionURI *string         `json:"connection_uri"`
		Config        json.RawMessage `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, h.logger, apierr.InvalidRequestBody())
		return
	}

	if err := validateName(req.Name); err != nil {
		writeAPIError(w, h.logger, err)
		return
	}
	if err := validateSourceType(req.SourceType); err != nil {
		writeAPIError(w, h.logger, err)
		return
	}

	project, ok := getProjectOr404(w, r, h.logger, h.store, projectSlug)
	if !ok {
		return
	}

	configBytes := []byte("{}")
	if len(req.Config) > 0 {
		configBytes = req.Config
	}

	source, err := h.store.CreateSource(r.Context(), postgres.CreateSourceParams{
		ProjectID:     project.ID,
		Name:          req.Name,
		SourceType:    req.SourceType,
		ConnectionUri: req.ConnectionURI,
		Config:        configBytes,
	})
	if err != nil {
		writeAPIError(w, h.logger, apierr.SourceCreateFailed(err))
		return
	}

	writeJSON(w, http.StatusCreated, source)
}

func (h *SourceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	sourceID, err := uuid.Parse(chi.URLParam(r, "sourceID"))
	if err != nil {
		writeAPIError(w, h.logger, apierr.InvalidSourceID())
		return
	}

	if _, ok := getSourceOr404(w, r, h.logger, h.store, sourceID); !ok {
		return
	}

	if err := h.store.DeleteSource(r.Context(), sourceID); err != nil {
		writeAPIError(w, h.logger, apierr.SourceDeleteFailed(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
