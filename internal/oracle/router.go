package oracle

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/maraichr/codegraph/internal/llm"
)

const routerSystemPrompt = `You route codebase questions to tools. Reply with ONLY a JSON object, nothing else.

Tools:
- search: Find symbols. Params: {"query":"..."}
- ranking: Top/most-used symbols. Params: {"kinds":["table"],"metric":"in_degree"}
- overview: Project summary. Params: {}
- subgraph: Connected module around topic. Params: {"topic":"..."}
- relationships: FK/joins between tables. Params: {"topic":"..."}
- lineage: Data flow for a symbol. Params: {"symbol_name":"...","direction":"both"}
- impact: What breaks if symbol changes. Params: {"symbol_name":"...","change_type":"modify"}

Examples:
User: "what are the most important tables?" → {"tool":"ranking","params":{"kinds":["table"],"metric":"in_degree"}}
User: "what happens if I delete users?" → {"tool":"impact","params":{"symbol_name":"users","change_type":"delete"}}
User: "show me everything about auth" → {"tool":"subgraph","params":{"topic":"auth"}}
User: "how many procedures access users?" → {"tool":"search","params":{"query":"users","kinds":["procedure"]}}

Reply ONLY valid JSON. No explanation, no markdown.`

// ToolSelection is the LLM's routing decision.
type ToolSelection struct {
	Tool   string         `json:"tool"`
	Params map[string]any `json:"params"`
}

// routeIntent uses the LLM to classify a question and select the appropriate tool.
func routeIntent(ctx context.Context, llmClient *llm.Client, question string, sessionRecap string) (*ToolSelection, error) {
	userContent := question
	if sessionRecap != "" {
		userContent = fmt.Sprintf("Prior context: %s\n\nQuestion: %s", sessionRecap, question)
	}

	messages := []llm.Message{
		{Role: "system", Content: routerSystemPrompt},
		{Role: "user", Content: userContent},
	}

	response, err := llmClient.Complete(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("LLM routing: %w", err)
	}

	return parseToolSelection(response)
}

// parseToolSelection extracts the tool selection from the LLM response.
func parseToolSelection(response string) (*ToolSelection, error) {
	response = strings.TrimSpace(response)

	// Strip markdown code fences if present
	if strings.HasPrefix(response, "```") {
		lines := strings.Split(response, "\n")
		var inner []string
		for _, l := range lines {
			if strings.HasPrefix(strings.TrimSpace(l), "```") {
				continue
			}
			inner = append(inner, l)
		}
		response = strings.TrimSpace(strings.Join(inner, "\n"))
	}

	// Try to extract JSON object from the response (LLM may add surrounding text)
	jsonStr := extractJSON(response)
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON found in response: %q", truncate(response, 200))
	}

	var sel ToolSelection
	if err := json.Unmarshal([]byte(jsonStr), &sel); err != nil {
		return nil, fmt.Errorf("parse tool selection: %w (raw: %s)", err, truncate(jsonStr, 200))
	}

	validTools := map[string]bool{
		"search": true, "ranking": true, "overview": true,
		"subgraph": true, "relationships": true, "lineage": true, "impact": true,
	}
	if !validTools[sel.Tool] {
		return nil, fmt.Errorf("unknown tool %q", sel.Tool)
	}

	if sel.Params == nil {
		sel.Params = make(map[string]any)
	}

	return &sel, nil
}

