package handler

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/codegraph-labs/codegraph/internal/ingestion"
	"github.com/codegraph-labs/codegraph/internal/store"
	"github.com/codegraph-labs/codegraph/internal/store/postgres"
	"github.com/codegraph-labs/codegraph/pkg/apierr"
)

type WebhookHandler struct {
	logger   *slog.Logger
	store    *store.Store
	producer *ingestion.Producer
}

func NewWebhookHandler(logger *slog.Logger, s *store.Store, producer *ingestion.Producer) *WebhookHandler {
	return &WebhookHandler{logger: logger, store: s, producer: producer}
}

// GitLabPush handles POST /api/v1/webhooks/gitlab/{sourceID}
func (h *WebhookHandler) GitLabPush(w http.ResponseWriter, r *http.Request) {
	sourceID, err := uuid.Parse(chi.URLParam(r, "sourceID"))
	if err != nil {
		writeAPIError(w, h.logger, apierr.InvalidSourceID())
		return
	}

	// Validate X-Gitlab-Token header
	token := r.Header.Get("X-Gitlab-Token")
	if token == "" {
		writeAPIError(w, h.logger, apierr.MissingAuthToken())
		return
	}

	source, ok := getSourceOr404(w, r, h.logger, h.store, sourceID)
	if !ok {
		return
	}

	// Validate webhook secret - for Phase 1, we use a static env var
	// In production this would be per-source from encrypted config
	expectedToken := "codegraph-webhook-secret" // TODO: read from source config or env
	if subtle.ConstantTimeCompare([]byte(token), []byte(expectedToken)) != 1 {
		writeAPIError(w, h.logger, apierr.InvalidAuthToken())
		return
	}

	// Create index run
	run, err := h.store.CreateIndexRun(r.Context(), postgres.CreateIndexRunParams{
		ProjectID: source.ProjectID,
		SourceID:  pgtype.UUID{Bytes: source.ID, Valid: true},
	})
	if err != nil {
		writeAPIError(w, h.logger, apierr.IndexRunCreateFailed(err))
		return
	}

	// Enqueue
	if h.producer != nil {
		h.enqueue(r.Context(), run, source)
	}

	h.logger.Info("webhook received",
		slog.String("source_id", sourceID.String()),
		slog.String("index_run_id", run.ID.String()))

	writeJSON(w, http.StatusCreated, map[string]any{
		"index_run": run,
	})
}

func (h *WebhookHandler) enqueue(ctx context.Context, run postgres.IndexRun, source postgres.Source) {
	msg := ingestion.IngestMessage{
		IndexRunID: run.ID,
		ProjectID:  source.ProjectID,
		SourceID:   source.ID,
		SourceType: source.SourceType,
		Trigger:    "webhook",
	}
	if _, err := h.producer.Enqueue(ctx, msg); err != nil {
		h.logger.Error("enqueue ingestion", slog.String("error", err.Error()))
	}
}
