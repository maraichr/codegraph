package mcp

import (
	"fmt"
	"strings"

	"github.com/codegraph-labs/codegraph/internal/mcp/session"
	"github.com/codegraph-labs/codegraph/internal/store/postgres"
)

const defaultMaxTokens = 4000

// Verbosity controls how much detail is included in symbol cards.
type Verbosity string

const (
	VerbositySummary  Verbosity = "summary"
	VerbosityStandard Verbosity = "standard"
	VerbosityFull     Verbosity = "full"
)

// ParseVerbosity returns a Verbosity from a string, defaulting to standard.
func ParseVerbosity(s string) Verbosity {
	switch strings.ToLower(s) {
	case "summary":
		return VerbositySummary
	case "full":
		return VerbosityFull
	default:
		return VerbosityStandard
	}
}

// ResponseBuilder constructs token-budgeted Markdown responses for MCP tools.
type ResponseBuilder struct {
	buf           strings.Builder
	tokenEstimate int
	maxTokens     int
	truncated     bool
	itemCount     int
}

// NewResponseBuilder creates a builder with the given token budget.
// If maxTokens <= 0, defaultMaxTokens is used.
func NewResponseBuilder(maxTokens int) *ResponseBuilder {
	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}
	return &ResponseBuilder{maxTokens: maxTokens}
}

// AddHeader writes a header line to the response.
func (rb *ResponseBuilder) AddHeader(text string) {
	line := text + "\n\n"
	rb.buf.WriteString(line)
	rb.tokenEstimate += len(line) / 4
}

// AddLine writes a single line to the response, returning false if budget exceeded.
func (rb *ResponseBuilder) AddLine(text string) bool {
	line := text + "\n"
	cost := len(line) / 4
	if rb.tokenEstimate+cost > rb.maxTokens {
		rb.truncated = true
		return false
	}
	rb.buf.WriteString(line)
	rb.tokenEstimate += cost
	return true
}

// AddSymbolCard renders a symbol at the requested verbosity.
// Returns false if the card would exceed the token budget.
func (rb *ResponseBuilder) AddSymbolCard(sym postgres.Symbol, verbosity Verbosity, sess *session.Session) bool {
	card := formatSymbolCard(sym, verbosity, sess)
	cost := len(card) / 4
	if rb.tokenEstimate+cost > rb.maxTokens {
		rb.truncated = true
		return false
	}
	rb.buf.WriteString(card)
	rb.tokenEstimate += cost
	rb.itemCount++
	return true
}

// AddSymbolStub renders a one-line stub for an already-seen symbol.
func (rb *ResponseBuilder) AddSymbolStub(sym postgres.Symbol) bool {
	stub := fmt.Sprintf("- ~%s~ (%s) — already examined | ID: `%s`\n",
		sym.Name, sym.Kind, sym.ID)
	cost := len(stub) / 4
	if rb.tokenEstimate+cost > rb.maxTokens {
		rb.truncated = true
		return false
	}
	rb.buf.WriteString(stub)
	rb.tokenEstimate += cost
	rb.itemCount++
	return true
}

// AddSection writes a section with a heading.
func (rb *ResponseBuilder) AddSection(heading string, content string) bool {
	section := fmt.Sprintf("### %s\n%s\n\n", heading, content)
	cost := len(section) / 4
	if rb.tokenEstimate+cost > rb.maxTokens {
		rb.truncated = true
		return false
	}
	rb.buf.WriteString(section)
	rb.tokenEstimate += cost
	return true
}

// AddRawText writes raw text, respecting the budget.
func (rb *ResponseBuilder) AddRawText(text string) bool {
	cost := len(text) / 4
	if rb.tokenEstimate+cost > rb.maxTokens {
		rb.truncated = true
		return false
	}
	rb.buf.WriteString(text)
	rb.tokenEstimate += cost
	return true
}

// Finalize appends truncation notice and returns the final response text.
func (rb *ResponseBuilder) Finalize(totalCount, returnedCount int) string {
	if rb.truncated || returnedCount < totalCount {
		rb.buf.WriteString(fmt.Sprintf(
			"\n---\n*Showing %d of %d results (truncated to ~%d tokens). Use `offset` to paginate or increase `max_response_tokens`.*\n",
			returnedCount, totalCount, rb.maxTokens))
	}
	return rb.buf.String()
}

