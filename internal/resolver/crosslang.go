package resolver

import (
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"github.com/codegraph-labs/codegraph/internal/parser"
)

// BridgeRule defines how to resolve references between different languages.
type BridgeRule struct {
	SourceLanguage string // e.g., "delphi", "asp", "java"
	TargetLanguage string // e.g., "tsql", "pgsql"
	MatchStrategy  string // exact, case_insensitive, schema_qualified, strip_prefix
}

// CrossLangResolver resolves references across language boundaries.
type CrossLangResolver struct {
	rules  []BridgeRule
	logger *slog.Logger
}

// NewCrossLangResolver creates a new cross-language resolver.
func NewCrossLangResolver(logger *slog.Logger) *CrossLangResolver {
	c := &CrossLangResolver{logger: logger}
	c.RegisterDefaultRules()
	return c
}

// RegisterDefaultRules sets up the default cross-language bridge rules.
func (c *CrossLangResolver) RegisterDefaultRules() {
	c.rules = []BridgeRule{
		// App → SQL: Delphi/ASP/Java/C# referencing SQL objects
		{SourceLanguage: "delphi", TargetLanguage: "tsql", MatchStrategy: "schema_qualified"},
		{SourceLanguage: "asp", TargetLanguage: "tsql", MatchStrategy: "case_insensitive"},
		{SourceLanguage: "java", TargetLanguage: "pgsql", MatchStrategy: "case_insensitive"},
		{SourceLanguage: "java", TargetLanguage: "tsql", MatchStrategy: "case_insensitive"},
		{SourceLanguage: "csharp", TargetLanguage: "tsql", MatchStrategy: "schema_qualified"},
		{SourceLanguage: "csharp", TargetLanguage: "tsql", MatchStrategy: "case_insensitive"},
		{SourceLanguage: "javascript", TargetLanguage: "tsql", MatchStrategy: "case_insensitive"},
		{SourceLanguage: "typescript", TargetLanguage: "tsql", MatchStrategy: "case_insensitive"},

		// Delphi T-prefix: strip T from class names when matching SQL objects
		{SourceLanguage: "delphi", TargetLanguage: "tsql", MatchStrategy: "strip_prefix"},
	}
}

// Resolve attempts to resolve a reference using cross-language bridge rules.
func (c *CrossLangResolver) Resolve(ref parser.RawReference, sourceLang string, table *SymbolTable) (uuid.UUID, bool) {
	targetName := ref.ToName
	targetQualified := ref.ToQualified
	if targetQualified == "" {
		targetQualified = targetName
	}

	for _, rule := range c.rules {
		if !matchesLanguage(sourceLang, rule.SourceLanguage) {
			continue
		}

		switch rule.MatchStrategy {
		case "exact":
			if id, ok := table.ByFQN[targetQualified]; ok {
				return id, true
			}

		case "case_insensitive":
			lower := strings.ToLower(targetName)
			for fqn, id := range table.ByFQN {
				if strings.ToLower(shortNameOf(fqn)) == lower {
					// Verify target language matches if available
					if lang, hasLang := table.ByLang[fqn]; hasLang && matchesLanguage(lang, rule.TargetLanguage) {
						return id, true
					} else if !hasLang {
						// If we can't verify language, still return the match
						return id, true
					}
				}
			}

		case "schema_qualified":
			// Try with dbo. prefix (T-SQL default schema)
			candidates := []string{
				targetQualified,
				"dbo." + targetName,
				targetName,
			}
			for _, candidate := range candidates {
				lower := strings.ToLower(candidate)
				for fqn, id := range table.ByFQN {
					if strings.ToLower(fqn) == lower {
						return id, true
					}
				}
			}

		case "strip_prefix":
			// Strip common prefixes (e.g., Delphi's T prefix for class names)
			stripped := targetName
			if strings.HasPrefix(stripped, "T") && len(stripped) > 1 {
				stripped = stripped[1:]
			}
			lower := strings.ToLower(stripped)
			for fqn, id := range table.ByFQN {
				if strings.ToLower(shortNameOf(fqn)) == lower {
					return id, true
				}
			}
		}
	}

	return uuid.Nil, false
}

func matchesLanguage(actual, pattern string) bool {
	return strings.EqualFold(actual, pattern)
}
