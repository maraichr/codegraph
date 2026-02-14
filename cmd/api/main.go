package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/codegraph-labs/codegraph/internal/api"
	"github.com/codegraph-labs/codegraph/internal/config"
	"github.com/codegraph-labs/codegraph/internal/embedding"
	"github.com/codegraph-labs/codegraph/internal/graph"
	"github.com/codegraph-labs/codegraph/internal/impact"
	"github.com/codegraph-labs/codegraph/internal/ingestion"
	"github.com/codegraph-labs/codegraph/internal/lineage"
	"github.com/codegraph-labs/codegraph/internal/store"
	minioclient "github.com/codegraph-labs/codegraph/internal/store/minio"
	"github.com/codegraph-labs/codegraph/internal/store/postgres"
	vk "github.com/codegraph-labs/codegraph/internal/store/valkey"
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

	// Initialize database pool
	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, cfg.Database.DSN(), cfg.Database.MaxConns, cfg.Database.MinConns)
	if err != nil {
		logger.Error("failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()
	logger.Info("connected to database")

	s := store.New(pool)

	deps := &api.RouterDeps{}

	// Neo4j (optional)
	graphClient, err := graph.NewClient(cfg.Neo4j)
	if err != nil {
		logger.Warn("neo4j connection failed, lineage queries disabled", slog.String("error", err.Error()))
	} else {
		deps.Graph = graphClient
		deps.Lineage = lineage.NewEngine(s, graphClient, logger)
		deps.Impact = impact.NewEngine(graphClient, s, logger)
		defer graphClient.Close(ctx)
		logger.Info("connected to neo4j")
	}

	// MinIO (optional — enables uploads)
	mc, err := minioclient.NewClient(cfg.MinIO)
	if err != nil {
		logger.Warn("minio connection failed, uploads disabled", slog.String("error", err.Error()))
	} else {
		deps.MinIO = mc
		logger.Info("connected to minio")
	}

	// Valkey (optional — enables job queue)
	vkClient, err := vk.NewClient(cfg.Valkey)
	if err != nil {
		logger.Warn("valkey connection failed, job queue disabled", slog.String("error", err.Error()))
	} else {
		deps.Producer = ingestion.NewProducer(vkClient)
		defer vkClient.Close()
		logger.Info("connected to valkey")
	}

	// Embeddings (auto-selects: OpenRouter > Bedrock > disabled)
	embedder, err := embedding.NewEmbedder(cfg)
	if err != nil {
		logger.Warn("embedder init failed, semantic search disabled", slog.String("error", err.Error()))
	} else if embedder != nil {
		deps.Embed = embedder
		logger.Info("embeddings enabled", slog.String("provider", fmt.Sprintf("%T", embedder)), slog.String("model", embedder.ModelID()))
	}

	router := api.NewRouter(logger, s, deps)

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("starting API server", slog.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", slog.String("error", err.Error()))
	}

	logger.Info("server stopped")
}
