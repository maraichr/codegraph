package mcp

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/maraichr/lattice/internal/mcp/session"
	"github.com/maraichr/lattice/internal/store/postgres"
)

func testSymbol(name, kind, fqn, lang string) postgres.Symbol {
	return postgres.Symbol{
		ID:            uuid.New(),
		ProjectID:     uuid.New(),
		FileID:        uuid.New(),
		Name:          name,
		QualifiedName: fqn,
		Kind:          kind,
		Language:      lang,
		StartLine:     10,
		EndLine:       50,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

// --- ParseVerbosity ---

func TestParseVerbosity_Defaults(t *testing.T) {
	tests := []struct {
		input    string
		expected Verbosity
	}{
		{"summary", VerbositySummary},
		{"SUMMARY", VerbositySummary},
		{"full", VerbosityFull},
		{"Full", VerbosityFull},
		{"standard", VerbosityStandard},
		{"", VerbosityStandard},
		{"unknown", VerbosityStandard},
	}

	for _, tt := range tests {
		got := ParseVerbosity(tt.input)
		if got != tt.expected {
			t.Errorf("ParseVerbosity(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

// --- ResponseBuilder ---

func TestResponseBuilder_DefaultMaxTokens(t *testing.T) {
	rb := NewResponseBuilder(0)
	if rb.maxTokens != defaultMaxTokens {
		t.Errorf("default max tokens should be %d, got %d", defaultMaxTokens, rb.maxTokens)
	}
}

func TestResponseBuilder_CustomMaxTokens(t *testing.T) {
	rb := NewResponseBuilder(1000)
	if rb.maxTokens != 1000 {
		t.Errorf("custom max tokens should be 1000, got %d", rb.maxTokens)
	}
}

func TestResponseBuilder_AddHeader(t *testing.T) {
	rb := NewResponseBuilder(1000)
	rb.AddHeader("# Test Header")
	result := rb.Finalize(0, 0)
	if !strings.Contains(result, "# Test Header") {
		t.Error("header should be present in output")
	}
	if rb.TokenEstimate() == 0 {
		t.Error("token estimate should be positive after adding header")
	}
}

func TestResponseBuilder_AddLine(t *testing.T) {
	rb := NewResponseBuilder(1000)
	ok := rb.AddLine("test line")
	if !ok {
		t.Error("adding small line within budget should succeed")
	}
	if !strings.Contains(rb.Finalize(0, 0), "test line") {
		t.Error("line should be present in output")
	}
}

func TestResponseBuilder_AddLine_BudgetExceeded(t *testing.T) {
	rb := NewResponseBuilder(5) // Very small budget
	rb.AddLine("short") // This might fit
	ok := rb.AddLine(strings.Repeat("x", 100))
	if ok {
		t.Error("adding line exceeding budget should fail")
	}
	if !rb.IsTruncated() {
		t.Error("should be marked as truncated")
	}
}

func TestResponseBuilder_AddSymbolCard_Summary(t *testing.T) {
	rb := NewResponseBuilder(2000)
	sym := testSymbol("Customers", "table", "dbo.Customers", "tsql")
	ok := rb.AddSymbolCard(sym, VerbositySummary, nil)
	if !ok {
		t.Error("should succeed within budget")
	}
	result := rb.Finalize(1, 1)
	if !strings.Contains(result, "Customers") {
		t.Error("should contain symbol name")
	}
	if !strings.Contains(result, "dbo.Customers") {
		t.Error("should contain FQN")
	}
	if rb.ItemCount() != 1 {
		t.Errorf("item count should be 1, got %d", rb.ItemCount())
	}
}

func TestResponseBuilder_AddSymbolCard_Standard(t *testing.T) {
	rb := NewResponseBuilder(2000)
	sig := "CREATE PROCEDURE dbo.GetCustomer"
	sym := testSymbol("GetCustomer", "procedure", "dbo.GetCustomer", "tsql")
	sym.Signature = &sig

	ok := rb.AddSymbolCard(sym, VerbosityStandard, nil)
	if !ok {
		t.Error("should succeed within budget")
	}
	result := rb.Finalize(1, 1)
	if !strings.Contains(result, "tsql") {
		t.Error("standard verbosity should include language")
	}
	if !strings.Contains(result, sig) {
		t.Error("standard verbosity should include signature")
	}
}

func TestResponseBuilder_AddSymbolCard_Full(t *testing.T) {
	rb := NewResponseBuilder(2000)
	sig := "func (r *Repo) GetByID(id int) (*Customer, error)"
	doc := "GetByID retrieves a customer by primary key."
	sym := testSymbol("GetByID", "method", "app.Repo.GetByID", "go")
	sym.Signature = &sig
	sym.DocComment = &doc

	ok := rb.AddSymbolCard(sym, VerbosityFull, nil)
	if !ok {
		t.Error("should succeed within budget")
	}
	result := rb.Finalize(1, 1)
	if !strings.Contains(result, doc) {
		t.Error("full verbosity should include doc comment")
	}
	if !strings.Contains(result, "L10") {
		t.Error("full verbosity should include location")
	}
}

func TestResponseBuilder_AddSymbolCard_SeenMarker(t *testing.T) {
	rb := NewResponseBuilder(2000)
	sym := testSymbol("Foo", "class", "app.Foo", "go")
	sess := &session.Session{SeenSymbols: map[string]bool{sym.ID.String(): true}}

	rb.AddSymbolCard(sym, VerbositySummary, sess)
	result := rb.Finalize(1, 1)
	if !strings.Contains(result, "seen") {
		t.Error("seen symbol should be annotated")
	}
}

func TestResponseBuilder_AddSymbolStub(t *testing.T) {
	rb := NewResponseBuilder(2000)
	sym := testSymbol("Foo", "class", "app.Foo", "go")
	ok := rb.AddSymbolStub(sym)
	if !ok {
		t.Error("stub should fit in budget")
	}
	result := rb.Finalize(1, 1)
	if !strings.Contains(result, "already examined") {
		t.Error("stub should contain 'already examined'")
	}
	if !strings.Contains(result, sym.ID.String()) {
		t.Error("stub should contain symbol ID")
	}
}

func TestResponseBuilder_AddSection(t *testing.T) {
	rb := NewResponseBuilder(2000)
	ok := rb.AddSection("Dependencies", "- A calls B\n- B reads C")
	if !ok {
		t.Error("section should fit in budget")
	}
	result := rb.Finalize(0, 0)
	if !strings.Contains(result, "### Dependencies") {
		t.Error("section should contain heading")
	}
}

func TestResponseBuilder_Finalize_TruncationNotice(t *testing.T) {
	rb := NewResponseBuilder(2000)
	result := rb.Finalize(100, 10) // showing 10 of 100
	if !strings.Contains(result, "10 of 100") {
		t.Error("truncation notice should show counts")
	}
}

func TestResponseBuilder_Finalize_NoTruncationWhenComplete(t *testing.T) {
	rb := NewResponseBuilder(2000)
	result := rb.Finalize(5, 5)
	if strings.Contains(result, "truncated") && strings.Contains(result, "Showing") {
		t.Error("no truncation notice when all results returned")
	}
}

func TestResponseBuilder_FinalizeWithHints(t *testing.T) {
	rb := NewResponseBuilder(2000)
	rb.AddLine("result line")
	hints := &NavigationHints{
		Steps: []NavigationStep{
			{Tool: "get_dependencies", Description: "Explore deps of Foo", EstimatedTokens: 600},
		},
	}
	result := rb.FinalizeWithHints(1, 1, hints)
	if !strings.Contains(result, "Next steps:") {
		t.Error("should contain navigation hints section")
	}
	if !strings.Contains(result, "get_dependencies") {
		t.Error("should contain suggested tool")
	}
	if !strings.Contains(result, "600 tokens") {
		t.Error("should contain token estimate")
	}
}

func TestResponseBuilder_FinalizeWithHints_NilHints(t *testing.T) {
	rb := NewResponseBuilder(2000)
	rb.AddLine("result")
	result := rb.FinalizeWithHints(1, 1, nil)
	if strings.Contains(result, "Next steps:") {
		t.Error("nil hints should not produce next steps section")
	}
}

// --- DryRunResult ---

func TestFormatDryRun(t *testing.T) {
	result := FormatDryRun(DryRunResult{
		SymbolCount:     47,
		EdgeCount:       62,
		EstimatedTokens: 4200,
		DepthReached:    8,
	})
	if !strings.Contains(result, "47") {
		t.Error("should contain symbol count")
	}
	if !strings.Contains(result, "62") {
		t.Error("should contain edge count")
	}
	if !strings.Contains(result, "4200") {
		t.Error("should contain token estimate")
	}
	if !strings.Contains(result, "8") {
		t.Error("should contain depth reached")
	}
}

func TestFormatDryRun_NoDepth(t *testing.T) {
	result := FormatDryRun(DryRunResult{SymbolCount: 5, EdgeCount: 3, EstimatedTokens: 200})
	if strings.Contains(result, "Depth") {
		t.Error("should not show depth when 0")
	}
}

// --- Token budget stress test ---

func TestResponseBuilder_ManyCards_RespectsBudget(t *testing.T) {
	rb := NewResponseBuilder(500) // Tight budget
	added := 0
	for i := range 100 {
		sym := testSymbol("Sym"+string(rune('A'+i%26)), "class", "app.Sym", "go")
		if rb.AddSymbolCard(sym, VerbositySummary, nil) {
			added++
		}
	}
	if added >= 100 {
		t.Error("should have been truncated before adding 100 cards")
	}
	if !rb.IsTruncated() {
		t.Error("should be marked truncated")
	}
	if rb.TokenEstimate() > 500 {
		t.Errorf("token estimate %d should not exceed budget 500", rb.TokenEstimate())
	}
}
