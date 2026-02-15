package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/maraichr/codegraph/internal/analytics"
	"github.com/maraichr/codegraph/internal/config"
	"github.com/maraichr/codegraph/internal/embedding"
	"github.com/maraichr/codegraph/internal/graph"
	"github.com/maraichr/codegraph/internal/ingestion"
	"github.com/maraichr/codegraph/internal/ingestion/connectors"
	"github.com/maraichr/codegraph/internal/lineage"
	"github.com/maraichr/codegraph/internal/parser"
	"github.com/maraichr/codegraph/internal/parser/asp"
	"github.com/maraichr/codegraph/internal/parser/delphi"
	csharpp "github.com/maraichr/codegraph/internal/parser/csharp"
	javap "github.com/maraichr/codegraph/internal/parser/java"
	jsts "github.com/maraichr/codegraph/internal/parser/javascript"
	"github.com/maraichr/codegraph/internal/parser/pgsql"
	"github.com/maraichr/codegraph/internal/parser/tsql"
	"github.com/maraichr/codegraph/internal/resolver"
	"github.com/maraichr/codegraph/internal/store"
	minioclient "github.com/maraichr/codegraph/internal/store/minio"
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

	// Valkey
	vkClient, err := vk.NewClient(cfg.Valkey)
	if err != nil {
		logger.Error("failed to connect to valkey", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer vkClient.Close()
	logger.Info("connected to valkey")

	// MinIO
	minioClient, err := minioclient.NewClient(cfg.MinIO)
	if err != nil {
		logger.Error("failed to connect to minio", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("connected to minio")

	// Neo4j
	graphClient, err := graph.NewClient(cfg.Neo4j)
	if err != nil {
		logger.Error("failed to connect to neo4j", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer graphClient.Close(ctx)
	if err := graphClient.EnsureIndexes(ctx); err != nil {
		logger.Warn("neo4j ensure indexes failed, sync may be slow", slog.String("error", err.Error()))
	}
	logger.Info("connected to neo4j")

	// Connectors
	zipConn := connectors.NewZipConnector(minioClient)
	gitConn := connectors.NewGitLabConnector()

	// S3 connector (optional)
	var s3Conn *connectors.S3Connector
	if cfg.S3.Bucket != "" {
		s3Conn, err = connectors.NewS3Connector(cfg.S3)
		if err != nil {
			logger.Warn("s3 connector init failed", slog.String("error", err.Error()))
		} else {
			logger.Info("s3 connector enabled", slog.String("bucket", cfg.S3.Bucket))
		}
	}

	// Parser registry
	registry := parser.NewRegistry()
	sqlRouter := parser.NewSQLRouter(tsql.New(), pgsql.New())
	registry.Register(".sql", sqlRouter)
	registry.Register(".sqldataprovider", sqlRouter)
	aspParser := asp.New()
	registry.Register(".asp", aspParser)
	registry.Register(".aspx", aspParser)
	registry.Register(".ascx", aspParser)
	registry.Register(".ashx", aspParser)
	registry.Register(".master", aspParser)
	delphiParser := delphi.New()
	registry.Register(".pas", delphiParser)
	registry.Register(".dfm", delphiParser)
	registry.Register(".dpr", delphiParser)
	registry.Register(".java", javap.New())
	registry.Register(".cs", csharpp.New())
	jsParser := jsts.NewJS()
	registry.Register(".js", jsParser)
	registry.Register(".jsx", jsParser)
	registry.Register(".mjs", jsParser)
	tsParser := jsts.NewTS()
	registry.Register(".ts", tsParser)
	registry.Register(".tsx", tsParser)

	// Embeddings (auto-selects: OpenRouter > Bedrock > disabled)
	var embedStage ingestion.Stage
	embedder, err := embedding.NewEmbedder(cfg)
	if err != nil {
		logger.Warn("embedder init failed, embedding stage disabled", slog.String("error", err.Error()))
		embedStage = ingestion.NewNoOpStage("embed")
	} else if embedder != nil {
		embedStage = ingestion.NewEmbedStage(embedder, s, logger)
		logger.Info("embeddings enabled", slog.String("provider", fmt.Sprintf("%T", embedder)), slog.String("model", embedder.ModelID()))
	} else {
		embedStage = ingestion.NewNoOpStage("embed")
	}

	// Resolver engine
	resolverEngine := resolver.NewEngine(s, logger)

	// Lineage engine
	lineageEngine := lineage.NewEngine(s, graphClient, logger)

	// Analytics engine (degree, PageRank, layers, summaries, bridges)
	analyticsEngine := analytics.NewEngine(s, logger)

	// Pipeline stages
	stages := []ingestion.Stage{
		ingestion.NewCloneStage(s, zipConn, gitConn, s3Conn),
		ingestion.NewParseStage(registry, s),
		ingestion.NewResolveStage(resolverEngine),
		ingestion.NewLineageStage(lineageEngine, logger),
		ingestion.NewGraphStage(s, graphClient, logger),
		embedStage,
		ingestion.NewAnalyticsStage(analyticsEngine, logger),
	}

	pipeline := ingestion.NewPipeline(s, stages, logger)

	// Consumer
	consumer := ingestion.NewConsumer(vkClient, "worker-1", logger)
	if err := consumer.EnsureGroup(ctx); err != nil {
		logger.Error("failed to ensure consumer group", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Info("starting worker, consuming from stream", slog.String("stream", ingestion.StreamName))

	if err := consumer.Consume(ctx, pipeline.Run); err != nil {
		if ctx.Err() != nil {
			logger.Info("worker stopped by signal")
		} else {
			logger.Error("consumer error", slog.String("error", err.Error()))
		}
	}

	logger.Info("worker stopped")
}
