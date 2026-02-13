package ingestion

import (
	"context"
	"fmt"

	"github.com/codegraph-labs/codegraph/internal/resolver"
)

// ResolveStage performs cross-file symbol resolution.
type ResolveStage struct {
	engine *resolver.Engine
}

func NewResolveStage(engine *resolver.Engine) *ResolveStage {
	return &ResolveStage{engine: engine}
}

func (s *ResolveStage) Name() string { return "resolve" }

func (s *ResolveStage) Execute(ctx context.Context, rc *IndexRunContext) error {
	if len(rc.ParseResults) == 0 {
		return nil
	}

	created, err := s.engine.Resolve(ctx, rc.ProjectID, rc.ParseResults)
	if err != nil {
		return fmt.Errorf("resolve: %w", err)
	}

	rc.EdgesFound += created
	return nil
}
