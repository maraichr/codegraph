package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/codegraph-labs/codegraph/internal/store"
	"github.com/codegraph-labs/codegraph/internal/store/postgres"
	"github.com/codegraph-labs/codegraph/pkg/apierr"
)

type IndexRunHandler struct {
	logger *slog.Logger
	store  *store.Store
}

func NewIndexRunHandler(logger *slog.Logger, s *store.Store) *IndexRunHandler {
	return &IndexRunHandler{logger: logger, store: s}
}

func (h *IndexRunHandler) List(w http.ResponseWriter, r *http.Request) {
	projectSlug := chi.URLParam(r, "slug")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	runs, err := h.store.ListIndexRunsByProject(r.Context(), postgres.ListIndexRunsByProjectParams{
		Slug:   projectSlug,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		writeAPIError(w, h.logger, apierr.IndexRunListFailed(err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"index_runs": runs,
		"total":      len(runs),
	})
}

func (h *IndexRunHandler) Get(w http.ResponseWriter, r *http.Request) {
	runID, err := uuid.Parse(chi.URLParam(r, "runID"))
	if err != nil {
		writeAPIError(w, h.logger, apierr.InvalidRunID())
		return
	}

	run, err := h.store.GetIndexRun(r.Context(), runID)
	if err != nil {
		if apierr.IsNotFound(err) {
			writeAPIError(w, h.logger, apierr.IndexRunNotFound())
		} else {
			writeAPIError(w, h.logger, apierr.InternalError(err))
		}
		return
	}

	writeJSON(w, http.StatusOK, run)
}

func (h *IndexRunHandler) Trigger(w http.ResponseWriter, r *http.Request) {
	projectSlug := chi.URLParam(r, "slug")

	project, ok := getProjectOr404(w, r, h.logger, h.store, projectSlug)
	if !ok {
		return
	}

	// Optional source_id from query or body
	var sourceID pgtype.UUID
	if sid := r.URL.Query().Get("source_id"); sid != "" {
		parsed, err := uuid.Parse(sid)
		if err != nil {
			writeAPIError(w, h.logger, apierr.InvalidSourceID())
			return
		}
		sourceID = pgtype.UUID{Bytes: parsed, Valid: true}
	}

	run, err := h.store.CreateIndexRun(r.Context(), postgres.CreateIndexRunParams{
		ProjectID: project.ID,
		SourceID:  sourceID,
	})
	if err != nil {
		writeAPIError(w, h.logger, apierr.IndexRunCreateFailed(err))
		return
	}

	// TODO: enqueue to Valkey stream for worker pickup

	writeJSON(w, http.StatusCreated, run)
}
