package oracle

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/maraichr/codegraph/internal/graph"
	"github.com/maraichr/codegraph/internal/impact"
	"github.com/maraichr/codegraph/internal/llm"
	"github.com/maraichr/codegraph/internal/mcp/session"
	"github.com/maraichr/codegraph/internal/store"
	"github.com/maraichr/codegraph/internal/store/postgres"
)

// Request is the input to the Oracle engine.
type Request struct {
	Question  string `json:"question"`
	SessionID string `json:"session_id,omitempty"`
	Verbosity string `json:"verbosity,omitempty"`
}

// Engine is the Oracle core: routes questions via LLM and executes tool chains.
type Engine struct {
	store   *store.Store
	session *session.Manager
	llm     *llm.Client
	graph   *graph.Client
	impact  *impact.Engine
	logger  *slog.Logger
}

// NewEngine creates a new Oracle engine.
func NewEngine(s *store.Store, sm *session.Manager, llmClient *llm.Client, graphClient *graph.Client, impactEngine *impact.Engine, logger *slog.Logger) *Engine {
	return &Engine{
		store:   s,
		session: sm,
		llm:     llmClient,
		graph:   graphClient,
		impact:  impactEngine,
		logger:  logger,
	}
}

// Store returns the underlying store for project lookups in the handler.
func (e *Engine) Store() *store.Store {
	return e.store
}

// Ask processes a user question for a given project.
func (e *Engine) Ask(ctx context.Context, project postgres.Project, req Request) (*Response, error) {
	// 1. Load/create session
	sess, err := e.session.Load(ctx, req.SessionID)
	if err != nil {
		e.logger.Warn("failed to load session, creating new", slog.String("error", err.Error()))
		sess, _ = e.session.Load(ctx, "")
	}

	// 2. Route intent via LLM (with fallback)
	var sel *ToolSelection
	sel, err = routeIntent(ctx, e.llm, req.Question, sess.RecapText())
	if err != nil {
		e.logger.Warn("LLM routing failed, using fallback", slog.String("error", err.Error()))
		sel = fallbackRoute(req.Question)
	}

	e.logger.Info("oracle routed",
		slog.String("question", req.Question),
		slog.String("tool", sel.Tool),
		slog.String("session", sess.ID))

	// 3. Execute tool
	var blocks []Block
	var items []SymbolItem
	var execErr error

	switch sel.Tool {
	case "search":
		blocks, items, execErr = executeSearch(ctx, e.store, project.Slug, sel.Params)
	case "ranking":
		blocks, items, execErr = executeRanking(ctx, e.store, project.Slug, sel.Params)
	case "overview":
		blocks, execErr = executeOverview(ctx, e.store, project.ID, project.Name)
	case "subgraph":
		blocks, items, execErr = executeSubgraph(ctx, e.store, project.Slug, sel.Params)
	case "relationships":
		blocks, items, execErr = executeRelationships(ctx, e.store, project.Slug, sel.Params)
	case "lineage":
		blocks, items, execErr = executeLineage(ctx, e.store, e.graph, project.Slug, sel.Params)
	case "impact":
		blocks, items, execErr = executeImpact(ctx, e.store, e.impact, project.Slug, sel.Params)
	default:
		blocks, items, execErr = executeSearch(ctx, e.store, project.Slug, sel.Params)
	}

	if execErr != nil {
		return nil, fmt.Errorf("execute %s: %w", sel.Tool, execErr)
	}

	// 4. Generate hints
	hints := generateHints(sel.Tool, items)

	// 5. Update session
	sess.AddQuery(req.Question)
	sess.AddRecap(fmt.Sprintf("Asked about: %s (tool: %s, %d results)", req.Question, sel.Tool, len(items)))
	for _, item := range items {
		sess.MarkSeen(uuidFromString(item.ID))
	}
	if err := e.session.Save(ctx, sess); err != nil {
		e.logger.Warn("failed to save session", slog.String("error", err.Error()))
	}

	// 6. Build response
	totalResults := len(items)
	if totalResults == 0 {
		totalResults = 1 // at least the text block counts
	}

	return &Response{
		SessionID: sess.ID,
		Tool:      sel.Tool,
		Blocks:    blocks,
		Hints:     hints,
		Meta: ResponseMeta{
			ToolSelected: sel.Tool,
			TotalResults: totalResults,
			Shown:        len(items),
		},
	}, nil
}

// generateHints produces follow-up question suggestions based on the tool used.
func generateHints(tool string, items []SymbolItem) []Hint {
	var hints []Hint

	switch tool {
	case "search":
		if len(items) > 0 {
			first := items[0]
			hints = append(hints,
				Hint{Label: "Impact", Question: fmt.Sprintf("What happens if I change %s?", first.Name)},
				Hint{Label: "Lineage", Question: fmt.Sprintf("Show data flow for %s", first.Name)},
				Hint{Label: "Related", Question: fmt.Sprintf("Show everything related to %s", first.Name)},
			)
		}
	case "ranking":
		hints = append(hints,
			Hint{Label: "Overview", Question: "Give me a project overview"},
			Hint{Label: "Relationships", Question: "Show table relationships"},
		)
		if len(items) > 0 {
			hints = append(hints,
				Hint{Label: "Deep dive", Question: fmt.Sprintf("Tell me about %s", items[0].Name)},
			)
		}
	case "overview":
		hints = append(hints,
			Hint{Label: "Top tables", Question: "What are the most important tables?"},
			Hint{Label: "Architecture", Question: "Show table relationships"},
			Hint{Label: "Entry points", Question: "What are the most used procedures?"},
		)
	case "subgraph", "relationships":
		if len(items) > 0 {
			hints = append(hints,
				Hint{Label: "Impact", Question: fmt.Sprintf("What breaks if %s changes?", items[0].Name)},
			)
		}
		hints = append(hints,
			Hint{Label: "Top symbols", Question: "What are the most connected symbols?"},
		)
	case "lineage":
		if len(items) > 0 {
			hints = append(hints,
				Hint{Label: "Impact", Question: fmt.Sprintf("What breaks if %s changes?", items[0].Name)},
				Hint{Label: "Related", Question: fmt.Sprintf("Show module around %s", items[0].Name)},
			)
		}
	case "impact":
		if len(items) > 0 {
			hints = append(hints,
				Hint{Label: "Lineage", Question: fmt.Sprintf("Show data flow for %s", items[0].Name)},
				Hint{Label: "Related", Question: fmt.Sprintf("Show everything related to %s", items[0].Name)},
			)
		}
	}

	return hints
}

func uuidFromString(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.UUID{}
	}
	return id
}
