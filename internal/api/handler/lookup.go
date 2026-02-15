package handler

import (
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/maraichr/codegraph/internal/auth"
	"github.com/maraichr/codegraph/internal/store"
	"github.com/maraichr/codegraph/internal/store/postgres"
	"github.com/maraichr/codegraph/pkg/apierr"
)

// getProjectOr404 fetches a project by slug and writes a 404/500 error on failure.
// Returns the project and true on success, or zero-value and false if an error was written.
func getProjectOr404(w http.ResponseWriter, r *http.Request, logger *slog.Logger, s *store.Store, slug string) (postgres.Project, bool) {
	project, err := s.GetProject(r.Context(), slug)
	if err != nil {
		if apierr.IsNotFound(err) {
			writeAPIError(w, logger, apierr.ProjectNotFound())
		} else {
			writeAPIError(w, logger, apierr.InternalError(err))
		}
		return postgres.Project{}, false
	}
	return project, true
}

// getSourceOr404 fetches a source by UUID and writes a 404/500 error on failure.
// Returns the source and true on success, or zero-value and false if an error was written.
func getSourceOr404(w http.ResponseWriter, r *http.Request, logger *slog.Logger, s *store.Store, id uuid.UUID) (postgres.Source, bool) {
	source, err := s.GetSource(r.Context(), id)
	if err != nil {
		if apierr.IsNotFound(err) {
			writeAPIError(w, logger, apierr.SourceNotFound())
		} else {
			writeAPIError(w, logger, apierr.InternalError(err))
		}
		return postgres.Source{}, false
	}
	return source, true
}

// checkTenantAccess verifies that the project belongs to the principal's tenant.
// Admins bypass the check. Returns true if access is allowed.
func checkTenantAccess(w http.ResponseWriter, r *http.Request, logger *slog.Logger, project postgres.Project) bool {
	p, ok := auth.PrincipalFrom(r.Context())
	if !ok {
		writeAPIError(w, logger, apierr.Unauthorized("Authentication required"))
		return false
	}
	if p.IsAdmin() {
		return true
	}
	if project.TenantID != p.TenantID {
		writeAPIError(w, logger, apierr.Forbidden("Access denied to this project"))
		return false
	}
	return true
}
