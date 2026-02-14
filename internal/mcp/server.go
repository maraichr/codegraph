package mcp

import (
	"log/slog"

	"github.com/valkey-io/valkey-go"

	"github.com/codegraph-labs/codegraph/internal/mcp/session"
	"github.com/codegraph-labs/codegraph/internal/store"
)

// Server implements the MCP (Model Context Protocol) server
// using Streamable HTTP transport (SSE deprecated March 2025).
type Server struct {
	Store    *store.Store
	Session  *session.Manager
	Nav      *Navigator
	Logger   *slog.Logger
}

// ServerDeps holds dependencies for the MCP server.
type ServerDeps struct {
	Store        *store.Store
	ValkeyClient valkey.Client
	Logger       *slog.Logger
}

// NewServer creates a new MCP server with session and navigation infrastructure.
func NewServer(deps ServerDeps) *Server {
	var sm *session.Manager
	if deps.ValkeyClient != nil {
		sm = session.NewManager(deps.ValkeyClient)
	}

	nav := NewNavigator(deps.Store.Queries)

	return &Server{
		Store:   deps.Store,
		Session: sm,
		Nav:     nav,
		Logger:  deps.Logger,
	}
}
