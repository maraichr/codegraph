package tools

import (
	"testing"

	"github.com/maraichr/codegraph/internal/mcp"
	"github.com/maraichr/codegraph/internal/store/postgres"
)

// --- classifyIntent ---

func TestClassifyIntent_Impact(t *testing.T) {
	tests := []string{
		"What breaks if I rename Customers?",
		"What happens if I delete this table?",
		"What is the impact of changing this function?",
		"Show me the blast radius of modifying OrderService",
		"What would be affected by removing this column?",
	}
	for _, q := range tests {
		if classifyIntent(q) != IntentImpact {
			t.Errorf("expected IntentImpact for %q, got %s", q, classifyIntent(q))
		}
	}
}

func TestClassifyIntent_Lineage(t *testing.T) {
	tests := []string{
		"Where does the data flow from Customers.Email?",
		"Show me the lineage of this column",
		"Where does this data come from?",
		"What transforms this field?",
		"What populates the OrderTotal column?",
	}
	for _, q := range tests {
		if classifyIntent(q) != IntentLineage {
			t.Errorf("expected IntentLineage for %q, got %s", q, classifyIntent(q))
		}
	}
}

func TestClassifyIntent_Overview(t *testing.T) {
	tests := []string{
		"Give me an overview of this project",
		"What is this codebase?",
		"Describe the architecture",
		"Show me a summary",
		"What languages are used?",
		"How big is this project?",
	}
	for _, q := range tests {
		if classifyIntent(q) != IntentOverview {
			t.Errorf("expected IntentOverview for %q, got %s", q, classifyIntent(q))
		}
	}
}

func TestClassifyIntent_Dependencies(t *testing.T) {
	tests := []string{
		"What depends on this table?",
		"Show me the dependencies of OrderService",
		"What calls this function?",
		"What uses this module?",
		"What imports this package?",
	}
	for _, q := range tests {
		if classifyIntent(q) != IntentDeps {
			t.Errorf("expected IntentDeps for %q, got %s", q, classifyIntent(q))
		}
	}
}

func TestClassifyIntent_Subgraph(t *testing.T) {
	tests := []string{
		"Show me everything about order processing",
		"What is the order processing pipeline?",
		"Show me the authentication workflow",
		"Tell me about the payment module",
	}
	for _, q := range tests {
		if classifyIntent(q) != IntentSubgraph {
			t.Errorf("expected IntentSubgraph for %q, got %s", q, classifyIntent(q))
		}
	}
}

func TestClassifyIntent_Search_Default(t *testing.T) {
	tests := []string{
		"CustomerRepository",
		"dbo.Customers",
		"Find the login handler",
	}
	for _, q := range tests {
		if classifyIntent(q) != IntentSearch {
			t.Errorf("expected IntentSearch for %q, got %s", q, classifyIntent(q))
		}
	}
}

// --- extractSearchTerms ---

func TestExtractSearchTerms_RemovesStopWords(t *testing.T) {
	result := extractSearchTerms("What is the CustomerRepository?")
	if result != "customerrepository" {
		t.Errorf("expected 'customerrepository', got %q", result)
	}
}

func TestExtractSearchTerms_PreservesSubstantiveWords(t *testing.T) {
	result := extractSearchTerms("Show me the order processing pipeline")
	// "show" and "me" and "the" are stop words
	if result == "" {
		t.Error("should preserve substantive words")
	}
	if contains(result, "show") || contains(result, "the") {
		t.Errorf("should remove stop words, got %q", result)
	}
}

func TestExtractSearchTerms_EmptyAfterStopWords(t *testing.T) {
	result := extractSearchTerms("what is it?")
	// If all words are stop words, return original
	if result != "what is it?" {
		t.Errorf("all-stop-word input should return original, got %q", result)
	}
}

func TestExtractSearchTerms_StripsPunctuation(t *testing.T) {
	result := extractSearchTerms("What about 'Customers'?")
	if contains(result, "'") || contains(result, "?") {
		t.Errorf("should strip punctuation, got %q", result)
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// --- isLowValue ---

func TestIsLowValue_LowPageRankColumn(t *testing.T) {
	sym := postgres.Symbol{Kind: "column"}
	if !isLowValue(sym) {
		t.Error("column with no pagerank should be low value")
	}
}

func TestIsLowValue_Table(t *testing.T) {
	sym := postgres.Symbol{Kind: "table"}
	if isLowValue(sym) {
		t.Error("table should never be low value")
	}
}

// --- symbolTokenEstimate ---

func TestSymbolTokenEstimate(t *testing.T) {
	if symbolTokenEstimate(mcp.VerbositySummary) != 30 {
		t.Error("summary should estimate 30 tokens")
	}
	if symbolTokenEstimate(mcp.VerbosityStandard) != 60 {
		t.Error("standard should estimate 60 tokens")
	}
	if symbolTokenEstimate(mcp.VerbosityFull) != 120 {
		t.Error("full should estimate 120 tokens")
	}
}

// --- estimateSubgraphTokens ---

func TestEstimateSubgraphTokens(t *testing.T) {
	symbols := make([]postgres.Symbol, 10)
	edges := make([]subgraphEdge, 5)
	tokens := estimateSubgraphTokens(symbols, edges, mcp.VerbosityStandard)
	expected := 10*60 + 5*15 + 100
	if tokens != expected {
		t.Errorf("expected %d tokens, got %d", expected, tokens)
	}
}

// --- identifyCore ---

func TestIdentifyCore(t *testing.T) {
	seeds := []postgres.Symbol{
		{ID: [16]byte{1}},
		{ID: [16]byte{2}},
	}
	subgraph := append(seeds, postgres.Symbol{ID: [16]byte{3}})
	core := identifyCore(seeds, subgraph)

	if !core[[16]byte{1}] || !core[[16]byte{2}] {
		t.Error("seed symbols should be core")
	}
	if core[[16]byte{3}] {
		t.Error("non-seed symbols should not be core")
	}
}
