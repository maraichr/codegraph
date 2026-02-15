package apierr

import "net/http"

// --- Common ---

func InvalidRequestBody() *Error {
	return New(CodeInvalidRequestBody, http.StatusBadRequest, "Invalid request body")
}

func InvalidID(entity string) *Error {
	return New(CodeInvalidID, http.StatusBadRequest, "Invalid "+entity+" ID")
}

func InternalError(cause error) *Error {
	return Wrap(CodeInternalError, http.StatusInternalServerError, "Internal server error", cause)
}

func NotImplemented(feature string) *Error {
	return New(CodeNotImplemented, http.StatusNotImplemented, feature+" is not implemented yet")
}

// --- Project ---

func ProjectNotFound() *Error {
	return New(CodeProjectNotFound, http.StatusNotFound, "Project not found")
}

func ProjectCreateFailed(cause error) *Error {
	return Wrap(CodeProjectCreateFailed, http.StatusInternalServerError, "Failed to create project", cause)
}

func ProjectUpdateFailed(cause error) *Error {
	return Wrap(CodeProjectUpdateFailed, http.StatusInternalServerError, "Failed to update project", cause)
}

func ProjectDeleteFailed(cause error) *Error {
	return Wrap(CodeProjectDeleteFailed, http.StatusInternalServerError, "Failed to delete project", cause)
}

func ProjectListFailed(cause error) *Error {
	return Wrap(CodeProjectListFailed, http.StatusInternalServerError, "Failed to list projects", cause)
}

func ProjectCountFailed(cause error) *Error {
	return Wrap(CodeProjectCountFailed, http.StatusInternalServerError, "Failed to count projects", cause)
}

// --- Source ---

func SourceNotFound() *Error {
	return New(CodeSourceNotFound, http.StatusNotFound, "Source not found")
}

func InvalidSourceID() *Error {
	return New(CodeInvalidSourceID, http.StatusBadRequest, "Invalid source ID")
}

func InvalidSourceType() *Error {
	return New(CodeInvalidSourceType, http.StatusBadRequest, "source_type must be one of: git, database, filesystem, upload")
}

func SourceCreateFailed(cause error) *Error {
	return Wrap(CodeSourceCreateFailed, http.StatusInternalServerError, "Failed to create source", cause)
}

func SourceDeleteFailed(cause error) *Error {
	return Wrap(CodeSourceDeleteFailed, http.StatusInternalServerError, "Failed to delete source", cause)
}

func SourceListFailed(cause error) *Error {
	return Wrap(CodeSourceListFailed, http.StatusInternalServerError, "Failed to list sources", cause)
}

// --- Index Run ---

func IndexRunNotFound() *Error {
	return New(CodeIndexRunNotFound, http.StatusNotFound, "Index run not found")
}

func InvalidRunID() *Error {
	return New(CodeInvalidRunID, http.StatusBadRequest, "Invalid run ID")
}

func IndexRunCreateFailed(cause error) *Error {
	return Wrap(CodeIndexRunCreateFailed, http.StatusInternalServerError, "Failed to create index run", cause)
}

func IndexRunListFailed(cause error) *Error {
	return Wrap(CodeIndexRunListFailed, http.StatusInternalServerError, "Failed to list index runs", cause)
}

func NoSources() *Error {
	return New(CodeNoSources, http.StatusBadRequest, "Project has no sources to index")
}

// --- Symbol ---

func SymbolNotFound() *Error {
	return New(CodeSymbolNotFound, http.StatusNotFound, "Symbol not found")
}

// --- Search & Lineage ---

func SearchFailed(cause error) *Error {
	return Wrap(CodeSearchFailed, http.StatusInternalServerError, "Search failed", cause)
}

func LineageQueryFailed(cause error) *Error {
	return Wrap(CodeLineageQueryFailed, http.StatusInternalServerError, "Lineage query failed", cause)
}

func EmbeddingFailed(cause error) *Error {
	return Wrap(CodeEmbeddingFailed, http.StatusInternalServerError, "Embedding generation failed", cause)
}

// --- Analytics ---

func AnalyticsFailed(cause error) *Error {
	return Wrap(CodeAnalyticsFailed, http.StatusInternalServerError, "Analytics query failed", cause)
}

// --- Validation ---

func SlugRequired() *Error {
	return New(CodeSlugRequired, http.StatusBadRequest, "Slug is required")
}

func SlugInvalid() *Error {
	return New(CodeSlugInvalid, http.StatusBadRequest, "Slug must be 3-63 chars, lowercase alphanumeric and hyphens, must start/end with alphanumeric")
}

func NameRequired() *Error {
	return New(CodeNameRequired, http.StatusBadRequest, "Name is required")
}

func NameTooLong() *Error {
	return New(CodeNameTooLong, http.StatusBadRequest, "Name must be 255 characters or fewer")
}

// --- Upload ---

func FileRequired() *Error {
	return New(CodeFileRequired, http.StatusBadRequest, "File is required (multipart field 'file')")
}

func UploadFailed(cause error) *Error {
	return Wrap(CodeUploadFailed, http.StatusInternalServerError, "Failed to upload file", cause)
}

// --- Webhook ---

func MissingAuthToken() *Error {
	return New(CodeMissingAuthToken, http.StatusUnauthorized, "Missing X-Gitlab-Token header")
}

func InvalidAuthToken() *Error {
	return New(CodeInvalidAuthToken, http.StatusUnauthorized, "Invalid webhook token")
}

// --- Health ---

func DatabaseNotReady() *Error {
	return New(CodeDatabaseNotReady, http.StatusServiceUnavailable, "Database not ready")
}
