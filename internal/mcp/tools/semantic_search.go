package tools

import (
	"context"
	"fmt"
	"log/slog"

	pgvector_go "github.com/pgvector/pgvector-go"

	"github.com/maraichr/lattice/internal/auth"
	"github.com/maraichr/lattice/internal/embedding"
	"github.com/maraichr/lattice/internal/mcp"
	"github.com/maraichr/lattice/internal/store"
	"github.com/maraichr/lattice/internal/store/postgres"
)

// SemanticSearchParams are the parameters for the semantic_search tool.
type SemanticSearchParams struct {
	Project string   `json:"project"`
	Query   string   `json:"query"`
	Kinds   []string `json:"kinds,omitempty"`
	TopK    int32    `json:"top_k,omitempty"`
}

// SemanticSearchHandler implements the semantic_search MCP tool.
type SemanticSearchHandler struct {
	store    *store.Store
	embedder embedding.Embedder
	logger   *slog.Logger
}

// NewSemanticSearchHandler creates a new handler.
func NewSemanticSearchHandler(s *store.Store, embedder embedding.Embedder, logger *slog.Logger) *SemanticSearchHandler {
	return &SemanticSearchHandler{store: s, embedder: embedder, logger: logger}
}

// Handle performs semantic (vector) search over symbols.
func (h *SemanticSearchHandler) Handle(ctx context.Context, params SemanticSearchParams) (string, error) {
	if h.embedder == nil {
		return "", fmt.Errorf("semantic search is not available: no embedding provider configured. Set OPENROUTER_API_KEY or BEDROCK_REGION")
	}
	if params.Query == "" {
		return "", fmt.Errorf("query is required")
	}
	if params.TopK <= 0 {
		params.TopK = 10
	}

	project, err := h.store.GetProject(ctx, params.Project)
	if err != nil {
		return "", WrapProjectError(err)
	}
	if p, ok := auth.PrincipalFrom(ctx); ok && !p.IsAdmin() && project.TenantID != p.TenantID {
		return "", fmt.Errorf("access denied to project %s", params.Project)
	}

	// Embed the query
	vectors, err := h.embedder.EmbedBatch(ctx, []string{params.Query}, "search_query")
	if err != nil {
		return "", fmt.Errorf("embed query: %w", err)
	}
	if len(vectors) == 0 || len(vectors[0]) == 0 {
		return "", fmt.Errorf("embedding returned empty vector")
	}

	kinds := params.Kinds
	if kinds == nil {
		kinds = []string{}
	}

	results, err := h.store.SemanticSearch(ctx, postgres.SemanticSearchParams{
		QueryEmbedding: pgvector_go.NewVector(vectors[0]),
		ProjectID:      project.ID,
		Kinds:          kinds,
		Lim:            params.TopK,
	})
	if err != nil {
		return "", fmt.Errorf("semantic search: %w", err)
	}

	if len(results) == 0 {
		return fmt.Sprintf("No semantic matches found for '%s'.", params.Query), nil
	}

	rb := mcp.NewResponseBuilder(4000)
	rb.AddHeader(fmt.Sprintf("**Semantic Search: %s** (%d results)", params.Query, len(results)))

	for i, r := range results {
		sig := ""
		if r.Signature != nil {
			sig = fmt.Sprintf("\n  Signature: `%s`", *r.Signature)
		}
		dist := ""
		if r.Distance != nil {
			dist = fmt.Sprintf(" (distance: %v)", r.Distance)
		}
		rb.AddLine(fmt.Sprintf("%d. **%s** `%s`%s\n   %s [%s] %s:%d-%d%s",
			i+1, r.Kind, r.Name, dist,
			r.QualifiedName, r.Language,
			r.FileID.String()[:8], r.StartLine, r.EndLine, sig))
	}

	return rb.Finalize(len(results), len(results)), nil
}
