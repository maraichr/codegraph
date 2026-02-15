package analytics

import (
	"testing"

	"github.com/maraichr/lattice/internal/store/postgres"
)

// --- generateProjectSummary ---

func TestGenerateProjectSummary_Basic(t *testing.T) {
	stats := postgres.GetProjectSymbolStatsRow{
		TotalSymbols:  1000,
		FileCount:     50,
		LanguageCount: 3,
		KindCount:     8,
	}
	langCounts := []postgres.GetSymbolCountsByLanguageRow{
		{Language: "go", Cnt: 500},
		{Language: "tsql", Cnt: 300},
		{Language: "java", Cnt: 200},
	}
	kindCounts := []postgres.GetSymbolCountsByKindRow{
		{Kind: "function", Cnt: 400},
		{Kind: "class", Cnt: 200},
		{Kind: "table", Cnt: 100},
	}

	summary := generateProjectSummary(stats, langCounts, kindCounts, 2500)

	if summary == "" {
		t.Fatal("summary should not be empty")
	}
	// Should contain key stats
	assertContains(t, summary, "1000")
	assertContains(t, summary, "50")
	assertContains(t, summary, "2500")
	assertContains(t, summary, "go")
	assertContains(t, summary, "function")
}

func TestGenerateProjectSummary_TruncatesAt5Languages(t *testing.T) {
	stats := postgres.GetProjectSymbolStatsRow{TotalSymbols: 100, FileCount: 10}
	langs := make([]postgres.GetSymbolCountsByLanguageRow, 8)
	for i := range langs {
		langs[i] = postgres.GetSymbolCountsByLanguageRow{Language: "lang" + string(rune('A'+i)), Cnt: 10}
	}

	summary := generateProjectSummary(stats, langs, nil, 100)
	assertContains(t, summary, "and 3 more")
}

func TestGenerateProjectSummary_EmptyLanguages(t *testing.T) {
	stats := postgres.GetProjectSymbolStatsRow{TotalSymbols: 0, FileCount: 0}
	summary := generateProjectSummary(stats, nil, nil, 0)
	if summary == "" {
		t.Error("should produce at least the base summary")
	}
}

func TestGenerateProjectSummary_TruncatesAt5Kinds(t *testing.T) {
	stats := postgres.GetProjectSymbolStatsRow{TotalSymbols: 100, FileCount: 10}
	kinds := make([]postgres.GetSymbolCountsByKindRow, 7)
	for i := range kinds {
		kinds[i] = postgres.GetSymbolCountsByKindRow{Kind: "kind" + string(rune('A'+i)), Cnt: 5}
	}

	summary := generateProjectSummary(stats, nil, kinds, 50)
	assertContains(t, summary, "and 2 more")
}

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if len(s) == 0 || len(substr) == 0 {
		return
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return
		}
	}
	t.Errorf("expected %q to contain %q", s, substr)
}
