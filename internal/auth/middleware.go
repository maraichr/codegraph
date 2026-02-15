package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type errorResponse struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeAuthError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errorResponse{
		Error: errorBody{Code: code, Message: message},
	})
}

// RequireAuth validates the JWT and injects the Principal into the context.
func RequireAuth(verifier *Verifier, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, err := verifier.VerifyRequest(r)
			if err != nil {
				logger.Warn("auth failed", slog.String("error", err.Error()), slog.String("path", r.URL.Path))
				writeAuthError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
				return
			}
			ctx := WithPrincipal(r.Context(), principal)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireScope checks that the Principal has at least one of the required scopes.
// Admins bypass scope checks.
func RequireScope(scopes ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := PrincipalFrom(r.Context())
			if !ok {
				writeAuthError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
				return
			}

			if p.IsAdmin() || p.HasAnyScope(scopes...) {
				next.ServeHTTP(w, r)
				return
			}

			writeAuthError(w, http.StatusForbidden, "FORBIDDEN", "Insufficient scope")
		})
	}
}
