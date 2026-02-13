package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	pgvector "github.com/pgvector/pgvector-go"

	"github.com/codegraph-labs/codegraph/internal/embedding"
	"github.com/codegraph-labs/codegraph/internal/store"
	"github.com/codegraph-labs/codegraph/internal/store/postgres"
	"github.com/codegraph-labs/codegraph/pkg/apierr"
)

type SearchHandler struct {
	logger *slog.Logger
	store  *store.Store
	embed  *embedding.Client
}

func NewSearchHandler(logger *slog.Logger, s *store.Store, embed *embedding.Client) *SearchHandler {
	return &SearchHandler{logger: logger, store: s, embed: embed}
}

// Semantic performs semantic search using vector embeddings.
// POST /projects/{slug}/search/semantic
func (h *SearchHandler) Semantic(w http.ResponseWriter, r *http.Request) {
	if h.embed == nil {
		writeAPIError(w, h.logger, apierr.NotImplemented("Semantic search (embeddings not configured)"))
		return
	}

	slug := chi.URLParam(r, "slug")

	var req struct {
		Query string   `json:"query"`
		Kinds []string `json:"kinds"`
		TopK  int      `json:"top_k"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, h.logger, apierr.InvalidRequestBody())
		return
	}

	if req.Query == "" {
		writeAPIError(w, h.logger, apierr.New("QUERY_REQUIRED", http.StatusBadRequest, "Query text is required"))
		return
	}
	if req.TopK <= 0 || req.TopK > 100 {
		req.TopK = 20
	}

	// Look up project to get its ID
	project, ok := getProjectOr404(w, r, h.logger, h.store, slug)
	if !ok {
		return
	}

	// Embed the query text
	embeddings, err := h.embed.EmbedBatch(r.Context(), []string{req.Query}, "search_query")
	if err != nil {
		writeAPIError(w, h.logger, apierr.EmbeddingFailed(err))
		return
	}
	if len(embeddings) == 0 {
		writeAPIError(w, h.logger, apierr.EmbeddingFailed(nil))
		return
	}

	queryVec := pgvector.NewVector(embeddings[0])

	rows, err := h.store.SemanticSearch(r.Context(), postgres.SemanticSearchParams{
		ProjectID:      project.ID,
		QueryEmbedding: queryVec,
		Kinds:          req.Kinds,
		Lim:            int32(req.TopK),
	})
	if err != nil {
		writeAPIError(w, h.logger, apierr.SearchFailed(err))
		return
	}

	type result struct {
		Symbol   postgres.SemanticSearchRow `json:"symbol"`
		Score    float64                    `json:"score"`
		Distance any                        `json:"distance"`
	}
	results := make([]result, len(rows))
	for i, row := range rows {
		results[i] = result{
			Symbol:   row,
			Distance: row.Distance,
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"results": results,
		"count":   len(results),
	})
}
