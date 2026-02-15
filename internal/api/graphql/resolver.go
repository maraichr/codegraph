package graphql

import (
	"log/slog"

	"github.com/maraichr/lattice/internal/embedding"
	"github.com/maraichr/lattice/internal/graph"
	"github.com/maraichr/lattice/internal/impact"
	"github.com/maraichr/lattice/internal/lineage"
	"github.com/maraichr/lattice/internal/store"
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
