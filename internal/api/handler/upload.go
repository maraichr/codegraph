package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/maraichr/codegraph/internal/ingestion"
	minioclient "github.com/maraichr/codegraph/internal/store/minio"
	"github.com/maraichr/codegraph/internal/store"
	"github.com/maraichr/codegraph/internal/store/postgres"
	"github.com/maraichr/codegraph/pkg/apierr"
)

type UploadHandler struct {
	logger   *slog.Logger
	store    *store.Store
	minio    *minioclient.Client
	producer *ingestion.Producer
}

func NewUploadHandler(logger *slog.Logger, s *store.Store, minio *minioclient.Client, producer *ingestion.Producer) *UploadHandler {
	return &UploadHandler{logger: logger, store: s, minio: minio, producer: producer}
}

func (h *UploadHandler) Upload(w http.ResponseWriter, r *http.Request) {
	projectSlug := chi.URLParam(r, "slug")

	// Max 100MB upload
	r.Body = http.MaxBytesReader(w, r.Body, 100*1024*1024)

	project, ok := getProjectOr404(w, r, h.logger, h.store, projectSlug)
	if !ok {
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeAPIError(w, h.logger, apierr.FileRequired())
		return
	}
	defer file.Close()

	// Create source record for this upload
	sourceName := header.Filename
	if sourceName == "" {
		sourceName = "upload-" + uuid.New().String()[:8]
	}

	// Pre-compute object name so we can store it in source config
	uploadID := uuid.New().String()
	objectName := fmt.Sprintf("%s/%s/%s", project.Slug, uploadID, header.Filename)
	configJSON, _ := json.Marshal(map[string]string{"object_name": objectName})

	source, err := h.store.CreateSource(r.Context(), postgres.CreateSourceParams{
		ProjectID:  project.ID,
		Name:       sourceName,
		SourceType: "upload",
		Config:     configJSON,
	})
	if err != nil {
		writeAPIError(w, h.logger, apierr.SourceCreateFailed(err))
		return
	}

	// Upload to MinIO
	if err := h.minio.UploadFile(r.Context(), objectName, file, header.Size); err != nil {
		writeAPIError(w, h.logger, apierr.UploadFailed(err))
		return
	}

	// Create IndexRun
	run, err := h.store.CreateIndexRun(r.Context(), postgres.CreateIndexRunParams{
		ProjectID: project.ID,
		SourceID:  pgtype.UUID{Bytes: source.ID, Valid: true},
	})
	if err != nil {
		writeAPIError(w, h.logger, apierr.IndexRunCreateFailed(err))
		return
	}

	// Enqueue for processing
	if h.producer != nil {
		h.enqueue(r.Context(), run, source, project)
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"source":    source,
		"index_run": run,
		"object":    objectName,
	})
}

func (h *UploadHandler) enqueue(ctx context.Context, run postgres.IndexRun, source postgres.Source, project postgres.Project) {
	msg := ingestion.IngestMessage{
		IndexRunID: run.ID,
		ProjectID:  project.ID,
		SourceID:   source.ID,
		SourceType: "upload",
		Trigger:    "manual",
	}
	if _, err := h.producer.Enqueue(ctx, msg); err != nil {
		h.logger.Error("enqueue ingestion", slog.String("error", err.Error()))
	}
}
