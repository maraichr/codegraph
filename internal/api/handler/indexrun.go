package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/codegraph-labs/codegraph/internal/ingestion"
	"github.com/codegraph-labs/codegraph/internal/store"
	"github.com/codegraph-labs/codegraph/internal/store/postgres"
	"github.com/codegraph-labs/codegraph/pkg/apierr"
)

type IndexRunHandler struct {
	logger   *slog.Logger
	store    *store.Store
	producer *ingestion.Producer
}

func NewIndexRunHandler(logger *slog.Logger, s *store.Store, producer *ingestion.Producer) *IndexRunHandler {
	return &IndexRunHandler{logger: logger, store: s, producer: producer}
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
	if sid := r.URL.Query().Get("source_id"); sid != "" {
		parsed, err := uuid.Parse(sid)
		if err != nil {
			writeAPIError(w, h.logger, apierr.InvalidSourceID())
			return
		}
		source, err := h.store.GetSource(r.Context(), parsed)
		if err != nil {
			writeAPIError(w, h.logger, apierr.SourceNotFound())
			return
		}
		run := h.triggerSource(w, r, project.ID, source)
		if run == nil {
			return
		}
		writeJSON(w, http.StatusCreated, run)
		return
	}

	// No source_id — trigger all sources for this project
	sources, err := h.store.ListSourcesByProjectID(r.Context(), project.ID)
	if err != nil || len(sources) == 0 {
		writeAPIError(w, h.logger, apierr.NoSources())
		return
	}

	var runs []postgres.IndexRun
	for _, source := range sources {
		run := h.triggerSource(w, r, project.ID, source)
		if run == nil {
			return // error already written
		}
		runs = append(runs, *run)
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"index_runs": runs,
	})
}

func (h *IndexRunHandler) triggerSource(w http.ResponseWriter, r *http.Request, projectID uuid.UUID, source postgres.Source) *postgres.IndexRun {
	sourceID := pgtype.UUID{Bytes: source.ID, Valid: true}
	run, err := h.store.CreateIndexRun(r.Context(), postgres.CreateIndexRunParams{
		ProjectID: projectID,
		SourceID:  sourceID,
	})
	if err != nil {
		writeAPIError(w, h.logger, apierr.IndexRunCreateFailed(err))
		return nil
	}

	if h.producer != nil {
		msg := ingestion.IngestMessage{
			IndexRunID: run.ID,
			ProjectID:  projectID,
			SourceID:   source.ID,
			SourceType: source.SourceType,
			Trigger:    "manual",
		}
		if _, err := h.producer.Enqueue(r.Context(), msg); err != nil {
			h.logger.Error("enqueue ingestion", slog.String("error", err.Error()))
		}
	}

	return &run
}
