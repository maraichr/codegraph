package tools

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/maraichr/lattice/internal/auth"
	"github.com/maraichr/lattice/internal/mcp"
	"github.com/maraichr/lattice/internal/store"
	"github.com/maraichr/lattice/internal/store/postgres"
)

// ListProjectsParams are the parameters for the list_projects tool.
type ListProjectsParams struct {
	Limit int32 `json:"limit,omitempty"`
}

// ListProjectsHandler implements the list_projects MCP tool.
type ListProjectsHandler struct {
	store  *store.Store
	logger *slog.Logger
}

// NewListProjectsHandler creates a new handler.
func NewListProjectsHandler(s *store.Store, logger *slog.Logger) *ListProjectsHandler {
	return &ListProjectsHandler{store: s, logger: logger}
}

// Handle lists projects accessible to the authenticated user.
func (h *ListProjectsHandler) Handle(ctx context.Context, params ListProjectsParams) (string, error) {
	if params.Limit <= 0 {
		params.Limit = 50
	}

	p, _ := auth.PrincipalFrom(ctx)

	var projects []postgres.Project
	var err error

	if p != nil && !p.IsAdmin() {
		projects, err = h.store.ListProjectsByTenant(ctx, postgres.ListProjectsByTenantParams{
			TenantID: p.TenantID,
			Limit:    params.Limit,
			Offset:   0,
		})
	} else {
		projects, err = h.store.ListProjects(ctx, postgres.ListProjectsParams{
			Limit:  params.Limit,
			Offset: 0,
		})
	}
	if err != nil {
		return "", fmt.Errorf("list projects: %w", err)
	}

	if len(projects) == 0 {
		return "No projects found.", nil
	}

	rb := mcp.NewResponseBuilder(4000)
	rb.AddHeader(fmt.Sprintf("**Projects** (%d found)", len(projects)))

	for _, proj := range projects {
		desc := ""
		if proj.Description != nil {
			desc = " — " + *proj.Description
		}
		if !rb.AddLine(fmt.Sprintf("- **%s** (`%s`)%s", proj.Name, proj.Slug, desc)) {
			break
		}
	}

	return rb.Finalize(len(projects), len(projects)), nil
}
