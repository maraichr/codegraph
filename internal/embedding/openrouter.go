package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/maraichr/codegraph/internal/config"
)

const (
	defaultOpenRouterModel   = "openai/text-embedding-3-small"
	defaultOpenRouterBaseURL = "https://openrouter.ai/api/v1/embeddings"
	defaultDimensions        = 1024
)

// OpenRouterClient implements Embedder using the OpenAI-compatible OpenRouter API.
type OpenRouterClient struct {
	apiKey     string
	model      string
	baseURL    string
	dimensions int
	http       *http.Client
}

// NewOpenRouterClient creates a new OpenRouter embedding client.
func NewOpenRouterClient(cfg config.OpenRouterConfig) (*OpenRouterClient, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("OPENROUTER_API_KEY is required")
	}

	model := cfg.Model
	if model == "" {
		model = defaultOpenRouterModel
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultOpenRouterBaseURL
	}

	dimensions := cfg.Dimensions
	if dimensions <= 0 {
		dimensions = defaultDimensions
	}

	return &OpenRouterClient{
		apiKey:     cfg.APIKey,
		model:      model,
		baseURL:    baseURL,
		dimensions: dimensions,
		http:       &http.Client{},
	}, nil
}

type openAIEmbedRequest struct {
	Model      string   `json:"model"`
	Input      []string `json:"input"`
	Dimensions int      `json:"dimensions,omitempty"`
}

type openAIEmbedResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// EmbedBatch generates embeddings for a batch of texts via OpenRouter.
func (c *OpenRouterClient) EmbedBatch(ctx context.Context, texts []string, inputType string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	reqBody, err := json.Marshal(openAIEmbedRequest{
		Model:      c.model,
		Input:      texts,
		Dimensions: c.dimensions,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openrouter API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result openAIEmbedResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("openrouter error: %s", result.Error.Message)
	}

	embeddings := make([][]float32, len(result.Data))
	for _, d := range result.Data {
		embeddings[d.Index] = d.Embedding
	}

	return embeddings, nil
}

// ModelID returns the model identifier.
func (c *OpenRouterClient) ModelID() string {
	return c.model
}
