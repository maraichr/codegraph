package apierr

// Code is a machine-readable error code returned in API responses.
type Code string

// Common errors.
const (
	CodeInvalidRequestBody Code = "INVALID_REQUEST_BODY"
	CodeInvalidID          Code = "INVALID_ID"
	CodeInternalError      Code = "INTERNAL_ERROR"
	CodeNotImplemented     Code = "NOT_IMPLEMENTED"
)

// Project errors.
const (
	CodeProjectNotFound    Code = "PROJECT_NOT_FOUND"
	CodeProjectCreateFailed Code = "PROJECT_CREATE_FAILED"
	CodeProjectUpdateFailed Code = "PROJECT_UPDATE_FAILED"
	CodeProjectDeleteFailed Code = "PROJECT_DELETE_FAILED"
	CodeProjectListFailed   Code = "PROJECT_LIST_FAILED"
	CodeProjectCountFailed  Code = "PROJECT_COUNT_FAILED"
)

// Source errors.
const (
	CodeSourceNotFound    Code = "SOURCE_NOT_FOUND"
	CodeInvalidSourceID   Code = "INVALID_SOURCE_ID"
	CodeInvalidSourceType Code = "INVALID_SOURCE_TYPE"
	CodeSourceCreateFailed Code = "SOURCE_CREATE_FAILED"
	CodeSourceDeleteFailed Code = "SOURCE_DELETE_FAILED"
	CodeSourceListFailed   Code = "SOURCE_LIST_FAILED"
)

// Index run errors.
const (
	CodeIndexRunNotFound    Code = "INDEX_RUN_NOT_FOUND"
	CodeInvalidRunID        Code = "INVALID_RUN_ID"
	CodeIndexRunCreateFailed Code = "INDEX_RUN_CREATE_FAILED"
	CodeIndexRunListFailed   Code = "INDEX_RUN_LIST_FAILED"
	CodeNoSources            Code = "NO_SOURCES"
)

// Symbol errors.
const (
	CodeSymbolNotFound Code = "SYMBOL_NOT_FOUND"
)

// Search & lineage errors.
const (
	CodeSearchFailed       Code = "SEARCH_FAILED"
	CodeLineageQueryFailed Code = "LINEAGE_QUERY_FAILED"
	CodeEmbeddingFailed    Code = "EMBEDDING_FAILED"
)

// Validation errors.
const (
	CodeSlugRequired Code = "SLUG_REQUIRED"
	CodeSlugInvalid  Code = "SLUG_INVALID"
	CodeNameRequired Code = "NAME_REQUIRED"
	CodeNameTooLong  Code = "NAME_TOO_LONG"
)

// Upload errors.
const (
	CodeFileRequired Code = "FILE_REQUIRED"
	CodeUploadFailed Code = "UPLOAD_FAILED"
)

// Webhook errors.
const (
	CodeMissingAuthToken Code = "MISSING_AUTH_TOKEN"
	CodeInvalidAuthToken Code = "INVALID_AUTH_TOKEN"
)

// Health errors.
const (
	CodeDatabaseNotReady Code = "DATABASE_NOT_READY"
)
