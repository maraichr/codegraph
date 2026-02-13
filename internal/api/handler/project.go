package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/codegraph-labs/codegraph/internal/store"
	"github.com/codegraph-labs/codegraph/internal/store/postgres"
	"github.com/codegraph-labs/codegraph/pkg/apierr"
)

type ProjectHandler struct {
	logger *slog.Logger
	store  *store.Store
}

func NewProjectHandler(logger *slog.Logger, s *store.Store) *ProjectHandler {
	return &ProjectHandler{logger: logger, store: s}
}

func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	projects, err := h.store.ListProjects(r.Context(), postgres.ListProjectsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		writeAPIError(w, h.logger, apierr.ProjectListFailed(err))
		return
	}

	total, err := h.store.CountProjects(r.Context())
	if err != nil {
		writeAPIError(w, h.logger, apierr.ProjectCountFailed(err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"projects": projects,
		"total":    total,
	})
}

func (h *ProjectHandler) Get(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	project, ok := getProjectOr404(w, r, h.logger, h.store, slug)
	if !ok {
		return
	}

	writeJSON(w, http.StatusOK, project)
}

func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string  `json:"name"`
		Slug        string  `json:"slug"`
		Description *string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, h.logger, apierr.InvalidRequestBody())
		return
	}

	if err := validateSlug(req.Slug); err != nil {
		writeAPIError(w, h.logger, err)
		return
	}
	if err := validateName(req.Name); err != nil {
		writeAPIError(w, h.logger, err)
		return
	}

	project, err := h.store.CreateProject(r.Context(), postgres.CreateProjectParams{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
	})
	if err != nil {
		writeAPIError(w, h.logger, apierr.ProjectCreateFailed(err))
		return
	}

	writeJSON(w, http.StatusCreated, project)
}

func (h *ProjectHandler) Update(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	var req struct {
		Name        string  `json:"name"`
		Description *string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, h.logger, apierr.InvalidRequestBody())
		return
	}

	if req.Name != "" {
		if err := validateName(req.Name); err != nil {
			writeAPIError(w, h.logger, err)
			return
		}
	}

	current, ok := getProjectOr404(w, r, h.logger, h.store, slug)
	if !ok {
		return
	}

	name := current.Name
	if req.Name != "" {
		name = req.Name
	}
	desc := current.Description
	if req.Description != nil {
		desc = req.Description
	}

	project, err := h.store.UpdateProject(r.Context(), postgres.UpdateProjectParams{
		Slug:        slug,
		Name:        name,
		Description: desc,
		Settings:    current.Settings,
	})
	if err != nil {
		writeAPIError(w, h.logger, apierr.ProjectUpdateFailed(err))
		return
	}

	writeJSON(w, http.StatusOK, project)
}

func (h *ProjectHandler) Delete(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	if _, ok := getProjectOr404(w, r, h.logger, h.store, slug); !ok {
		return
	}

	if err := h.store.DeleteProject(r.Context(), slug); err != nil {
		writeAPIError(w, h.logger, apierr.ProjectDeleteFailed(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
