package api

import (
	"log/slog"
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	apihandler "github.com/maraichr/lattice/internal/api/handler"
	"github.com/maraichr/lattice/internal/api/graphql"
	apimw "github.com/maraichr/lattice/internal/api/middleware"
	"github.com/maraichr/lattice/internal/auth"
	"github.com/maraichr/lattice/internal/embedding"
	"github.com/maraichr/lattice/internal/graph"
	"github.com/maraichr/lattice/internal/impact"
	"github.com/maraichr/lattice/internal/ingestion"
	"github.com/maraichr/lattice/internal/lineage"
	"github.com/maraichr/lattice/internal/oracle"
	minioclient "github.com/maraichr/lattice/internal/store/minio"
	"github.com/maraichr/lattice/internal/store"
)

// RouterDeps holds optional dependencies for the router.
type RouterDeps struct {
	MinIO       *minioclient.Client
	Producer    *ingestion.Producer
	Graph       *graph.Client
	Embed       embedding.Embedder
	Lineage     *lineage.Engine
	Impact      *impact.Engine
	Oracle      *oracle.Engine
	Verifier    *auth.Verifier
	AuthEnabled bool
}

func NewRouter(logger *slog.Logger, s *store.Store, deps *RouterDeps) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(apimw.Logger(logger))
	r.Use(apimw.CORS)
	r.Use(chimw.Recoverer)

	// Health checks — always unauthenticated
	health := apihandler.NewHealthHandler(s.Pool())
	r.Get("/healthz", health.Healthz)
	r.Get("/readyz", health.Readyz)

	if deps == nil {
		deps = &RouterDeps{}
	}

	// Select auth middleware
	authHandler := selectAuthMiddleware(logger, deps)

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(authHandler)

		r.Route("/projects", func(r chi.Router) {
			projects := apihandler.NewProjectHandler(logger, s)

			r.With(auth.RequireScope("lattice:read")).Get("/", projects.List)
			r.With(auth.RequireScope("lattice:write")).Post("/", projects.Create)
			r.Route("/{slug}", func(r chi.Router) {
				r.With(auth.RequireScope("lattice:read")).Get("/", projects.Get)
				r.With(auth.RequireScope("lattice:write")).Put("/", projects.Update)
				r.With(auth.RequireScope("lattice:write")).Delete("/", projects.Delete)

				sources := apihandler.NewSourceHandler(logger, s)
				r.Route("/sources", func(r chi.Router) {
					r.With(auth.RequireScope("lattice:read")).Get("/", sources.List)
					r.With(auth.RequireScope("lattice:write")).Post("/", sources.Create)
					r.Route("/{sourceID}", func(r chi.Router) {
						r.With(auth.RequireScope("lattice:read")).Get("/", sources.Get)
						r.With(auth.RequireScope("lattice:write")).Delete("/", sources.Delete)
					})
				})

				indexRuns := apihandler.NewIndexRunHandler(logger, s, deps.Producer)
				r.Route("/index-runs", func(r chi.Router) {
					r.With(auth.RequireScope("lattice:read")).Get("/", indexRuns.List)
					r.With(auth.RequireScope("lattice:ingest")).Post("/", indexRuns.Trigger)
					r.With(auth.RequireScope("lattice:read")).Get("/{runID}", indexRuns.Get)
				})

				symbolsInProject := apihandler.NewSymbolHandler(logger, s, deps.Graph, deps.Lineage, deps.Impact)
				r.With(auth.RequireScope("lattice:read")).Get("/symbols", symbolsInProject.Search)

				search := apihandler.NewSearchHandler(logger, s, deps.Embed)
				r.With(auth.RequireScope("lattice:read")).Post("/search/semantic", search.Semantic)

				analytics := apihandler.NewAnalyticsHandler(logger, s)
				r.Route("/analytics", func(r chi.Router) {
					r.Use(auth.RequireScope("lattice:read"))
					r.Get("/summary", analytics.Summary)
					r.Get("/stats", analytics.Stats)
					r.Get("/languages", analytics.Languages)
					r.Get("/kinds", analytics.Kinds)
					r.Get("/layers", analytics.Layers)
					r.Get("/layers/{layer}", analytics.LayerSymbols)
					r.Get("/top/in-degree", analytics.TopByInDegree)
					r.Get("/top/pagerank", analytics.TopByPageRank)
					r.Get("/bridges", analytics.Bridges)
					r.Get("/sources", analytics.Sources)
					r.Get("/coverage", analytics.Coverage)
				})

				if deps.Oracle != nil {
					oracleH := apihandler.NewOracleHandler(logger, deps.Oracle)
					r.With(auth.RequireScope("lattice:read")).Post("/oracle", oracleH.Ask)
				}

				if deps.MinIO != nil {
					upload := apihandler.NewUploadHandler(logger, s, deps.MinIO, deps.Producer)
					r.With(auth.RequireScope("lattice:ingest")).Post("/upload", upload.Upload)
				}
			})
		})

		symbols := apihandler.NewSymbolHandler(logger, s, deps.Graph, deps.Lineage, deps.Impact)
		r.Route("/symbols", func(r chi.Router) {
			r.Use(auth.RequireScope("lattice:read"))
			r.Get("/search", symbols.SearchGlobal)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", symbols.Get)
				r.Get("/references", symbols.References)
				r.Get("/lineage", symbols.Lineage)
				r.Get("/impact", symbols.Impact)
				r.Get("/column-lineage", symbols.ColumnLineage)
			})
		})

		webhooks := apihandler.NewWebhookHandler(logger, s, deps.Producer)
		r.With(auth.RequireScope("lattice:ingest")).Post("/webhooks/gitlab/{sourceID}", webhooks.GitLabPush)
	})

	// GraphQL — auth on handler, playground stays open
	gqlResolver := graphql.NewResolver(logger, s, deps.Graph, deps.Embed, deps.Lineage, deps.Impact)
	gqlSrv := handler.New(graphql.NewExecutableSchema(graphql.Config{Resolvers: gqlResolver}))
	gqlSrv.SetErrorPresenter(graphql.ErrorPresenter())
	gqlSrv.AddTransport(transport.POST{})
	gqlSrv.Use(extension.Introspection{})

	r.With(authHandler).Handle("/graphql", gqlSrv)
	r.Get("/graphql/playground", playground.Handler("Lattice", "/graphql"))

	return r
}

func selectAuthMiddleware(logger *slog.Logger, deps *RouterDeps) func(http.Handler) http.Handler {
	if deps.AuthEnabled && deps.Verifier != nil {
		return auth.RequireAuth(deps.Verifier, logger)
	}
	return auth.DevModeMiddleware(logger)
}
