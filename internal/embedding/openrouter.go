package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/maraichr/codegraph/internal/config"
)

const (
	defaultOpenRouterModel   = "openai/text-embedding-3-small"
	defaultOpenRouterBaseURL = "https://openrouter.ai/api/v1/embeddings"
	defaultDimensions        = 1024
	openRouterMaxRetries     = 3
	openRouterRetryDelay     = 2 * time.Second
	openRouterBatchSize      = 100 // avoid huge responses that get truncated or time out
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

	baseURL := cfg.BaseURLEmbeddings
	if baseURL == "" {
		baseURL = cfg.BaseURL
	}
	if baseURL == "" {
		baseURL = defaultOpenRouterBaseURL
	} else {
		baseURL = strings.TrimRight(baseURL, "/")
		// If set to OpenRouter site root or /api/v1 without /embeddings, use the embeddings endpoint
		if baseURL == "https://openrouter.ai" || baseURL == "https://openrouter.ai/api/v1" {
			baseURL = defaultOpenRouterBaseURL
		}
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

type openRouterProvider struct {
	AllowFallbacks bool `json:"allow_fallbacks"`
}

type openAIEmbedRequest struct {
	Model          string              `json:"model"`
	Input          []string            `json:"input"`
	Dimensions     int                 `json:"dimensions,omitempty"`
	EncodingFormat string              `json:"encoding_format,omitempty"` // "float" (default) or "base64"; some models (e.g. Codestral) expect it
	Provider       *openRouterProvider `json:"provider,omitempty"`
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
// Splits into sub-batches of openRouterBatchSize to avoid huge responses that get truncated or time out.
func (c *OpenRouterClient) EmbedBatch(ctx context.Context, texts []string, inputType string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	var allEmbeddings [][]float32
	for i := 0; i < len(texts); i += openRouterBatchSize {
		end := min(i+openRouterBatchSize, len(texts))
		batch := texts[i:end]

		payload := openAIEmbedRequest{
			Model:          c.model,
			Input:          batch,
			EncodingFormat: "float",
			Provider:       &openRouterProvider{AllowFallbacks: true},
		}
		if strings.HasPrefix(c.model, "openai/") || strings.HasPrefix(c.model, "qwen/") {
			payload.Dimensions = c.dimensions
		}
		reqBody, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}

		var lastErr error
		for attempt := 0; attempt < openRouterMaxRetries; attempt++ {
			if attempt > 0 {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(openRouterRetryDelay * time.Duration(attempt)):
				}
			}

			embeddings, err := c.doEmbedRequest(ctx, reqBody)
			if err == nil {
				allEmbeddings = append(allEmbeddings, embeddings...)
				break
			}
			lastErr = err
			errStr := err.Error()
			if !strings.Contains(errStr, "No successful provider responses") &&
				!strings.Contains(errStr, "status 529") &&
				!strings.Contains(errStr, "Provider Overloaded") &&
				!strings.Contains(errStr, "empty response") &&
				!strings.Contains(errStr, "unexpected end of JSON") {
				return nil, err
			}
		}
		if len(allEmbeddings) < end {
			return nil, fmt.Errorf("batch %d: %w", i/openRouterBatchSize, lastErr)
		}
	}
	return allEmbeddings, nil
}

func (c *OpenRouterClient) doEmbedRequest(ctx context.Context, reqBody []byte) ([][]float32, error) {
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

	// Often HTML when base URL is wrong, auth fails, or a proxy returns an error page
	if len(body) > 0 && body[0] == '<' {
		snippet := string(body)
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}
		return nil, fmt.Errorf("embedding API returned HTML instead of JSON: check OPENROUTER_BASE_URL (use https://openrouter.ai/api/v1/embeddings) and OPENROUTER_API_KEY; body: %s", snippet)
	}

	// Empty or whitespace-only body: connection closed, timeout, or response truncated (try smaller batch)
	if len(bytes.TrimSpace(body)) == 0 {
		return nil, fmt.Errorf("embedding API returned empty response (connection closed, timeout, or response truncated; batches are limited to %d texts)", openRouterBatchSize)
	}

	var result openAIEmbedResponse
	if err := json.Unmarshal(body, &result); err != nil {
		snippet := string(bytes.TrimSpace(body))
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}
		if len(snippet) == 0 {
			snippet = "(empty)"
		}
		return nil, fmt.Errorf("unmarshal response: %w; body len=%d: %s", err, len(body), snippet)
	}

	if result.Error != nil {
		msg := result.Error.Message
		if strings.Contains(msg, "No successful provider responses") {
			msg += " (model may not be available for embeddings on OpenRouter; try OPENROUTER_MODEL=openai/text-embedding-3-small)"
		}
		return nil, fmt.Errorf("openrouter error: %s", msg)
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
