package embedding

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	pgvector "github.com/pgvector/pgvector-go"

	"github.com/maraichr/codegraph/internal/store"
	"github.com/maraichr/codegraph/internal/store/postgres"
)

// EmbedSymbols generates and stores embeddings for all symbols in a project
// that don't already have them. Returns the number of symbols embedded.
func EmbedSymbols(ctx context.Context, client Embedder, s *store.Store, projectID uuid.UUID, logger *slog.Logger) (int, error) {
	// Find symbols without embeddings
	symbols, err := s.ListSymbolsWithoutEmbeddings(ctx, projectID)
	if err != nil {
		return 0, fmt.Errorf("list symbols without embeddings: %w", err)
	}

	if len(symbols) == 0 {
		return 0, nil
	}

	logger.Info("embedding symbols", slog.Int("count", len(symbols)))

	// Build text representations
	texts := make([]string, len(symbols))
	for i, sym := range symbols {
		texts[i] = BuildEmbeddingText(sym)
	}

	// Generate embeddings
	embeddings, err := client.EmbedBatch(ctx, texts, "search_document")
	if err != nil {
		return 0, fmt.Errorf("embed batch: %w", err)
	}

	if len(embeddings) != len(symbols) {
		return 0, fmt.Errorf("embedding count mismatch: got %d, expected %d", len(embeddings), len(symbols))
	}

	// Store embeddings
	for i, sym := range symbols {
		vec := pgvector.NewVector(embeddings[i])
		err := s.UpsertSymbolEmbedding(ctx, postgres.UpsertSymbolEmbeddingParams{
			SymbolID:  sym.ID,
			Embedding: vec,
			Model:     client.ModelID(),
		})
		if err != nil {
			return i, fmt.Errorf("upsert embedding for %s: %w", sym.QualifiedName, err)
		}
	}

	return len(symbols), nil
}
