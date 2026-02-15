package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"log/slog"

	"github.com/google/uuid"
)

func TestPrincipalContext(t *testing.T) {
	ctx := context.Background()

	// No principal yet
	if _, ok := PrincipalFrom(ctx); ok {
		t.Fatal("expected no principal in empty context")
	}

	p := &Principal{
		Sub:      "user-123",
		TenantID: uuid.MustParse("00000000-0000-0000-0000-000000000099"),
		Scopes:   map[string]bool{"codegraph:read": true},
		Roles:    map[string]bool{"codegraph_admin": true},
	}

	ctx = WithPrincipal(ctx, p)
	got, ok := PrincipalFrom(ctx)
	if !ok {
		t.Fatal("expected principal in context")
	}
	if got.Sub != "user-123" {
		t.Fatalf("got sub %q, want %q", got.Sub, "user-123")
	}
	if got.TenantID != p.TenantID {
		t.Fatalf("got tenant %v, want %v", got.TenantID, p.TenantID)
	}
}

func TestHasScope(t *testing.T) {
	p := &Principal{
		Scopes: map[string]bool{
			"codegraph:read":  true,
			"codegraph:write": true,
		},
	}

	if !p.HasScope("codegraph:read") {
		t.Error("expected HasScope(codegraph:read) = true")
	}
	if p.HasScope("codegraph:admin") {
		t.Error("expected HasScope(codegraph:admin) = false")
	}
}

func TestHasAnyScope(t *testing.T) {
	p := &Principal{
		Scopes: map[string]bool{"codegraph:read": true},
	}

	if !p.HasAnyScope("codegraph:write", "codegraph:read") {
		t.Error("expected HasAnyScope to match codegraph:read")
	}
	if p.HasAnyScope("codegraph:write", "codegraph:admin") {
		t.Error("expected HasAnyScope to return false when none match")
	}
}

func TestIsAdmin(t *testing.T) {
	admin := &Principal{Roles: map[string]bool{"codegraph_admin": true}}
	reader := &Principal{Roles: map[string]bool{"codegraph_reader": true}}

	if !admin.IsAdmin() {
		t.Error("expected admin to be admin")
	}
	if reader.IsAdmin() {
		t.Error("expected reader to not be admin")
	}
}

func TestDevModeMiddleware(t *testing.T) {
	logger := slog.Default()
	mw := DevModeMiddleware(logger)

	var gotPrincipal *Principal
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, ok := PrincipalFrom(r.Context())
		if !ok {
			t.Fatal("expected principal in context")
		}
		gotPrincipal = p
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200", rec.Code)
	}
	if gotPrincipal == nil {
		t.Fatal("principal was nil")
	}
	if !gotPrincipal.IsAdmin() {
		t.Error("dev mode principal should be admin")
	}
	if !gotPrincipal.HasScope("codegraph:read") {
		t.Error("dev mode principal should have codegraph:read scope")
	}
	if gotPrincipal.TenantID != DefaultTenantID {
		t.Errorf("got tenant %v, want default %v", gotPrincipal.TenantID, DefaultTenantID)
	}
}

func TestRequireScope_Pass(t *testing.T) {
	p := &Principal{
		Scopes: map[string]bool{"codegraph:read": true},
	}

	mw := RequireScope("codegraph:read")
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = req.WithContext(WithPrincipal(req.Context(), p))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200", rec.Code)
	}
}

func TestRequireScope_Fail(t *testing.T) {
	p := &Principal{
		Scopes: map[string]bool{"codegraph:read": true},
	}

	mw := RequireScope("codegraph:write")
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = req.WithContext(WithPrincipal(req.Context(), p))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("got status %d, want 403", rec.Code)
	}
}

func TestRequireScope_AdminBypass(t *testing.T) {
	p := &Principal{
		Scopes: map[string]bool{},
		Roles:  map[string]bool{"codegraph_admin": true},
	}

	mw := RequireScope("codegraph:write")
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = req.WithContext(WithPrincipal(req.Context(), p))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("admin should bypass scope check, got status %d", rec.Code)
	}
}

func TestRequireScope_NoPrincipal(t *testing.T) {
	mw := RequireScope("codegraph:read")
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want 401", rec.Code)
	}
}
