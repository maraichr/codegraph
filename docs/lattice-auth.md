# Lattice OAuth 2.1 Integration Spec (Keycloak reference)

## 0) Versions (latest supported)

* **Keycloak:** 26.5.3 (reference deployment + docs). ([Keycloak][1])
* **OAuth 2.1:** use the latest OAuth 2.1 draft at time of writing (datatracker “draft-ietf-oauth-v2-1”). ([IETF Datatracker][2])
* **MCP Authorization (HTTP transports):** follow MCP authorization guidance for remote servers. ([Model Context Protocol][3])

> Note: OAuth 2.1 is a draft (not a final RFC), but it consolidates modern best practices: **Authorization Code + PKCE, no implicit, no password grant**, strict redirects. ([IETF Datatracker][2])

---

## 1) Goals & non-goals

### Goals

1. Protect MCP Streamable HTTP endpoint with **Bearer access tokens**.
2. Enforce **multi-tenant isolation**: all requests operate under `(tenant_id, sub)`.
3. Support **RBAC** (roles) and **scopes** (recommended).
4. Work across clients:

   * Claude Code / Goose (interactive) → Auth Code + PKCE
   * AgentCore (headless) → Bearer tokens attached by runtime (later: optional exchange/mint)

### Non-goals (v1)

* No provider admin APIs.
* No requirement that clients support token exchange (optional later).

---

## 2) Architecture

### Actors

* **Authorization Server (AS):** Keycloak (reference)
* **Resource Server (RS):** Lattice MCP server (Go)
* **MCP clients:** Claude Code, Goose, AgentCore

### Trust boundary

* Lattice trusts **only** signed tokens from the configured OIDC issuer(s), validated via discovery + JWKS.

---

## 3) Token contract (what Lattice requires)

### Required JWT validation

Lattice MUST validate:

* Signature via **JWKS** from OIDC discovery
* `iss` matches configured issuer
* `aud` includes configured audience (e.g. `lattice`)
* `exp`, `nbf`, `iat` sane

### Required claims

* `sub`: user id
* `tenant_id`: tenant identifier (string) **required** for multi-tenant
* Scopes and/or roles:

  * `scope`: space-delimited scopes (preferred)
  * roles: Keycloak roles may appear under `realm_access.roles` / `resource_access[client].roles`

### Recommended scopes

* `lattice:read` (required for MCP tools)
* `lattice:write` (if you later add mutations)
* `lattice:admin` (admin-only tools)
* `lattice:ingest` (if ingestion actions are user-gated)

### Recommended RBAC roles (mapped to scopes)

* `lattice_reader` → `lattice:read`
* `lattice_admin` → `lattice:admin`
* `lattice_ingestor` → `lattice:ingest`

**Rule:** Enforce **scopes** in Lattice. Roles are an IdP-side convenience to grant scopes.

---

## 4) Multi-tenant + project scoping model

### Data model (minimum)

* `tenants(id, name, ...)`
* `projects(id, tenant_id, name, ...)`
* `memberships(tenant_id, user_sub, role, ...)`  // tenant-level membership
* `project_permissions(project_id, user_sub, role, ...)`  // optional override

### Authorization checks

Every tool call must:

1. Resolve Principal `(tenant_id, sub, scopes, roles)`
2. Confirm `tenant_id` exists and user is a member (or `lattice:admin`)
3. Confirm project belongs to tenant
4. Confirm user has permission for that project OR has tenant-level admin

---

## 5) Code changes (minimal + correct)

### 5.1 Add `internal/auth` package

**(A) Context principal**

```go
// internal/auth/context.go
package auth

import "context"

type ctxKey int
const principalKey ctxKey = 1

type Principal struct {
	Sub      string
	TenantID string
	Scopes   map[string]bool
	Roles    map[string]bool
	ClientID string
	Issuer   string
}

func WithPrincipal(ctx context.Context, p *Principal) context.Context {
	return context.WithValue(ctx, principalKey, p)
}

func PrincipalFrom(ctx context.Context) (*Principal, bool) {
	p, ok := ctx.Value(principalKey).(*Principal)
	return p, ok
}
```

**(B) OIDC verifier (discovery + JWKS)**
Use OIDC discovery so you stay provider-agnostic. MCP specifically frames remote servers as OAuth-protected resources and recommends OAuth 2.1 auth code + related standards. ([Model Context Protocol][3])

Skeleton:

```go
// internal/auth/oidc.go
package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
)

type Verifier struct {
	idt *oidc.IDTokenVerifier
}

type Claims struct {
	Sub   string `json:"sub"`
	Scope string `json:"scope"`
	Azp   string `json:"azp"`
	TenantID string `json:"tenant_id"`
	RealmAccess struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
}

func NewVerifier(ctx context.Context, issuer, audience string) (*Verifier, error) {
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil { return nil, err }
	// ClientID here enforces "aud". Use audience value.
	v := provider.Verifier(&oidc.Config{ClientID: audience})
	return &Verifier{idt: v}, nil
}

func (v *Verifier) VerifyRequest(r *http.Request) (*Principal, error) {
	raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	tok, err := v.idt.Verify(r.Context(), raw)
	if err != nil { return nil, err }

	var c Claims
	if err := tok.Claims(&c); err != nil { return nil, err }

	p := &Principal{
		Sub: c.Sub, TenantID: c.TenantID, ClientID: c.Azp,
		Scopes: parseScopes(c.Scope),
		Roles:  sliceToSet(c.RealmAccess.Roles),
		Issuer: tok.Issuer,
	}
	return p, nil
}

func parseScopes(s string) map[string]bool {
	out := map[string]bool{}
	for _, f := range strings.Fields(s) { out[f] = true }
	return out
}
func sliceToSet(xs []string) map[string]bool {
	out := map[string]bool{}
	for _, x := range xs { out[x] = true }
	return out
}
```

