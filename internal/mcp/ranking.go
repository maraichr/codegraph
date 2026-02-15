package mcp

import (
	"encoding/json"
	"math"
	"sort"
	"strings"

	"github.com/google/uuid"

	"github.com/maraichr/lattice/internal/mcp/session"
	"github.com/maraichr/lattice/internal/store/postgres"
)

// RankConfig controls the weights of different ranking signals.
type RankConfig struct {
	QueryRelevance  float64 // Weight for text similarity (default 0.3)
	Centrality      float64 // Weight for PageRank/degree (default 0.2)
	FocusProximity  float64 // Weight for proximity to session focus area (default 0.2)
	KindPriority    float64 // Weight for symbol kind importance (default 0.15)
	SessionNovelty  float64 // Weight for unseen symbols (default 0.15)
}

// DefaultRankConfig returns the standard ranking weights.
func DefaultRankConfig() RankConfig {
	return RankConfig{
		QueryRelevance: 0.3,
		Centrality:     0.2,
		FocusProximity: 0.2,
		KindPriority:   0.15,
		SessionNovelty: 0.15,
	}
}

// RankedSymbol pairs a symbol with its computed score.
type RankedSymbol struct {
	Symbol postgres.Symbol
	Score  float64
}

// RankSymbols scores and sorts symbols using weighted signals.
func RankSymbols(symbols []postgres.Symbol, query string, config RankConfig, sess *session.Session) []RankedSymbol {
	if len(symbols) == 0 {
		return nil
	}

	ranked := make([]RankedSymbol, len(symbols))
	for i, sym := range symbols {
		score := 0.0

		// 1. Query relevance (trigram-like name match)
		if query != "" {
			score += config.QueryRelevance * queryRelevance(sym, query)
		} else {
			score += config.QueryRelevance * 0.5 // neutral when no query
		}

		// 2. Centrality (PageRank from metadata)
		score += config.Centrality * centralityScore(sym)

		// 3. Focus proximity (closer to session focus = higher)
		if sess != nil && len(sess.FocusArea) > 0 {
			score += config.FocusProximity * focusProximityScore(sym, sess)
		} else {
			score += config.FocusProximity * 0.5
		}

		// 4. Kind priority
		score += config.KindPriority * kindPriorityScore(sym.Kind)

		// 5. Session novelty (unseen = higher)
		if sess != nil {
			score += config.SessionNovelty * noveltyScore(sym, sess)
		} else {
			score += config.SessionNovelty * 0.5
		}

		ranked[i] = RankedSymbol{Symbol: sym, Score: score}
	}

	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].Score > ranked[j].Score
	})

	return ranked
}

// queryRelevance computes a 0-1 score for how well a symbol matches the query.
func queryRelevance(sym postgres.Symbol, query string) float64 {
	q := strings.ToLower(query)
	name := strings.ToLower(sym.Name)
	fqn := strings.ToLower(sym.QualifiedName)

	// Exact name match
	if name == q {
		return 1.0
	}

	// FQN exact match
	if fqn == q {
		return 0.95
	}

	// Name contains query
	if strings.Contains(name, q) {
		return 0.8
	}

	// FQN contains query
	if strings.Contains(fqn, q) {
		return 0.6
	}

	// Query contains name (partial match)
	if strings.Contains(q, name) {
		return 0.5
	}

	// Substring overlap
	overlap := longestCommonSubstring(name, q)
	if overlap > 2 {
		return float64(overlap) / float64(max(len(name), len(q))) * 0.4
	}

	return 0.0
}

// centralityScore extracts PageRank from symbol metadata, normalized to 0-1.
func centralityScore(sym postgres.Symbol) float64 {
	if len(sym.Metadata) == 0 {
		return 0.0
	}

	var meta map[string]any
	if err := json.Unmarshal(sym.Metadata, &meta); err != nil {
		return 0.0
	}

	// Use PageRank if available
	if pr, ok := meta["pagerank"].(float64); ok {
		// PageRank values are typically small; normalize using log scale
		return math.Min(1.0, math.Log1p(pr*1000)/math.Log1p(10))
	}

	// Fall back to in-degree
	if inDeg, ok := meta["in_degree"].(float64); ok {
		return math.Min(1.0, math.Log1p(inDeg)/math.Log1p(50))
	}

	return 0.0
}

// focusProximityScore gives a higher score to symbols in the session's focus area.
func focusProximityScore(sym postgres.Symbol, sess *session.Session) float64 {
	if sess == nil || len(sess.FocusArea) == 0 {
		return 0.5
	}

	symID := sym.ID.String()
	for _, focusID := range sess.FocusArea {
		if focusID == symID {
			return 1.0
		}
	}

	// Check if symbol shares a namespace with any focus symbol
	// (This is a heuristic — real graph distance would require additional queries)
	return 0.3
}

// kindPriorityScore assigns importance weights to symbol kinds.
func kindPriorityScore(kind string) float64 {
	priorities := map[string]float64{
		"table":     1.0,
		"class":     0.95,
		"view":      0.9,
		"interface": 0.85,
		"procedure": 0.85,
		"module":    0.8,
		"package":   0.8,
		"method":    0.7,
		"function":  0.7,
		"type":      0.65,
		"enum":      0.6,
		"trigger":   0.6,
		"property":  0.5,
		"field":     0.45,
		"variable":  0.4,
		"constant":  0.4,
		"column":    0.35,
	}

	if p, ok := priorities[strings.ToLower(kind)]; ok {
		return p
	}
	return 0.3
}

// noveltyScore gives higher scores to symbols not yet seen in the session.
func noveltyScore(sym postgres.Symbol, sess *session.Session) float64 {
	if sess == nil {
		return 0.5
	}
	if sess.IsSeen(sym.ID) {
		return 0.1
	}
	return 1.0
}

// longestCommonSubstring returns the length of the longest common substring.
func longestCommonSubstring(a, b string) int {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}

	maxLen := 0
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)

	for i := range len(a) {
		for j := range len(b) {
			if a[i] == b[j] {
				curr[j+1] = prev[j] + 1
				if curr[j+1] > maxLen {
					maxLen = curr[j+1]
				}
			} else {
				curr[j+1] = 0
			}
		}
		prev, curr = curr, prev
		clear(curr)
	}

	return maxLen
}

// FilterAndRank applies session dedup, ranks, and returns results with counts.
func FilterAndRank(
	symbols []postgres.Symbol,
	query string,
	config RankConfig,
	sess *session.Session,
	limit int,
) (results []RankedSymbol, total int, seenCount int) {
	total = len(symbols)

	if sess != nil {
		for _, sym := range symbols {
			if sess.IsSeen(sym.ID) {
				seenCount++
			}
		}
	}

	ranked := RankSymbols(symbols, query, config, sess)

	if limit > 0 && len(ranked) > limit {
		ranked = ranked[:limit]
	}

	return ranked, total, seenCount
}

// SymbolIDsFromRanked extracts UUIDs from ranked results.
func SymbolIDsFromRanked(ranked []RankedSymbol) []uuid.UUID {
	ids := make([]uuid.UUID, len(ranked))
	for i, r := range ranked {
		ids[i] = r.Symbol.ID
	}
	return ids
}
