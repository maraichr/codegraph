package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/maraichr/codegraph/internal/config"
	"github.com/maraichr/codegraph/internal/mcp"
	"github.com/maraichr/codegraph/internal/mcp/tools"
	"github.com/maraichr/codegraph/internal/store"
	"github.com/maraichr/codegraph/internal/store/postgres"
	vk "github.com/maraichr/codegraph/internal/store/valkey"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Database
	pool, err := postgres.NewPool(ctx, cfg.Database.DSN(), cfg.Database.MaxConns, cfg.Database.MinConns)
	if err != nil {
		logger.Error("failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()
	logger.Info("connected to database")

	s := store.New(pool)

	// Valkey (optional for sessions)
	vkClient, err := vk.NewClient(cfg.Valkey)
	if err != nil {
		logger.Warn("valkey unavailable, sessions disabled", slog.String("error", err.Error()))
	} else {
		defer vkClient.Close()
		logger.Info("connected to valkey")
	}

	// Create MCP server with infrastructure
	mcpServer := mcp.NewServer(mcp.ServerDeps{
		Store:        s,
		ValkeyClient: vkClient,
		Logger:       logger,
	})

	// Wire tool handlers (in cmd to avoid import cycle mcp <-> mcp/tools)
	extractSubgraph := tools.NewExtractSubgraphHandler(s, mcpServer.Session, logger)
	askCodebase := tools.NewAskCodebaseHandler(s, mcpServer.Session, logger)

	// Suppress unused warnings — these will be registered with the MCP transport
	_ = extractSubgraph
	_ = askCodebase

	logger.Info("starting MCP server (Streamable HTTP transport)")

	// TODO: Start Streamable HTTP transport listener
	// Register all tool handlers with the transport:
	// - search_symbols, get_symbol_details, get_dependencies,
	//   trace_lineage, analyze_impact, get_file_contents, query_graph,
	//   list_project_overview, find_usages, compare_snapshots,
	//   extract_subgraph, ask_codebase

	<-ctx.Done()
	logger.Info("MCP server stopped")
}