**(C) HTTP middleware**

```go
// internal/auth/middleware.go
package auth

import "net/http"

func RequireAuth(v *Verifier, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			http.Error(w, "missing bearer token", http.StatusUnauthorized)
			return
		}
		p, err := v.VerifyRequest(r)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), p)))
	})
}
```

### 5.2 Wrap your MCP handler in `main.go`

Change:

```go
handler := sdkmcp.NewStreamableHTTPHandler(...)
httpServer := &http.Server{Addr: cfg.MCP.Addr, Handler: handler}
```

To:

```go
mcpHandler := sdkmcp.NewStreamableHTTPHandler(func(*http.Request) *sdkmcp.Server { return sdkServer }, nil)

verifier, err := auth.NewVerifier(ctx, cfg.Auth.IssuerURL, cfg.Auth.Audience)
if err != nil { /* fail fast */ }

handler := auth.RequireAuth(verifier, mcpHandler)
httpServer := &http.Server{Addr: cfg.MCP.Addr, Handler: handler}
```

**Streaming note:** auth must happen **before** streaming starts (this middleware does). ([Model Context Protocol][3])

### 5.3 Enforce tenant + RBAC in tool handlers

At the top of each handler:

```go
p, ok := auth.PrincipalFrom(ctx)
if !ok { return "", errors.New("unauthenticated") }

if !p.Scopes["lattice:read"] && !p.Roles["lattice_reader"] && !p.Roles["lattice_admin"] {
	return "", errors.New("forbidden")
}

if p.TenantID == "" {
	return "", errors.New("tenant_id missing")
}

// Then: enforce project belongs to tenant + membership
```

---

## 6) Keycloak 26.5.3 reference setup

### 6.1 Realm

* Create realm: `lattice`

### 6.2 Clients

Create **two** clients (recommended):

**A) Public client (interactive clients)**

* `client_id`: `lattice-public`
* Type: Public
* Flow: **Authorization Code**
* PKCE: **S256 required**
* Implicit: off
* Direct access grants (password): off
* Redirect URIs: strict (only what your MCP client uses)

**B) Confidential client (service runtimes / automation)**

* `client_id`: `lattice-service`
* Type: Confidential
* Flow: Authorization Code optional (usually off)
* Client auth: on

Keycloak 26.5.3 is current and should be your baseline for docs and examples. ([Keycloak][1])

### 6.3 Scopes

* Configure client scopes so access tokens include `scope` like:

  * `lattice:read lattice:ingest`

### 6.4 tenant_id claim

Add a protocol mapper:

* Claim: `tenant_id`
* Source: user attribute `tenant_id` (or group attribute)
* Add to access token: on

### 6.5 Roles (RBAC)

Create realm roles:

* `lattice_reader`
* `lattice_admin`
* `lattice_ingestor`

Optionally map roles → scopes via client scope mapping (so Lattice can rely primarily on scopes).

---

## 7) Config contract (Lattice)

Add to `config`:

* `AUTH_ISSUER_URL` (e.g. `https://keycloak.example.com/realms/lattice`)
* `AUTH_AUDIENCE` (`lattice`)
* `AUTH_REQUIRED_SCOPE` (`lattice:read`)

---

## 8) Testing workflow (Claude Code, Goose, AgentCore)

### Phase 1 (fast)

* Mint a token in Keycloak (dev) and call:

  * `curl -H "Authorization: Bearer $TOKEN" http://localhost:.../mcp`
* Verify:

  * 401 no token
  * 401 invalid token
  * 403 missing scope
  * 200 success + tool output

### Phase 2 (interactive)

* Ensure your MCP client can do Auth Code + PKCE (some do, some require token pasting first).
* MCP remote auth guidance expects OAuth 2.1 style for remote servers. ([Model Context Protocol][3])

### Phase 3 (AgentCore)

* AgentCore attaches a bearer token when calling Lattice MCP.
* Later you can add a **mint/exchange lane** (optional) if you want AgentCore to exchange upstream identity for a Lattice-audience token.

---

## 9) Optional later: Token exchange / headless minting

MCP and OAuth 2.1 don’t require token exchange, but it can help headless flows. Keep it optional so you don’t become provider-dependent. ([IETF Datatracker][2])

---

## 10) What I’d lock in as your public contract

Publish in `docs/auth.md`:

* Required headers (`Authorization: Bearer …`)
* Claim schema (`sub`, `tenant_id`, `scope`)
* Required scope (`lattice:read`)
* Audience (`lattice`)
* OIDC discovery usage
* Role-to-scope mapping guidance (RBAC)

[1]: https://www.keycloak.org/2026/02/keycloak-2653-released?utm_source=chatgpt.com "Keycloak 26.5.3 released"
[2]: https://datatracker.ietf.org/doc/draft-ietf-oauth-v2-1/?utm_source=chatgpt.com "The OAuth 2.1 Authorization Framework"
[3]: https://modelcontextprotocol.io/specification/draft/basic/authorization?utm_source=chatgpt.com "Authorization"
