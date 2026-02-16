package resolver

import (
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"github.com/maraichr/lattice/internal/parser"
)

// BridgeRule defines how to resolve references between different languages.
type BridgeRule struct {
	SourceLanguage string // e.g., "delphi", "asp", "java"
	TargetLanguage string // e.g., "tsql", "pgsql"
	MatchStrategy  string // exact, case_insensitive, schema_qualified, strip_prefix
}

// BridgeMatch represents a successful cross-language resolution with confidence.
type BridgeMatch struct {
	TargetID   uuid.UUID
	Confidence float64 // exact=1.0, schema_qualified=0.95, case_insensitive=0.85, strip_prefix=0.75, orm_convention=0.7
	Strategy   string
	Bridge     string // e.g., "csharp→tsql"
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

		// JS/TS → PostgreSQL (common with Node.js stacks)
		{SourceLanguage: "javascript", TargetLanguage: "pgsql", MatchStrategy: "case_insensitive"},
		{SourceLanguage: "typescript", TargetLanguage: "pgsql", MatchStrategy: "case_insensitive"},

		// C# → PostgreSQL
		{SourceLanguage: "csharp", TargetLanguage: "pgsql", MatchStrategy: "schema_qualified"},

		// ORM convention matching (pluralize/singularize)
		{SourceLanguage: "csharp", TargetLanguage: "tsql", MatchStrategy: "orm_convention"},
		{SourceLanguage: "java", TargetLanguage: "pgsql", MatchStrategy: "orm_convention"},
		{SourceLanguage: "java", TargetLanguage: "tsql", MatchStrategy: "orm_convention"},

		// Delphi T-prefix: strip T from class names when matching SQL objects
		{SourceLanguage: "delphi", TargetLanguage: "tsql", MatchStrategy: "strip_prefix"},
	}
}

// Resolve attempts to resolve a reference using cross-language bridge rules.
// Returns a BridgeMatch with confidence and strategy information.
func (c *CrossLangResolver) Resolve(ref parser.RawReference, sourceLang string, table *SymbolTable) (BridgeMatch, bool) {
	targetName := ref.ToName
	targetQualified := ref.ToQualified
	if targetQualified == "" {
		targetQualified = targetName
	}

	for _, rule := range c.rules {
		if !matchesLanguage(sourceLang, rule.SourceLanguage) {
			continue
		}

		bridge := rule.SourceLanguage + "→" + rule.TargetLanguage

		switch rule.MatchStrategy {
		case "exact":
			if id, ok := table.ByFQN[targetQualified]; ok {
				return BridgeMatch{TargetID: id, Confidence: 1.0, Strategy: "exact", Bridge: bridge}, true
			}

		case "case_insensitive":
			lower := strings.ToLower(targetName)
			for fqn, id := range table.ByFQN {
				if strings.ToLower(shortNameOf(fqn)) == lower {
					// Verify target language matches if available
					if lang, hasLang := table.ByLang[fqn]; hasLang && matchesLanguage(lang, rule.TargetLanguage) {
						return BridgeMatch{TargetID: id, Confidence: 0.85, Strategy: "case_insensitive", Bridge: bridge}, true
					} else if !hasLang {
						return BridgeMatch{TargetID: id, Confidence: 0.85, Strategy: "case_insensitive", Bridge: bridge}, true
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
						return BridgeMatch{TargetID: id, Confidence: 0.95, Strategy: "schema_qualified", Bridge: bridge}, true
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
					return BridgeMatch{TargetID: id, Confidence: 0.75, Strategy: "strip_prefix", Bridge: bridge}, true
				}
			}

		case "orm_convention":
			// ORM naming: try pluralize/singularize
			variants := ormNameVariants(targetName)
			for _, variant := range variants {
				lower := strings.ToLower(variant)
				for fqn, id := range table.ByFQN {
					if strings.ToLower(shortNameOf(fqn)) == lower {
						if lang, hasLang := table.ByLang[fqn]; hasLang && matchesLanguage(lang, rule.TargetLanguage) {
							return BridgeMatch{TargetID: id, Confidence: 0.7, Strategy: "orm_convention", Bridge: bridge}, true
						} else if !hasLang {
							return BridgeMatch{TargetID: id, Confidence: 0.7, Strategy: "orm_convention", Bridge: bridge}, true
						}
					}
				}
			}
		}
	}

	return BridgeMatch{}, false
}

// ormNameVariants returns naming convention variants for ORM resolution.
func ormNameVariants(name string) []string {
	variants := []string{name}

	// Pluralize
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, "y") && !strings.HasSuffix(lower, "ey") && !strings.HasSuffix(lower, "ay") && !strings.HasSuffix(lower, "oy") {
		variants = append(variants, name[:len(name)-1]+"ies")
	} else if strings.HasSuffix(lower, "s") || strings.HasSuffix(lower, "x") || strings.HasSuffix(lower, "ch") || strings.HasSuffix(lower, "sh") {
		variants = append(variants, name+"es")
	} else {
		variants = append(variants, name+"s")
	}

	// Singularize
	if strings.HasSuffix(lower, "ies") {
		variants = append(variants, name[:len(name)-3]+"y")
	} else if strings.HasSuffix(lower, "es") {
		variants = append(variants, name[:len(name)-2])
	} else if strings.HasSuffix(lower, "s") && !strings.HasSuffix(lower, "ss") {
		variants = append(variants, name[:len(name)-1])
	}

	return variants
}

func matchesLanguage(actual, pattern string) bool {
	return strings.EqualFold(actual, pattern)
}
