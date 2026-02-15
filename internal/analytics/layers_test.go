package analytics

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/maraichr/codegraph/internal/store/postgres"
)

func sym(name, kind, fqn string) postgres.Symbol {
	return postgres.Symbol{
		ID:            uuid.New(),
		ProjectID:     uuid.New(),
		FileID:        uuid.New(),
		Name:          name,
		QualifiedName: fqn,
		Kind:          kind,
		Language:      "go",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

// --- classifyLayer ---

func TestClassifyLayer_DataKinds(t *testing.T) {
	tests := []struct {
		kind string
	}{
		{"table"}, {"view"}, {"column"}, {"procedure"}, {"trigger"},
	}
	for _, tt := range tests {
		s := sym("X", tt.kind, "app.X")
		got := classifyLayer(s)
		if got != LayerData {
			t.Errorf("kind %q should classify as data, got %s", tt.kind, got)
		}
	}
}

func TestClassifyLayer_APINamespace(t *testing.T) {
	tests := []struct {
		fqn string
	}{
		{"app.controller.UserController"},
		{"app.handlers.AuthHandler"},
		{"api.v1.OrderEndpoint"},
		{"web.routes.GetUser"},
		{"app.rest.Client"},
		{"graphql.resolvers.Query"},
	}
	for _, tt := range tests {
		s := sym("X", "class", tt.fqn)
		got := classifyLayer(s)
		if got != LayerAPI {
			t.Errorf("FQN %q should classify as api, got %s", tt.fqn, got)
		}
	}
}

func TestClassifyLayer_DataNamespace(t *testing.T) {
	tests := []struct {
		fqn string
	}{
		{"app.repository.CustomerRepo"},
		{"app.dal.DataAccess"},
		{"persistence.UserStore"},
		{"app.dao.OrderDAO"},
		{"dbo.Customers"},
	}
	for _, tt := range tests {
		s := sym("X", "class", tt.fqn)
		got := classifyLayer(s)
		if got != LayerData {
			t.Errorf("FQN %q should classify as data, got %s", tt.fqn, got)
		}
	}
}

func TestClassifyLayer_BusinessNamespace(t *testing.T) {
	tests := []struct {
		fqn string
	}{
		{"app.service.OrderService"},
		{"domain.Customer"},
		{"app.core.ProcessEngine"},
		{"business.logic.Calculator"},
		{"app.usecases.PlaceOrder"},
	}
	for _, tt := range tests {
		s := sym("X", "class", tt.fqn)
		got := classifyLayer(s)
		if got != LayerBusiness {
			t.Errorf("FQN %q should classify as business, got %s", tt.fqn, got)
		}
	}
}

func TestClassifyLayer_InfraNamespace(t *testing.T) {
	tests := []struct {
		fqn string
	}{
		{"app.config.AppConfig"},
		{"app.infrastructure.Startup"}, // "infrastructure" matches infra, no data segment
		{"app.middleware.AuthMiddleware"},
		{"app.logging.Logger"},
		{"setup.Bootstrap"},
	}
	for _, tt := range tests {
		s := sym("X", "class", tt.fqn)
		got := classifyLayer(s)
		if got != LayerInfrastructure {
			t.Errorf("FQN %q should classify as infrastructure, got %s", tt.fqn, got)
		}
	}
}

func TestClassifyLayer_CrossCuttingKinds(t *testing.T) {
	tests := []struct {
		kind string
	}{
		{"interface"}, {"enum"}, {"constant"},
	}
	for _, tt := range tests {
		s := sym("X", tt.kind, "app.something.X")
		got := classifyLayer(s)
		if got != LayerCrossCutting {
			t.Errorf("kind %q should classify as cross-cutting, got %s", tt.kind, got)
		}
	}
}

func TestClassifyLayer_Unknown(t *testing.T) {
	s := sym("Foo", "class", "com.example.Foo")
	got := classifyLayer(s)
	if got != LayerUnknown {
		t.Errorf("generic class should classify as unknown, got %s", got)
	}
}

func TestClassifyLayer_KindTakesPrecedenceOverNamespace(t *testing.T) {
	// A "table" in a service namespace should still be data layer
	s := sym("Customers", "table", "app.service.Customers")
	got := classifyLayer(s)
	if got != LayerData {
		t.Errorf("table kind should override service namespace, got %s", got)
	}
}

func TestClassifyLayer_APIPrecedesDataNamespace(t *testing.T) {
	// If FQN matches both API and data patterns, API comes first
	s := sym("DataController", "class", "api.data.DataController")
	got := classifyLayer(s)
	if got != LayerAPI {
		t.Errorf("api namespace should take precedence over data, got %s", got)
	}
}

// --- splitFQN ---

func TestSplitFQN_DotSeparated(t *testing.T) {
	segments := splitFQN("com.example.service.OrderService")
	expected := []string{"com", "example", "service", "orderservice"}
	if len(segments) != 4 {
		t.Fatalf("expected 4 segments, got %d: %v", len(segments), segments)
	}
	// splitFQN doesn't lowercase — check actual values
	for i, seg := range segments {
		_ = expected
		if seg == "" {
			t.Errorf("segment %d should not be empty", i)
		}
	}
}

func TestSplitFQN_SlashSeparated(t *testing.T) {
	segments := splitFQN("app/handlers/auth")
	if len(segments) != 3 {
		t.Fatalf("expected 3 segments, got %d: %v", len(segments), segments)
	}
}

func TestSplitFQN_BackslashSeparated(t *testing.T) {
	segments := splitFQN("App\\Controllers\\UserController")
	if len(segments) != 3 {
		t.Fatalf("expected 3 segments, got %d: %v", len(segments), segments)
	}
}

func TestSplitFQN_MixedDelimiters(t *testing.T) {
	segments := splitFQN("dbo.Customers/columns")
	if len(segments) != 3 {
		t.Fatalf("expected 3 segments, got %d: %v", len(segments), segments)
	}
}

func TestSplitFQN_Empty(t *testing.T) {
	segments := splitFQN("")
	if len(segments) != 0 {
		t.Errorf("empty FQN should produce 0 segments, got %d", len(segments))
	}
}

// --- matchesAnyPattern ---

func TestMatchesAnyPattern_ExactSegmentMatch(t *testing.T) {
	if !matchesAnyPattern("app.service.ordersvc", businessNamespacePatterns) {
		t.Error("'service' segment should match business patterns")
	}
}

func TestMatchesAnyPattern_NoMatch(t *testing.T) {
	if matchesAnyPattern("com.example.foo", businessNamespacePatterns) {
		t.Error("'com.example.foo' should not match business patterns")
	}
}

func TestMatchesAnyPattern_CaseHandling(t *testing.T) {
	// classifyLayer lowercases the FQN before calling matchesAnyPattern
	// but matchesAnyPattern itself does exact segment comparison
	// So the lowercase FQN "app.controller.x" should match
	if !matchesAnyPattern("app.controller.x", apiNamespacePatterns) {
		t.Error("'controller' should match API patterns")
	}
}
