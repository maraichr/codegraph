package api

import (
	"log/slog"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	apihandler "github.com/codegraph-labs/codegraph/internal/api/handler"
	"github.com/codegraph-labs/codegraph/internal/api/graphql"
	apimw "github.com/codegraph-labs/codegraph/internal/api/middleware"
	"github.com/codegraph-labs/codegraph/internal/embedding"
	"github.com/codegraph-labs/codegraph/internal/graph"
	"github.com/codegraph-labs/codegraph/internal/impact"
	"github.com/codegraph-labs/codegraph/internal/ingestion"
	"github.com/codegraph-labs/codegraph/internal/lineage"
	minioclient "github.com/codegraph-labs/codegraph/internal/store/minio"
	"github.com/codegraph-labs/codegraph/internal/store"
)

// RouterDeps holds optional dependencies for the router.
type RouterDeps struct {
	MinIO   *minioclient.Client
	Producer *ingestion.Producer
	Graph   *graph.Client
	Embed   embedding.Embedder
	Lineage *lineage.Engine
	Impact  *impact.Engine
}

func NewRouter(logger *slog.Logger, s *store.Store, deps *RouterDeps) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(apimw.Logger(logger))
	r.Use(apimw.CORS)
	r.Use(chimw.Recoverer)

	// Health checks
	health := apihandler.NewHealthHandler(s.Pool())
	r.Get("/healthz", health.Healthz)
	r.Get("/readyz", health.Readyz)

	if deps == nil {
		deps = &RouterDeps{}
	}

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		projects := apihandler.NewProjectHandler(logger, s)
		r.Route("/projects", func(r chi.Router) {
			r.Get("/", projects.List)
			r.Post("/", projects.Create)
			r.Route("/{slug}", func(r chi.Router) {
				r.Get("/", projects.Get)
				r.Put("/", projects.Update)
				r.Delete("/", projects.Delete)

				sources := apihandler.NewSourceHandler(logger, s)
				r.Route("/sources", func(r chi.Router) {
					r.Get("/", sources.List)
					r.Post("/", sources.Create)
					r.Route("/{sourceID}", func(r chi.Router) {
						r.Get("/", sources.Get)
						r.Delete("/", sources.Delete)
					})
				})

				indexRuns := apihandler.NewIndexRunHandler(logger, s, deps.Producer)
				r.Route("/index-runs", func(r chi.Router) {
					r.Get("/", indexRuns.List)
					r.Post("/", indexRuns.Trigger)
					r.Get("/{runID}", indexRuns.Get)
				})

				// Symbol search within project
				symbolsInProject := apihandler.NewSymbolHandler(logger, s, deps.Graph, deps.Lineage, deps.Impact)
				r.Get("/symbols", symbolsInProject.Search)

				// Semantic search within project
				search := apihandler.NewSearchHandler(logger, s, deps.Embed)
				r.Post("/search/semantic", search.Semantic)

				// Upload (requires MinIO)
				if deps.MinIO != nil {
					upload := apihandler.NewUploadHandler(logger, s, deps.MinIO, deps.Producer)
					r.Post("/upload", upload.Upload)
				}
			})
		})

		// Symbols
		symbols := apihandler.NewSymbolHandler(logger, s, deps.Graph, deps.Lineage, deps.Impact)
		r.Route("/symbols", func(r chi.Router) {
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", symbols.Get)
				r.Get("/references", symbols.References)
				r.Get("/lineage", symbols.Lineage)
				r.Get("/impact", symbols.Impact)
				r.Get("/column-lineage", symbols.ColumnLineage)
			})
		})

		// Webhooks
		webhooks := apihandler.NewWebhookHandler(logger, s, deps.Producer)
		r.Post("/webhooks/gitlab/{sourceID}", webhooks.GitLabPush)
	})

	// GraphQL
	gqlResolver := graphql.NewResolver(logger, s, deps.Graph, deps.Embed, deps.Lineage, deps.Impact)
	gqlSrv := handler.New(graphql.NewExecutableSchema(graphql.Config{Resolvers: gqlResolver}))
	gqlSrv.SetErrorPresenter(graphql.ErrorPresenter())
	gqlSrv.AddTransport(transport.POST{})
	gqlSrv.Use(extension.Introspection{})
	r.Handle("/graphql", gqlSrv)
	r.Get("/graphql/playground", playground.Handler("CodeGraph", "/graphql"))

	return r
}
