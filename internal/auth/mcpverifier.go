package auth

import (
	"context"
	"fmt"
	"net/http"

	sdkauth "github.com/modelcontextprotocol/go-sdk/auth"
)

// NewMCPTokenVerifier adapts our Verifier to the SDK's auth.TokenVerifier
// function type. It verifies the raw token using OIDC, maps the Principal
// and expiry into an auth.TokenInfo, and stores the Principal in Extra
// so that PrincipalFrom can retrieve it from the SDK context.
func NewMCPTokenVerifier(v *Verifier) sdkauth.TokenVerifier {
	return func(ctx context.Context, token string, _ *http.Request) (*sdkauth.TokenInfo, error) {
		principal, expiry, err := v.VerifyToken(ctx, token)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", sdkauth.ErrInvalidToken, err)
		}

		scopes := make([]string, 0, len(principal.Scopes))
		for s := range principal.Scopes {
			scopes = append(scopes, s)
		}

		return &sdkauth.TokenInfo{
			UserID:     principal.Sub,
			Scopes:     scopes,
			Expiration: expiry,
			Extra: map[string]any{
				"principal": principal,
			},
		}, nil
	}
}
