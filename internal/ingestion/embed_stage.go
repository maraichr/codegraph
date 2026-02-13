package ingestion

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/codegraph-labs/codegraph/internal/embedding"
	"github.com/codegraph-labs/codegraph/internal/store"
)

// EmbedStage generates vector embeddings for symbols via AWS Bedrock.
type EmbedStage struct {
	client *embedding.Client
	store  *store.Store
	logger *slog.Logger
}

func NewEmbedStage(client *embedding.Client, s *store.Store, logger *slog.Logger) *EmbedStage {
	return &EmbedStage{client: client, store: s, logger: logger}
}

func (s *EmbedStage) Name() string { return "embed" }

func (s *EmbedStage) Execute(ctx context.Context, rc *IndexRunContext) error {
	count, err := embedding.EmbedSymbols(ctx, s.client, s.store, rc.ProjectID, s.logger)
	if err != nil {
		return fmt.Errorf("embed symbols: %w", err)
	}

	s.logger.Info("embedded symbols", slog.Int("count", count))
	return nil
}