// extractJSON finds the first complete JSON object in a string.
func extractJSON(s string) string {
	start := strings.Index(s, "{")
	if start < 0 {
		return ""
	}

	// Walk forward to find matching closing brace
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(s); i++ {
		ch := s[i]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && inString {
			escaped = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if ch == '{' {
			depth++
		} else if ch == '}' {
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}

	// If we never found a complete object, return empty
	return ""
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// fallbackRoute uses keyword heuristics when the LLM is unavailable.
func fallbackRoute(question string) *ToolSelection {
	q := strings.ToLower(question)

	// Check for specific symbol mentions + action patterns
	rankingPatterns := []string{
		"most used", "most important", "most referenced", "most connected",
		"top ", "busiest", "highest", "largest",
	}
	for _, p := range rankingPatterns {
		if strings.Contains(q, p) {
			return &ToolSelection{Tool: "ranking", Params: map[string]any{"kinds": extractKinds(q)}}
		}
	}

	impactPatterns := []string{
		"what breaks", "what happens if", "impact", "blast radius",
		"affected",
	}
	for _, p := range impactPatterns {
		if strings.Contains(q, p) {
			return &ToolSelection{Tool: "impact", Params: map[string]any{"symbol_name": extractMainSubject(question), "change_type": "modify"}}
		}
	}

	lineagePatterns := []string{
		"data flow", "lineage", "where does", "data come from",
		"upstream", "downstream", "populates",
	}
	for _, p := range lineagePatterns {
		if strings.Contains(q, p) {
			return &ToolSelection{Tool: "lineage", Params: map[string]any{"symbol_name": extractMainSubject(question), "direction": "both"}}
		}
	}

	overviewPatterns := []string{
		"overview", "what is this", "describe the project", "summary",
		"architecture", "how big",
	}
	for _, p := range overviewPatterns {
		if strings.Contains(q, p) {
			return &ToolSelection{Tool: "overview", Params: map[string]any{}}
		}
	}

	relPatterns := []string{
		"foreign key", "relationship", "joins", "references between",
	}
	for _, p := range relPatterns {
		if strings.Contains(q, p) {
			return &ToolSelection{Tool: "relationships", Params: map[string]any{"topic": extractMainSubject(question)}}
		}
	}

	// "accessing", "uses", "calls", "references" + a symbol → search with kinds
	accessPatterns := []string{
		"accessing", "access", "uses", "calls", "references", "depends",
	}
	for _, p := range accessPatterns {
		if strings.Contains(q, p) {
			kinds := extractKinds(q)
			return &ToolSelection{Tool: "search", Params: map[string]any{"query": extractMainSubject(question), "kinds": kinds}}
		}
	}

	subgraphPatterns := []string{
		"everything about", "all related", "module", "pipeline",
	}
	for _, p := range subgraphPatterns {
		if strings.Contains(q, p) {
			return &ToolSelection{Tool: "subgraph", Params: map[string]any{"topic": extractMainSubject(question)}}
		}
	}

	return &ToolSelection{Tool: "search", Params: map[string]any{"query": extractMainSubject(question)}}
}

// extractMainSubject removes stop words from a question to find the core subject.
func extractMainSubject(question string) string {
	stopWords := map[string]bool{
		"what": true, "where": true, "how": true, "does": true, "is": true,
		"the": true, "a": true, "an": true, "are": true, "can": true,
		"do": true, "if": true, "i": true, "to": true, "of": true,
		"in": true, "for": true, "it": true, "this": true, "that": true,
		"about": true, "show": true, "me": true, "find": true, "get": true,
		"tell": true, "breaks": true, "happens": true, "everything": true,
		"most": true, "used": true, "important": true, "top": true,
		"then": true, "so": true, "many": true, "much": true,
	}

	words := strings.Fields(strings.ToLower(question))
	var terms []string
	for _, w := range words {
		w = strings.Trim(w, "?.,!\"'")
		if !stopWords[w] && len(w) > 1 {
			terms = append(terms, w)
		}
	}

	if len(terms) == 0 {
		return question
	}
	return strings.Join(terms, " ")
}

// extractKinds pulls symbol kinds from a question.
func extractKinds(q string) []string {
	kindMap := map[string]string{
		"table": "table", "tables": "table",
		"procedure": "procedure", "procedures": "procedure", "proc": "procedure", "procs": "procedure", "stored procedure": "procedure",
		"function": "function", "functions": "function",
		"class": "class", "classes": "class",
		"method": "method", "methods": "method",
		"column": "column", "columns": "column",
		"view": "view", "views": "view",
	}
	seen := make(map[string]bool)
	var kinds []string
	for word, kind := range kindMap {
		if strings.Contains(q, word) && !seen[kind] {
			seen[kind] = true
			kinds = append(kinds, kind)
		}
	}
	return kinds
}
