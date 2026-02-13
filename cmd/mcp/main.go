package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger.Info("starting MCP server (Streamable HTTP transport)")

	// TODO: Initialize MCP server with:
	// - Streamable HTTP transport (NOT SSE — deprecated March 2025)
	// - Tool registration (search_symbols, get_lineage, etc.)
	// - Resource templates (project://, symbol://)
	// - PostgreSQL + Neo4j backends

	<-ctx.Done()
	logger.Info("MCP server stopped")
}
