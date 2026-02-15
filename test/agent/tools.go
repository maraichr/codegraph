//go:build integration

package agent

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/maraichr/lattice/internal/mcp/session"
	"github.com/maraichr/lattice/internal/mcp/tools"
	"github.com/maraichr/lattice/internal/store"
)

// buildToolsAndDispatch returns the OpenAI tool schemas and a dispatch map for the eval harness.
func buildToolsAndDispatch(s *store.Store, sm *session.Manager, logger *slog.Logger) ([]openaiTool, map[string]ToolFunc) {
	subgraphHandler := tools.NewExtractSubgraphHandler(s, sm, nil, logger)
	askHandler := tools.NewAskCodebaseHandler(s, sm, nil, logger)

	schemas := []openaiTool{
		{
			Type: "function",
			Function: toolFunction{
				Name:        "extract_subgraph",
				Description: "Extract a subgraph of symbols and relationships around a topic or set of seed symbols. Returns symbol cards with metadata, edges, and navigation hints.",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"project": {
							"type": "string",
							"description": "Project slug identifier"
						},
						"topic": {
							"type": "string",
							"description": "Symbol name or partial name to search for (e.g. 'Users', 'GetCustomer', 'Repository'). NOT natural language — use actual symbol names."
						},
						"kinds": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Filter seed symbols by kind: table, procedure, class, method, function, column, interface, field, property, enum"
						},
						"seed_symbols": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Explicit symbol UUIDs to use as BFS seeds"
						},
						"max_depth": {
							"type": "integer",
							"description": "Maximum BFS depth (default 2)"
						},
						"max_nodes": {
							"type": "integer",
							"description": "Maximum symbols to return (default 50)"
						},
						"verbosity": {
							"type": "string",
							"enum": ["summary", "normal", "full"],
							"description": "Level of detail in symbol cards"
						},
						"session_id": {
							"type": "string",
							"description": "Session ID for deduplication and navigation context"
						}
					},
					"required": ["project"]
				}`),
			},
		},
		{
			Type: "function",
			Function: toolFunction{
				Name:        "ask_codebase",
				Description: "Ask a natural language question about the codebase. Routes to the appropriate analysis: overview, search, ranking (most used/important), impact analysis, lineage tracing, or subgraph exploration. Supports filtering by symbol kind (table, procedure, class, method, function, column, interface, field, property, enum) and language.",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"project": {
							"type": "string",
							"description": "Project slug identifier"
						},
						"question": {
							"type": "string",
							"description": "Natural language question about the codebase"
						},
						"kinds": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Filter by symbol kinds: table, procedure, class, method, function, column, interface, field, property, enum"
						},
						"languages": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Filter by languages: csharp, tsql, javascript, typescript, go, java, etc."
						},
						"verbosity": {
							"type": "string",
							"enum": ["summary", "normal", "full"],
							"description": "Level of detail in response"
						},
						"session_id": {
							"type": "string",
							"description": "Session ID for context continuity"
						}
					},
					"required": ["project", "question"]
				}`),
			},
		},
	}

	dispatch := map[string]ToolFunc{
		"extract_subgraph": func(ctx context.Context, argsJSON json.RawMessage) (string, error) {
			var params tools.ExtractSubgraphParams
			if err := json.Unmarshal(argsJSON, &params); err != nil {
				return "", err
			}
			return subgraphHandler.Handle(ctx, params)
		},
		"ask_codebase": func(ctx context.Context, argsJSON json.RawMessage) (string, error) {
			var params tools.AskCodebaseParams
			if err := json.Unmarshal(argsJSON, &params); err != nil {
				return "", err
			}
			return askHandler.Handle(ctx, params)
		},
	}

	return schemas, dispatch
}
