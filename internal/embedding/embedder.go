package embedding

import (
	"context"
	"fmt"

	"github.com/maraichr/lattice/internal/config"
)

// Embedder is the interface for embedding providers.
type Embedder interface {
	EmbedBatch(ctx context.Context, texts []string, inputType string) ([][]float32, error)
	ModelID() string
}

// NewEmbedder auto-selects provider: OpenRouter (if API key set) > Bedrock (if region set) > nil.
func NewEmbedder(cfg *config.Config) (Embedder, error) {
	if cfg.OpenRouter.APIKey != "" {
		client, err := NewOpenRouterClient(cfg.OpenRouter)
		if err != nil {
			return nil, fmt.Errorf("openrouter client: %w", err)
		}
		return client, nil
	}

	if cfg.Bedrock.Region != "" {
		client, err := NewClient(cfg.Bedrock)
		if err != nil {
			return nil, fmt.Errorf("bedrock client: %w", err)
		}
		return client, nil
	}

	return nil, nil
}
