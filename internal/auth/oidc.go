package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
)

// Verifier validates JWTs using OIDC discovery and JWKS.
type Verifier struct {
	provider *oidc.Provider
	verifier *oidc.IDTokenVerifier
	audience string
}

// NewVerifier creates a Verifier using OIDC discovery from the issuer URL.
// publicIssuer optionally specifies the expected token issuer when it differs
// from the discovery URL (e.g. in Docker where discovery uses http://keycloak:8081
// but tokens contain iss: http://localhost:8081).
func NewVerifier(ctx context.Context, issuerURL, publicIssuer, audience string) (*Verifier, error) {
	if publicIssuer != "" && publicIssuer != issuerURL {
		// Tell go-oidc to accept tokens whose iss claim matches publicIssuer
		// even though discovery is fetched from issuerURL.
		ctx = oidc.InsecureIssuerURLContext(ctx, publicIssuer)
	}

	provider, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		return nil, fmt.Errorf("oidc discovery: %w", err)
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID: audience,
	})

	return &Verifier{
		provider: provider,
		verifier: verifier,
		audience: audience,
	}, nil
}

// claims represents the JWT claims we extract.
type claims struct {
	Sub             string      `json:"sub"`
	Email           string      `json:"email"`
	TenantID        string      `json:"tenant_id"`
	Scope           string      `json:"scope"`
	LatticeScopes string      `json:"lattice_scopes"`
	Azp             string      `json:"azp"`
	RealmAccess     realmAccess `json:"realm_access"`
}

type realmAccess struct {
	Roles []string `json:"roles"`
}

// VerifyToken verifies a raw Bearer token string and returns the Principal
// and the token's expiry time.
func (v *Verifier) VerifyToken(ctx context.Context, rawToken string) (*Principal, time.Time, error) {
	token, err := v.verifier.Verify(ctx, rawToken)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("token verification failed: %w", err)
	}

	var c claims
	if err := token.Claims(&c); err != nil {
		return nil, time.Time{}, fmt.Errorf("failed to parse claims: %w", err)
	}

	if c.TenantID == "" {
		return nil, time.Time{}, fmt.Errorf("missing tenant_id claim")
	}

	tenantID, err := uuid.Parse(c.TenantID)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("invalid tenant_id claim: %w", err)
	}

	scopes := make(map[string]bool)
	for _, s := range strings.Fields(c.Scope) {
		scopes[s] = true
	}
	// Also parse lattice-specific scopes from custom claim
	for _, s := range strings.Fields(c.LatticeScopes) {
		scopes[s] = true
	}

	roles := make(map[string]bool)
	for _, r := range c.RealmAccess.Roles {
		roles[r] = true
	}

	return &Principal{
		Sub:      c.Sub,
		TenantID: tenantID,
		Scopes:   scopes,
		Roles:    roles,
		ClientID: c.Azp,
		Issuer:   token.Issuer,
		Email:    c.Email,
	}, token.Expiry, nil
}

// VerifyRequest extracts and verifies the Bearer token from the request.
func (v *Verifier) VerifyRequest(r *http.Request) (*Principal, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("missing Authorization header")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return nil, fmt.Errorf("invalid Authorization header format")
	}

	p, _, err := v.VerifyToken(r.Context(), parts[1])
	return p, err
}
