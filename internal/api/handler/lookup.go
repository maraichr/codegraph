package handler

import (
	"log/slog"
	"net/http"

	"github.com/google/uuid"

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
