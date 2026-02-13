package mcp

import (
	"log/slog"
)

// Server implements the MCP (Model Context Protocol) server
// using Streamable HTTP transport (SSE deprecated March 2025).
type Server struct {
	logger *slog.Logger
}

// NewServer creates a new MCP server instance.
func NewServer(logger *slog.Logger) *Server {
	return &Server{logger: logger}
}

// TODO: Implement MCP tools:
// - search_symbols: Semantic code search across project
// - get_lineage: Data lineage tracing
// - get_symbol: Get symbol details with context
// - list_projects: List available projects
// - get_dependencies: Get dependency graph for a symbol
//
// TODO: Implement MCP resources:
// - project://{slug} — Project metadata
// - symbol://{id} — Symbol details
// - file://{project}/{path} — File content with annotations
