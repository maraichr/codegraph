package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	defaultBaseURL    = "https://openrouter.ai/api/v1/chat/completions"
	defaultModel      = "minimax/minimax-m1"
	maxRetries        = 3
	retryDelay        = 2 * time.Second
	defaultMaxTokens  = 4096
	defaultTemperature = 0.0
)

// Client is a lightweight OpenAI-compatible chat completions client.
type Client struct {
	apiKey  string
	model   string
	baseURL string
	http    *http.Client
}

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `json:"temperature"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// NewClient creates a new LLM chat client.
func NewClient(apiKey, model, baseURL string) *Client {
	if model == "" {
		model = defaultModel
	}
	if baseURL == "" {
		baseURL = defaultBaseURL
	} else {
		baseURL = strings.TrimRight(baseURL, "/")
		if !strings.HasSuffix(baseURL, "/chat/completions") {
			baseURL += "/chat/completions"
		}
	}
	return &Client{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// Complete sends messages to the LLM and returns the response content.
func (c *Client) Complete(ctx context.Context, messages []Message) (string, error) {
	payload := chatRequest{
		Model:       c.model,
		Messages:    messages,
		MaxTokens:   defaultMaxTokens,
		Temperature: defaultTemperature,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(retryDelay * time.Duration(attempt)):
			}
		}

		result, err := c.doRequest(ctx, body)
		if err == nil {
			return result, nil
		}
		lastErr = err
		errStr := err.Error()
		if !strings.Contains(errStr, "status 429") &&
			!strings.Contains(errStr, "status 529") &&
			!strings.Contains(errStr, "status 503") {
			return "", err
		}
	}
	return "", fmt.Errorf("after %d retries: %w", maxRetries, lastErr)
}

func (c *Client) doRequest(ctx context.Context, body []byte) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("LLM API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result chatResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("LLM error: %s", result.Error.Message)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("LLM returned no choices")
	}

	return strings.TrimSpace(result.Choices[0].Message.Content), nil
}

// Model returns the model identifier.
func (c *Client) Model() string {
	return c.model
}
