package graphql

import (
	"log/slog"

	"github.com/maraichr/codegraph/internal/embedding"
	"github.com/maraichr/codegraph/internal/graph"
	"github.com/maraichr/codegraph/internal/impact"
	"github.com/maraichr/codegraph/internal/lineage"
	"github.com/maraichr/codegraph/internal/store"
)

// Resolver is the root resolver for all GraphQL queries and mutations.
type Resolver struct {
	Logger  *slog.Logger
	Store   *store.Store
	Graph   *graph.Client
	Embed   embedding.Embedder
	Lineage *lineage.Engine
	Impact  *impact.Engine
}

// NewResolver creates a new root resolver.
func NewResolver(logger *slog.Logger, s *store.Store, g *graph.Client, embed embedding.Embedder, lin *lineage.Engine, imp *impact.Engine) *Resolver {
	return &Resolver{Logger: logger, Store: s, Graph: g, Embed: embed, Lineage: lin, Impact: imp}
}
