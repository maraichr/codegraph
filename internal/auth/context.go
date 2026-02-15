package auth

import (
	"context"

	"github.com/google/uuid"
)

type ctxKey struct{}

// Principal represents an authenticated identity extracted from a JWT.
type Principal struct {
	Sub      string            `json:"sub"`
	TenantID uuid.UUID         `json:"tenant_id"`
	Scopes   map[string]bool   `json:"scopes"`
	Roles    map[string]bool   `json:"roles"`
	ClientID string            `json:"client_id"`
	Issuer   string            `json:"issuer"`
	Email    string            `json:"email"`
}

// WithPrincipal stores a Principal in the context.
func WithPrincipal(ctx context.Context, p *Principal) context.Context {
	return context.WithValue(ctx, ctxKey{}, p)
}

// PrincipalFrom extracts the Principal from the context.
func PrincipalFrom(ctx context.Context) (*Principal, bool) {
	p, ok := ctx.Value(ctxKey{}).(*Principal)
	return p, ok
}

// HasScope returns true if the principal has the given scope.
func (p *Principal) HasScope(s string) bool {
	return p.Scopes[s]
}

// HasAnyScope returns true if the principal has any of the given scopes.
func (p *Principal) HasAnyScope(scopes ...string) bool {
	for _, s := range scopes {
		if p.Scopes[s] {
			return true
		}
	}
	return false
}

// IsAdmin returns true if the principal has the codegraph_admin role.
func (p *Principal) IsAdmin() bool {
	return p.Roles["codegraph_admin"]
}

// HasRole returns true if the principal has the given role.
func (p *Principal) HasRole(r string) bool {
	return p.Roles[r]
}
