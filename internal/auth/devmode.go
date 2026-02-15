package auth

import (
	"log/slog"
	"net/http"

	"github.com/google/uuid"
)

// DefaultTenantID is the UUID of the seed "default" tenant.
var DefaultTenantID = uuid.MustParse("00000000-0000-0000-0000-000000000099")

// DevModeMiddleware injects a synthetic Principal with all scopes and admin role.
// Use only when AUTH_ENABLED=false (development).
func DevModeMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	logger.Warn("DEV MODE: Authentication disabled — all requests get admin principal")
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p := &Principal{
				Sub:      "dev-user",
				TenantID: DefaultTenantID,
				Scopes: map[string]bool{
					"openid":           true,
					"codegraph:read":   true,
					"codegraph:write":  true,
					"codegraph:ingest": true,
					"codegraph:admin":  true,
				},
				Roles: map[string]bool{
					"codegraph_admin":    true,
					"codegraph_reader":   true,
					"codegraph_ingestor": true,
				},
				ClientID: "dev",
				Issuer:   "dev",
				Email:    "dev@codegraph.dev",
			}
			ctx := WithPrincipal(r.Context(), p)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
