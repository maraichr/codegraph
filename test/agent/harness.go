//go:build integration

package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/maraichr/lattice/internal/mcp/session"
	"github.com/maraichr/lattice/internal/store"
)

const maxTurns = 15

// ToolFunc dispatches a tool call and returns the string result.
type ToolFunc func(ctx context.Context, argsJSON json.RawMessage) (string, error)

// Harness drives an LLM agent loop with tool use against live MCP handlers.
type Harness struct {
	apiKey   string
	model    string
	baseURL  string
	store    *store.Store
	session  *session.Manager
	tools    []openaiTool
	dispatch map[string]ToolFunc
	logger   *slog.Logger
	http     *http.Client
}

// EvalResult captures metrics from a single agent evaluation run.
type EvalResult struct {
	Question     string
	FinalAnswer  string
	ToolCalls    int
	TotalTokens  int
	Turns        int
	ToolSequence []string
}

// HarnessConfig holds the configuration for creating a Harness.
type HarnessConfig struct {
	APIKey  string
	Model   string
	BaseURL string
	Store   *store.Store
	Session *session.Manager
	Logger  *slog.Logger
}

// NewHarness creates a new evaluation harness.
func NewHarness(cfg HarnessConfig) *Harness {
	tools, dispatch := buildToolsAndDispatch(cfg.Store, cfg.Session, cfg.Logger)
	return &Harness{
		apiKey:   cfg.APIKey,
		model:    cfg.Model,
		baseURL:  cfg.BaseURL,
		store:    cfg.Store,
		session:  cfg.Session,
		tools:    tools,
		dispatch: dispatch,
		logger:   cfg.Logger,
		http:     &http.Client{},
	}
}

// --- OpenAI-compatible request/response types ---

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Tools    []openaiTool  `json:"tools,omitempty"`
}

type chatMessage struct {
	Role       string          `json:"role"`
	Content    string          `json:"content,omitempty"`
	ToolCalls  []toolCall      `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
}

type openaiTool struct {
	Type     string       `json:"type"`
	Function toolFunction `json:"function"`
}

type toolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type toolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type chatResponse struct {
	Choices []struct {
		Message      chatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Run executes the agent loop: send question, dispatch tool calls, repeat until answer or cap.
func (h *Harness) Run(ctx context.Context, question string) (*EvalResult, error) {
	result := &EvalResult{Question: question}

	systemPrompt := `You are a code analysis assistant. You have access to tools that let you explore a codebase's symbol graph.
Use the tools to answer the user's question. The project slug is the identifier you pass to tools.

IMPORTANT RULES:
- Be concise in your final answer.
- When you have enough information to answer, STOP calling tools and provide your answer immediately.
- If a tool returns "No symbols found" or an empty result, do NOT retry with similar queries. Instead, try a different tool or answer with what you already know.
- The ask_codebase tool can answer overview/architecture questions directly — prefer it for broad questions.
- Prefer fewer tool calls. 1-3 calls should be enough for most questions.`

	messages := []chatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: question},
	}

	for turn := 0; turn < maxTurns; turn++ {
		result.Turns = turn + 1

		resp, err := h.chat(ctx, messages)
		if err != nil {
			return result, fmt.Errorf("chat turn %d: %w", turn, err)
		}

		result.TotalTokens += resp.Usage.TotalTokens

		if resp.Error != nil {
			return result, fmt.Errorf("API error: %s", resp.Error.Message)
		}

		if len(resp.Choices) == 0 {
			return result, fmt.Errorf("no choices in response")
		}

		choice := resp.Choices[0]
		messages = append(messages, choice.Message)

		// If no tool calls, we have the final answer
		if len(choice.Message.ToolCalls) == 0 || choice.FinishReason == "stop" {
			result.FinalAnswer = choice.Message.Content
			return result, nil
		}

		// Dispatch each tool call
		for _, tc := range choice.Message.ToolCalls {
			result.ToolCalls++
			result.ToolSequence = append(result.ToolSequence, tc.Function.Name)

			h.logger.Info("tool call",
				slog.String("name", tc.Function.Name),
				slog.String("args", tc.Function.Arguments),
			)

			fn, ok := h.dispatch[tc.Function.Name]
			if !ok {
				messages = append(messages, chatMessage{
					Role:       "tool",
					ToolCallID: tc.ID,
					Content:    fmt.Sprintf("Unknown tool: %s", tc.Function.Name),
				})
				continue
			}

			toolResult, err := fn(ctx, json.RawMessage(tc.Function.Arguments))
			if err != nil {
				toolResult = fmt.Sprintf("Error: %s", err.Error())
			}

			h.logger.Info("tool result",
				slog.String("name", tc.Function.Name),
				slog.Int("result_len", len(toolResult)),
			)

			messages = append(messages, chatMessage{
				Role:       "tool",
				ToolCallID: tc.ID,
				Content:    toolResult,
			})
		}
	}

	// Hit max turns — return what we have
	result.FinalAnswer = "(max turns reached)"
	return result, nil
}

func (h *Harness) chat(ctx context.Context, messages []chatMessage) (*chatResponse, error) {
	reqBody, err := json.Marshal(chatRequest{
		Model:    h.model,
		Messages: messages,
		Tools:    h.tools,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.baseURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+h.apiKey)

	resp, err := h.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result chatResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}
