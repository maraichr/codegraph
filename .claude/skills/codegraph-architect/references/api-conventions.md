# API Conventions

Conventions for the CodeGraph REST and GraphQL APIs, error handling, and middleware.

## Table of Contents

1. [REST API conventions](#rest-api-conventions)
2. [GraphQL conventions](#graphql-conventions)
3. [Error handling](#error-handling)
4. [Middleware stack](#middleware-stack)
5. [Authentication](#authentication)
6. [Adding a new endpoint](#adding-a-new-endpoint)
7. [Adding a new GraphQL field](#adding-a-new-graphql-field)

---

## REST API Conventions

Base path: `/api/v1`
Router: chi v5 (stdlib-compatible)
Handlers: `internal/api/handler/`

### URL patterns

- Resources are plural nouns: `/projects`, `/sources`, `/symbols`
- Nested resources for ownership: `/projects/{slug}/sources`
- Actions as sub-resources: `/sources/{id}/resync`
- Use project slug (not UUID) in URLs for readability
- Use UUIDs for all other resource identifiers

### Request/Response

- Request bodies: JSON, decoded with `json.NewDecoder(r.Body)`
- Response bodies: JSON, encoded with `json.NewEncoder(w).Encode()`
- Always set `Content-Type: application/json` header
- Pagination: `page` + `per_page` query params, return `X-Total-Count` header
- Sorting: `sort` query param (e.g., `sort=fqn`, `sort=-created_at` for descending)

### HTTP methods

- `POST` for creation (returns 201 + Location header)
- `GET` for reads
- `PUT` for full updates
- `PATCH` for partial updates (not yet used, but prefer over PUT for future endpoints)
- `DELETE` for removal (returns 204 No Content)

### Handler structure

```go
func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
    // 1. Decode request body
    var req CreateProjectRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        apierr.Write(w, apierr.InvalidBody(err))
        return
    }

    // 2. Validate
    if err := req.Validate(); err != nil {
        apierr.Write(w, err)
        return
    }

    // 3. Execute business logic
    project, err := h.store.CreateProject(r.Context(), req.ToParams())
    if err != nil {
        apierr.Write(w, apierr.Internal(err))
        return
    }

    // 4. Return response
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(project)
}
```

## GraphQL Conventions

Generator: gqlgen
Schema: `internal/api/graphql/schema/`
Resolvers: `internal/api/graphql/resolver/`

### Schema design

- Use connections for paginated lists (Relay-style cursor pagination):
  ```graphql
  type Query {
    symbols(projectSlug: String!, filters: SymbolFilters, first: Int, after: String): SymbolConnection!
  }
  ```
- Use input types for mutations:
  ```graphql
  input CreateProjectInput {
    name: String!
    slug: String!
    description: String
  }
  ```
- Enums are UPPER_CASE in GraphQL, mapped from lowercase DB values in resolvers

### Resolver patterns

- Resolvers call the same store layer as REST handlers
- Use dataloaders for N+1 prevention on nested fields
- Return `NOT_IMPLEMENTED` for Phase 2+ features:
  ```go
  func (r *queryResolver) Lineage(ctx context.Context, symbolID string, direction model.LineageDirection, maxDepth *int) (*model.LineageGraph, error) {
      return nil, apierr.NotImplemented("lineage queries")
  }
  ```

## Error Handling

All errors go through `pkg/apierr`. Never return raw error strings or status codes.

### Error catalog

Defined in `pkg/apierr/code.go`. Every error has:
- A string code (e.g., `PROJECT_NOT_FOUND`) — machine-readable
- An HTTP status (e.g., 404) — for REST responses
- A human-readable message

### REST error format

```json
{
  "error": {
    "code": "PROJECT_NOT_FOUND",
    "message": "project not found"
  }
}
```

### GraphQL error format

```json
{
  "errors": [{
    "message": "project not found",
    "path": ["project"],
    "extensions": { "code": "PROJECT_NOT_FOUND" }
  }]
}
```

### Adding a new error code

1. Add the code constant to `pkg/apierr/code.go`
2. Add a constructor function (e.g., `apierr.SourceNotFound()`)
3. Use it in handlers — never construct error responses manually

### Common error constructors

```go
apierr.InvalidBody(err)              // 400, INVALID_REQUEST_BODY
apierr.ValidationError(code, msg)    // 400, custom code
apierr.NotFound(code, msg)           // 404, custom code
apierr.Unauthorized(code, msg)       // 401, custom code
apierr.Internal(err)                 // 500, INTERNAL_ERROR
apierr.NotImplemented(feature)       // 501, NOT_IMPLEMENTED
apierr.ServiceUnavailable(msg)       // 503, DATABASE_NOT_READY
```

## Middleware Stack

Applied in order in `cmd/api/main.go`:

1. **Request ID** — generates unique ID, sets `X-Request-ID` header
2. **Structured logging** — logs request/response with slog
3. **Recovery** — panic recovery, returns 500
4. **CORS** — configurable allowed origins
5. **Auth** — JWT validation (skipped for health check, webhooks)
6. **Rate limiting** — per API key / IP
7. **Request timeout** — configurable per-route

## Authentication

- External: JWT from IdP (Keycloak, Azure AD, Okta)
- Internal: API keys for service accounts (MCP server, CI/CD)
- Webhooks: Per-source secret in `X-Gitlab-Token` header

JWT validation middleware extracts claims and sets user context:
```go
user := auth.UserFromContext(r.Context())
```

Webhook validation is per-handler (not middleware) since it uses source-specific secrets.

## Adding a New Endpoint

1. Define the route in `cmd/api/main.go` (or the appropriate router group)
2. Create handler method in the appropriate handler file
3. Define request/response types with JSON tags
4. Add validation logic
5. Use `apierr` for all error responses
6. Add integration test in `test/integration/`
7. Update the API section in SPEC.md if this is a new resource

## Adding a New GraphQL Field

1. Update schema in `internal/api/graphql/schema/*.graphql`
2. Run `go generate ./internal/api/graphql/...` to regenerate
3. Implement resolver method (gqlgen generates a stub)
4. Add dataloader if the field causes N+1 queries
5. Test with GraphQL playground in dev environment
