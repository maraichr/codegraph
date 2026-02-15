package session

import (
	"testing"

	"github.com/google/uuid"
)

// --- Session creation ---

func TestNewSession_Initialized(t *testing.T) {
	sess := newSession("test-id")
	if sess.ID != "test-id" {
		t.Errorf("session ID should be 'test-id', got %q", sess.ID)
	}
	if sess.SeenSymbols == nil {
		t.Error("SeenSymbols should be initialized")
	}
	if sess.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
}

// --- MarkSeen / IsSeen ---

func TestMarkSeen_SingleSymbol(t *testing.T) {
	sess := newSession("test")
	id := uuid.New()
	sess.MarkSeen(id)
	if !sess.IsSeen(id) {
		t.Error("symbol should be seen after MarkSeen")
	}
}

func TestMarkSeen_MultipleSymbols(t *testing.T) {
	sess := newSession("test")
	ids := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	sess.MarkSeen(ids...)
	for _, id := range ids {
		if !sess.IsSeen(id) {
			t.Errorf("symbol %s should be seen", id)
		}
	}
}

func TestIsSeen_UnseenSymbol(t *testing.T) {
	sess := newSession("test")
	if sess.IsSeen(uuid.New()) {
		t.Error("unseen symbol should return false")
	}
}

func TestIsSeen_NilMap(t *testing.T) {
	sess := &Session{}
	if sess.IsSeen(uuid.New()) {
		t.Error("nil map should return false")
	}
}

func TestMarkSeen_NilMap(t *testing.T) {
	sess := &Session{}
	id := uuid.New()
	sess.MarkSeen(id) // should not panic
	if !sess.IsSeen(id) {
		t.Error("should work even when starting from nil map")
	}
}

func TestSeenCount(t *testing.T) {
	sess := newSession("test")
	sess.MarkSeen(uuid.New(), uuid.New(), uuid.New())
	if sess.SeenCount() != 3 {
		t.Errorf("seen count should be 3, got %d", sess.SeenCount())
	}
}

// --- AddQuery ---

func TestAddQuery_AddsToHistory(t *testing.T) {
	sess := newSession("test")
	sess.AddQuery("search for customers")
	if len(sess.QueryHistory) != 1 {
		t.Errorf("query history should have 1 entry, got %d", len(sess.QueryHistory))
	}
	if sess.QueryHistory[0] != "search for customers" {
		t.Errorf("query should be preserved")
	}
}

func TestAddQuery_TruncatesHistory(t *testing.T) {
	sess := newSession("test")
	for i := range 25 {
		sess.AddQuery("query " + string(rune('A'+i)))
	}
	if len(sess.QueryHistory) != maxQueryHistory {
		t.Errorf("query history should be capped at %d, got %d", maxQueryHistory, len(sess.QueryHistory))
	}
	// Oldest queries should be dropped
	if sess.QueryHistory[0] == "query A" {
		t.Error("oldest query should have been trimmed")
	}
}

// --- UpdateFocus ---

func TestUpdateFocus_AddsSymbols(t *testing.T) {
	sess := newSession("test")
	id1, id2 := uuid.New(), uuid.New()
	sess.UpdateFocus(id1, id2)
	if len(sess.FocusArea) != 2 {
		t.Errorf("focus area should have 2 entries, got %d", len(sess.FocusArea))
	}
}

func TestUpdateFocus_TruncatesOldest(t *testing.T) {
	sess := newSession("test")
	for range 15 {
		sess.UpdateFocus(uuid.New())
	}
	if len(sess.FocusArea) != maxFocusArea {
		t.Errorf("focus area should be capped at %d, got %d", maxFocusArea, len(sess.FocusArea))
	}
}

func TestFocusAreaUUIDs(t *testing.T) {
	sess := newSession("test")
	id := uuid.New()
	sess.UpdateFocus(id)
	uuids := sess.FocusAreaUUIDs()
	if len(uuids) != 1 || uuids[0] != id {
		t.Error("FocusAreaUUIDs should parse stored IDs correctly")
	}
}

func TestFocusAreaUUIDs_InvalidEntries(t *testing.T) {
	sess := newSession("test")
	sess.FocusArea = []string{"not-a-uuid", uuid.New().String()}
	uuids := sess.FocusAreaUUIDs()
	if len(uuids) != 1 {
		t.Errorf("should skip invalid UUIDs, got %d results", len(uuids))
	}
}

// --- Waypoints ---

func TestAddWaypoint(t *testing.T) {
	sess := newSession("test")
	id := uuid.New()
	sess.AddWaypoint(id, "key table")
	if len(sess.Waypoints) != 1 {
		t.Fatalf("expected 1 waypoint, got %d", len(sess.Waypoints))
	}
	wp := sess.Waypoints[0]
	if wp.SymbolID != id {
		t.Error("waypoint symbol ID mismatch")
	}
	if wp.Label != "key table" {
		t.Error("waypoint label mismatch")
	}
	if wp.AddedAt.IsZero() {
		t.Error("waypoint timestamp should be set")
	}
}

// --- Recap ---

func TestAddRecap(t *testing.T) {
	sess := newSession("test")
	sess.AddRecap("Found CustomerRepository reads dbo.Customers")
	if len(sess.Recap) != 1 {
		t.Error("should have 1 recap entry")
	}
}

func TestAddRecap_TruncatesToTokenLimit(t *testing.T) {
	sess := newSession("test")
	// Add many long findings to exceed token limit
	for range 100 {
		sess.AddRecap("Found a very long finding that contains many words and will consume many tokens in the recap buffer of the session")
	}
	tokens := estimateTokens(sess.Recap)
	if tokens > maxRecapTokens {
		t.Errorf("recap tokens %d should not exceed %d", tokens, maxRecapTokens)
	}
	if len(sess.Recap) == 100 {
		t.Error("old recap entries should have been trimmed")
	}
}

func TestRecapText_Empty(t *testing.T) {
	sess := newSession("test")
	if sess.RecapText() != "" {
		t.Error("empty recap should return empty string")
	}
}

func TestRecapText_Formatted(t *testing.T) {
	sess := newSession("test")
	sess.AddRecap("First finding")
	sess.AddRecap("Second finding")
	text := sess.RecapText()
	if text == "" {
		t.Fatal("recap text should not be empty")
	}
	// Should be numbered
	if text[0] != '1' {
		t.Error("recap should start with numbered entry")
	}
}

// --- estimateTokens ---

func TestEstimateTokens(t *testing.T) {
	lines := []string{"hello world"} // 11 chars -> 2 tokens
	tokens := estimateTokens(lines)
	if tokens != 2 {
		t.Errorf("expected 2 tokens, got %d", tokens)
	}
}

func TestEstimateTokens_Empty(t *testing.T) {
	if estimateTokens(nil) != 0 {
		t.Error("nil should return 0")
	}
}
