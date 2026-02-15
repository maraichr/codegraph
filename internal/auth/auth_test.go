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
		Scopes:   map[string]bool{"lattice:read": true},
		Roles:    map[string]bool{"lattice_admin": true},
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
			"lattice:read":  true,
			"lattice:write": true,
		},
	}

	if !p.HasScope("lattice:read") {
		t.Error("expected HasScope(lattice:read) = true")
	}
	if p.HasScope("lattice:admin") {
		t.Error("expected HasScope(lattice:admin) = false")
	}
}

func TestHasAnyScope(t *testing.T) {
	p := &Principal{
		Scopes: map[string]bool{"lattice:read": true},
	}

	if !p.HasAnyScope("lattice:write", "lattice:read") {
		t.Error("expected HasAnyScope to match lattice:read")
	}
	if p.HasAnyScope("lattice:write", "lattice:admin") {
		t.Error("expected HasAnyScope to return false when none match")
	}
}

func TestIsAdmin(t *testing.T) {
	admin := &Principal{Roles: map[string]bool{"lattice_admin": true}}
	reader := &Principal{Roles: map[string]bool{"lattice_reader": true}}

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
	if !gotPrincipal.HasScope("lattice:read") {
		t.Error("dev mode principal should have lattice:read scope")
	}
	if gotPrincipal.TenantID != DefaultTenantID {
		t.Errorf("got tenant %v, want default %v", gotPrincipal.TenantID, DefaultTenantID)
	}
}

func TestRequireScope_Pass(t *testing.T) {
	p := &Principal{
		Scopes: map[string]bool{"lattice:read": true},
	}

	mw := RequireScope("lattice:read")
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
		Scopes: map[string]bool{"lattice:read": true},
	}

	mw := RequireScope("lattice:write")
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
		Roles:  map[string]bool{"lattice_admin": true},
	}

	mw := RequireScope("lattice:write")
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
	mw := RequireScope("lattice:read")
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