// FinalizeWithHints appends navigation hints and truncation notice.
func (rb *ResponseBuilder) FinalizeWithHints(totalCount, returnedCount int, hints *NavigationHints) string {
	if rb.truncated || returnedCount < totalCount {
		rb.buf.WriteString(fmt.Sprintf(
			"\n---\n*Showing %d of %d results (~%d tokens).*\n",
			returnedCount, totalCount, rb.tokenEstimate))
	}

	if hints != nil && len(hints.Steps) > 0 {
		rb.buf.WriteString("\n---\n**Next steps:**\n")
		for _, step := range hints.Steps {
			rb.buf.WriteString(fmt.Sprintf("- %s → `%s`", step.Description, step.Tool))
			if step.EstimatedTokens > 0 {
				rb.buf.WriteString(fmt.Sprintf(" (~%d tokens)", step.EstimatedTokens))
			}
			rb.buf.WriteString("\n")
		}
	}

	return rb.buf.String()
}

// TokenEstimate returns the current estimated token count.
func (rb *ResponseBuilder) TokenEstimate() int {
	return rb.tokenEstimate
}

// IsTruncated returns whether the response was truncated.
func (rb *ResponseBuilder) IsTruncated() bool {
	return rb.truncated
}

// ItemCount returns the number of items added.
func (rb *ResponseBuilder) ItemCount() int {
	return rb.itemCount
}

// DryRunResult represents the result of a dry run (cost preview).
type DryRunResult struct {
	SymbolCount     int `json:"symbol_count"`
	EdgeCount       int `json:"edge_count"`
	EstimatedTokens int `json:"estimated_tokens"`
	DepthReached    int `json:"depth_reached,omitempty"`
}

// FormatDryRun formats a dry run result as a Markdown response.
func FormatDryRun(result DryRunResult) string {
	var b strings.Builder
	b.WriteString("**Dry Run Preview**\n\n")
	b.WriteString(fmt.Sprintf("- Symbols: %d\n", result.SymbolCount))
	b.WriteString(fmt.Sprintf("- Edges: %d\n", result.EdgeCount))
	b.WriteString(fmt.Sprintf("- Estimated tokens: ~%d\n", result.EstimatedTokens))
	if result.DepthReached > 0 {
		b.WriteString(fmt.Sprintf("- Depth reached: %d\n", result.DepthReached))
	}
	return b.String()
}

// formatSymbolCard renders a symbol as a Markdown card at the given verbosity.
func formatSymbolCard(sym postgres.Symbol, verbosity Verbosity, sess *session.Session) string {
	var b strings.Builder

	// Check if already seen
	seen := ""
	if sess != nil && sess.IsSeen(sym.ID) {
		seen = " *(seen)*"
	}

	switch verbosity {
	case VerbositySummary:
		b.WriteString(fmt.Sprintf("**%s** (%s)%s\n", sym.Name, sym.Kind, seen))
		b.WriteString(fmt.Sprintf("  FQN: `%s`\n", sym.QualifiedName))
		b.WriteString(fmt.Sprintf("  ID: `%s`\n\n", sym.ID))

	case VerbosityFull:
		b.WriteString(fmt.Sprintf("**%s** (%s)%s\n", sym.Name, sym.Kind, seen))
		b.WriteString(fmt.Sprintf("  FQN: `%s`\n", sym.QualifiedName))
		b.WriteString(fmt.Sprintf("  Language: %s\n", sym.Language))
		b.WriteString(fmt.Sprintf("  Location: L%d–L%d\n", sym.StartLine, sym.EndLine))
		if sym.Signature != nil {
			b.WriteString(fmt.Sprintf("  Signature: `%s`\n", *sym.Signature))
		}
		if sym.DocComment != nil {
			b.WriteString(fmt.Sprintf("  Doc: %s\n", *sym.DocComment))
		}
		b.WriteString(fmt.Sprintf("  ID: `%s`\n\n", sym.ID))

	default: // standard
		b.WriteString(fmt.Sprintf("**%s** (%s)%s\n", sym.Name, sym.Kind, seen))
		b.WriteString(fmt.Sprintf("  FQN: `%s`\n", sym.QualifiedName))
		b.WriteString(fmt.Sprintf("  Language: %s | L%d–L%d\n", sym.Language, sym.StartLine, sym.EndLine))
		if sym.Signature != nil {
			b.WriteString(fmt.Sprintf("  Signature: `%s`\n", *sym.Signature))
		}
		b.WriteString(fmt.Sprintf("  ID: `%s`\n\n", sym.ID))
	}

	return b.String()
}
