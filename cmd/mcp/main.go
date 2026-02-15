package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/maraichr/codegraph/internal/auth"
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

	// SDK MCP server and Streamable HTTP transport (only extract_subgraph and ask_codebase
	// are registered; other spec tools are used internally by ask_codebase or are future work).
	sdkServer := sdkmcp.NewServer(&sdkmcp.Implementation{Name: "codegraph", Version: "1.0.0"}, nil)

	sdkmcp.AddTool(sdkServer, &sdkmcp.Tool{
		Name:        "extract_subgraph",
		Description: "Extract a subgraph of symbols and relationships around a topic or set of seed symbols. Returns symbol cards with metadata, edges, and navigation hints.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, params *tools.ExtractSubgraphParams) (*sdkmcp.CallToolResult, any, error) {
		if params == nil {
			params = &tools.ExtractSubgraphParams{}
		}
		result, err := extractSubgraph.Handle(ctx, *params)
		if err != nil {
			return &sdkmcp.CallToolResult{
				IsError: true,
				Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: err.Error()}},
			}, nil, nil
		}
		return &sdkmcp.CallToolResult{
			Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: result}},
		}, nil, nil
	})

	sdkmcp.AddTool(sdkServer, &sdkmcp.Tool{
		Name:        "ask_codebase",
		Description: "Ask a natural language question about the codebase. Routes to overview, search, ranking, impact analysis, lineage tracing, or subgraph exploration.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, params *tools.AskCodebaseParams) (*sdkmcp.CallToolResult, any, error) {
		if params == nil {
			params = &tools.AskCodebaseParams{}
		}
		result, err := askCodebase.Handle(ctx, *params)
		if err != nil {
			return &sdkmcp.CallToolResult{
				IsError: true,
				Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: err.Error()}},
			}, nil, nil
		}
		return &sdkmcp.CallToolResult{
			Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: result}},
		}, nil, nil
	})

	sdkHandler := sdkmcp.NewStreamableHTTPHandler(func(*http.Request) *sdkmcp.Server { return sdkServer }, nil)

	// Wrap with auth middleware
	var mcpHandler http.Handler = sdkHandler
	if cfg.Auth.Enabled {
		if cfg.Auth.IssuerURL == "" {
			logger.Error("AUTH_ENABLED=true but AUTH_ISSUER_URL is empty")
			os.Exit(1)
		}
		verifier, err := auth.NewVerifier(ctx, cfg.Auth.IssuerURL, cfg.Auth.PublicIssuer, cfg.Auth.Audience)
		if err != nil {
			logger.Error("failed to init OIDC verifier for MCP", slog.String("error", err.Error()))
			os.Exit(1)
		}
		mcpHandler = auth.RequireAuth(verifier, logger)(sdkHandler)
		logger.Info("MCP OIDC auth enabled", slog.String("issuer", cfg.Auth.IssuerURL))
	} else {
		mcpHandler = auth.DevModeMiddleware(logger)(sdkHandler)
	}

	httpServer := &http.Server{Addr: cfg.MCP.Addr, Handler: mcpHandler}

	go func() {
		logger.Info("MCP server listening", slog.String("addr", cfg.MCP.Addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("MCP HTTP server error", slog.String("error", err.Error()))
		}
	}()

	<-ctx.Done()
	logger.Info("MCP server shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Warn("MCP HTTP shutdown", slog.String("error", err.Error()))
	}
	logger.Info("MCP server stopped")
}
