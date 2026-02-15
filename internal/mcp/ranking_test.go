package mcp

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/maraichr/codegraph/internal/mcp/session"
	"github.com/maraichr/codegraph/internal/store/postgres"
)

func makeSymbol(name, kind, fqn string) postgres.Symbol {
	return postgres.Symbol{
		ID:            uuid.New(),
		ProjectID:     uuid.New(),
		FileID:        uuid.New(),
		Name:          name,
		QualifiedName: fqn,
		Kind:          kind,
		Language:      "go",
		StartLine:     1,
		EndLine:       50,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

func makeSymbolWithMeta(name, kind, fqn string, meta map[string]any) postgres.Symbol {
	sym := makeSymbol(name, kind, fqn)
	if meta != nil {
		sym.Metadata, _ = json.Marshal(meta)
	}
	return sym
}

// --- queryRelevance ---

func TestQueryRelevance_ExactNameMatch(t *testing.T) {
	sym := makeSymbol("Customers", "table", "dbo.Customers")
	score := queryRelevance(sym, "Customers")
	if score != 1.0 {
		t.Errorf("exact name match should score 1.0, got %f", score)
	}
}

func TestQueryRelevance_CaseInsensitive(t *testing.T) {
	sym := makeSymbol("Customers", "table", "dbo.Customers")
	score := queryRelevance(sym, "customers")
	if score != 1.0 {
		t.Errorf("case-insensitive exact match should score 1.0, got %f", score)
	}
}

func TestQueryRelevance_FQNExactMatch(t *testing.T) {
	sym := makeSymbol("Customers", "table", "dbo.Customers")
	score := queryRelevance(sym, "dbo.Customers")
	if score != 0.95 {
		t.Errorf("FQN exact match should score 0.95, got %f", score)
	}
}

func TestQueryRelevance_NameContainsQuery(t *testing.T) {
	sym := makeSymbol("CustomerRepository", "class", "app.data.CustomerRepository")
	score := queryRelevance(sym, "customer")
	if score != 0.8 {
		t.Errorf("name contains query should score 0.8, got %f", score)
	}
}

func TestQueryRelevance_FQNContainsQuery(t *testing.T) {
	sym := makeSymbol("Repository", "class", "app.data.CustomerRepository")
	score := queryRelevance(sym, "customer")
	// Name "repository" does not contain "customer", but FQN does
	if score != 0.6 {
		t.Errorf("FQN contains query should score 0.6, got %f", score)
	}
}

func TestQueryRelevance_NoMatch(t *testing.T) {
	sym := makeSymbol("OrderService", "class", "app.services.OrderService")
	score := queryRelevance(sym, "xyz")
	if score != 0.0 {
		t.Errorf("no match should score 0.0, got %f", score)
	}
}

func TestQueryRelevance_SubstringOverlap(t *testing.T) {
	sym := makeSymbol("CustomerAddress", "class", "app.CustomerAddress")
	score := queryRelevance(sym, "customerphone")
	// "customer" is common substring (len 8), max(len("customeraddress"),len("customerphone"))=15
	// score = 8/15 * 0.4 ≈ 0.213
	if score < 0.1 || score > 0.4 {
		t.Errorf("substring overlap should score between 0.1-0.4, got %f", score)
	}
}

// --- centralityScore ---

func TestCentralityScore_NoMetadata(t *testing.T) {
	sym := makeSymbol("Foo", "class", "app.Foo")
	score := centralityScore(sym)
	if score != 0.0 {
		t.Errorf("no metadata should score 0.0, got %f", score)
	}
}

func TestCentralityScore_WithPageRank(t *testing.T) {
	sym := makeSymbolWithMeta("Foo", "class", "app.Foo", map[string]any{
		"pagerank": 0.05,
	})
	score := centralityScore(sym)
	if score <= 0.0 || score > 1.0 {
		t.Errorf("pagerank 0.05 should produce score in (0, 1], got %f", score)
	}
}

func TestCentralityScore_HighPageRank(t *testing.T) {
	low := makeSymbolWithMeta("Lo", "class", "app.Lo", map[string]any{"pagerank": 0.001})
	high := makeSymbolWithMeta("Hi", "class", "app.Hi", map[string]any{"pagerank": 0.1})
	lo := centralityScore(low)
	hi := centralityScore(high)
	if hi <= lo {
		t.Errorf("higher pagerank should give higher score: hi=%f, lo=%f", hi, lo)
	}
}

func TestCentralityScore_FallbackToInDegree(t *testing.T) {
	sym := makeSymbolWithMeta("Foo", "class", "app.Foo", map[string]any{
		"in_degree": float64(25),
	})
	score := centralityScore(sym)
	if score <= 0.0 {
		t.Errorf("in_degree 25 should produce positive score, got %f", score)
	}
}

// --- kindPriorityScore ---

func TestKindPriorityScore_Table(t *testing.T) {
	score := kindPriorityScore("table")
	if score != 1.0 {
		t.Errorf("table should have highest priority 1.0, got %f", score)
	}
}

func TestKindPriorityScore_Column(t *testing.T) {
	score := kindPriorityScore("column")
	if score != 0.35 {
		t.Errorf("column should score 0.35, got %f", score)
	}
}

func TestKindPriorityScore_Unknown(t *testing.T) {
	score := kindPriorityScore("widget")
	if score != 0.3 {
		t.Errorf("unknown kind should score 0.3, got %f", score)
	}
}

func TestKindPriorityScore_CaseInsensitive(t *testing.T) {
	score := kindPriorityScore("TABLE")
	if score != 1.0 {
		t.Errorf("TABLE (uppercase) should match table, got %f", score)
	}
}

// --- noveltyScore ---

func TestNoveltyScore_NilSession(t *testing.T) {
	sym := makeSymbol("Foo", "class", "app.Foo")
	score := noveltyScore(sym, nil)
	if score != 0.5 {
		t.Errorf("nil session should score 0.5, got %f", score)
	}
}

func TestNoveltyScore_UnseenSymbol(t *testing.T) {
	sym := makeSymbol("Foo", "class", "app.Foo")
	sess := &session.Session{SeenSymbols: make(map[string]bool)}
	score := noveltyScore(sym, sess)
	if score != 1.0 {
		t.Errorf("unseen symbol should score 1.0, got %f", score)
	}
}

func TestNoveltyScore_SeenSymbol(t *testing.T) {
	sym := makeSymbol("Foo", "class", "app.Foo")
	sess := &session.Session{SeenSymbols: map[string]bool{sym.ID.String(): true}}
	score := noveltyScore(sym, sess)
	if score != 0.1 {
		t.Errorf("seen symbol should score 0.1, got %f", score)
	}
}

// --- longestCommonSubstring ---

func TestLCS_Identical(t *testing.T) {
	if longestCommonSubstring("hello", "hello") != 5 {
		t.Error("identical strings should return full length")
	}
}

func TestLCS_NoOverlap(t *testing.T) {
	if longestCommonSubstring("abc", "xyz") != 0 {
		t.Error("no overlap should return 0")
	}
}

func TestLCS_PartialOverlap(t *testing.T) {
	result := longestCommonSubstring("customer", "customerphone")
	if result != 8 {
		t.Errorf("expected 8, got %d", result)
	}
}

func TestLCS_Empty(t *testing.T) {
	if longestCommonSubstring("", "hello") != 0 {
		t.Error("empty string should return 0")
	}
}

// --- RankSymbols ---

func TestRankSymbols_Empty(t *testing.T) {
	result := RankSymbols(nil, "test", DefaultRankConfig(), nil)
	if result != nil {
		t.Errorf("nil input should return nil, got %v", result)
	}
}

func TestRankSymbols_SortsByScore(t *testing.T) {
	sym1 := makeSymbol("OrderService", "class", "app.services.OrderService")
	sym2 := makeSymbol("Customer", "table", "dbo.Customer") // exact match + high kind priority

	ranked := RankSymbols([]postgres.Symbol{sym1, sym2}, "Customer", DefaultRankConfig(), nil)
	if len(ranked) != 2 {
		t.Fatalf("expected 2 results, got %d", len(ranked))
	}
	if ranked[0].Symbol.Name != "Customer" {
		t.Errorf("Customer should rank first (exact match), got %s", ranked[0].Symbol.Name)
	}
	if ranked[0].Score <= ranked[1].Score {
		t.Error("first result should have higher score than second")
	}
}

func TestRankSymbols_CentralityBoost(t *testing.T) {
	low := makeSymbolWithMeta("FooService", "class", "app.FooService", nil)
	high := makeSymbolWithMeta("BarService", "class", "app.BarService", map[string]any{
		"pagerank": 0.5,
	})

	// Use centrality-only config
	config := RankConfig{Centrality: 1.0}
	ranked := RankSymbols([]postgres.Symbol{low, high}, "", config, nil)
	if ranked[0].Symbol.Name != "BarService" {
		t.Errorf("BarService (high centrality) should rank first, got %s", ranked[0].Symbol.Name)
	}
}

func TestRankSymbols_SessionNoveltyBoost(t *testing.T) {
	seen := makeSymbol("Seen", "class", "app.Seen")
	unseen := makeSymbol("Unseen", "class", "app.Unseen")
	sess := &session.Session{SeenSymbols: map[string]bool{seen.ID.String(): true}}

	config := RankConfig{SessionNovelty: 1.0}
	ranked := RankSymbols([]postgres.Symbol{seen, unseen}, "", config, sess)
	if ranked[0].Symbol.Name != "Unseen" {
		t.Errorf("unseen symbol should rank first, got %s", ranked[0].Symbol.Name)
	}
}

// --- FilterAndRank ---

func TestFilterAndRank_LimitApplied(t *testing.T) {
	syms := []postgres.Symbol{
		makeSymbol("A", "class", "app.A"),
		makeSymbol("B", "class", "app.B"),
		makeSymbol("C", "class", "app.C"),
	}

	results, total, _ := FilterAndRank(syms, "A", DefaultRankConfig(), nil, 2)
	if total != 3 {
		t.Errorf("total should be 3, got %d", total)
	}
	if len(results) != 2 {
		t.Errorf("limit 2 should return 2 results, got %d", len(results))
	}
}

func TestFilterAndRank_SeenCount(t *testing.T) {
	sym1 := makeSymbol("A", "class", "app.A")
	sym2 := makeSymbol("B", "class", "app.B")
	sess := &session.Session{SeenSymbols: map[string]bool{sym1.ID.String(): true}}

	_, _, seenCount := FilterAndRank([]postgres.Symbol{sym1, sym2}, "", DefaultRankConfig(), sess, 0)
	if seenCount != 1 {
		t.Errorf("seen count should be 1, got %d", seenCount)
	}
}

// --- SymbolIDsFromRanked ---

func TestSymbolIDsFromRanked(t *testing.T) {
	sym := makeSymbol("A", "class", "app.A")
	ranked := []RankedSymbol{{Symbol: sym, Score: 1.0}}
	ids := SymbolIDsFromRanked(ranked)
	if len(ids) != 1 || ids[0] != sym.ID {
		t.Error("should extract the correct UUID")
	}
}
