package tools

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/maraichr/lattice/internal/mcp"
	"github.com/maraichr/lattice/internal/store"
	"github.com/maraichr/lattice/internal/store/postgres"
)

// ToolHandler is the interface that all tool handlers implement.
type ToolHandler[P any] interface {
	Handle(ctx context.Context, params P) (string, error)
}

// WrapHandler adapts a ToolHandler into the SDK's AddTool callback.
// It handles nil params by using a zero value and maps errors to CallToolResult.
func WrapHandler[P any](h ToolHandler[P]) func(context.Context, *sdkmcp.CallToolRequest, *P) (*sdkmcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *sdkmcp.CallToolRequest, params *P) (*sdkmcp.CallToolResult, any, error) {
		if params == nil {
			params = new(P)
		}
		result, err := h.Handle(ctx, *params)
		if err != nil {
			return &sdkmcp.CallToolResult{
				IsError: true,
				Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: err.Error()}},
			}, nil, nil
		}
		return &sdkmcp.CallToolResult{
			Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: result}},
		}, nil, nil
	}
}

// WrapProjectError translates database errors from GetProject into user-friendly messages.
func WrapProjectError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("project not found")
	}
	return fmt.Errorf("get project: %w", err)
}

// WrapSymbolError translates database errors from GetSymbol into user-friendly messages.
func WrapSymbolError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("symbol not found")
	}
	return fmt.Errorf("get symbol: %w", err)
}

// ResolveSymbolByName searches for a symbol by name using ranked search and returns the best match.
func ResolveSymbolByName(ctx context.Context, s *store.Store, projectSlug, name string) (postgres.Symbol, error) {
	results, err := s.SearchSymbolsRanked(ctx, postgres.SearchSymbolsRankedParams{
		ProjectSlug: projectSlug,
		Query:       &name,
		Kinds:       []string{},
		Languages:   []string{},
		Lim:         int32(10),
	})
	if err != nil {
		return postgres.Symbol{}, fmt.Errorf("search symbol: %w", err)
	}
	if len(results) == 0 {
		return postgres.Symbol{}, fmt.Errorf("no symbol found matching '%s'", name)
	}
	ranked := mcp.RankSymbols(results, name, mcp.DefaultRankConfig(), nil)
	return ranked[0].Symbol, nil
}
