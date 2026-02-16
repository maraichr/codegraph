package resolver

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"github.com/maraichr/lattice/internal/parser"
	"github.com/maraichr/lattice/internal/store"
	"github.com/maraichr/lattice/internal/store/postgres"
)

// Engine performs cross-file symbol resolution within a project.
type Engine struct {
	store     *store.Store
	crossLang *CrossLangResolver
	logger    *slog.Logger
}

func NewEngine(s *store.Store, logger *slog.Logger) *Engine {
	return &Engine{
		store:     s,
		crossLang: NewCrossLangResolver(logger),
		logger:    logger,
	}
}

// SymbolTable indexes all symbols in a project for fast lookup.
type SymbolTable struct {
	ByFQN       map[string]uuid.UUID   // qualified_name → symbol ID
	ByShortName map[string][]uuid.UUID // short name → candidate IDs
	ByFile      map[uuid.UUID][]uuid.UUID // file ID → symbol IDs
	FileByPath  map[string]uuid.UUID   // file path → file ID
	ByLang      map[string]string      // qualified_name → language
}

func newSymbolTable() *SymbolTable {
	return &SymbolTable{
		ByFQN:       make(map[string]uuid.UUID),
		ByShortName: make(map[string][]uuid.UUID),
		ByFile:      make(map[uuid.UUID][]uuid.UUID),
		FileByPath:  make(map[string]uuid.UUID),
		ByLang:      make(map[string]string),
	}
}

// Resolve performs cross-file symbol resolution for a project.
// It looks at unresolved references from the parse results and tries to
// match them against the project-wide symbol table.
// Returns the number of new edges created.
func (e *Engine) Resolve(ctx context.Context, projectID uuid.UUID, parseResults []parser.FileResult) (int, error) {
	// Build the project-wide symbol table from PG
	symbols, err := e.store.ListSymbolsByProject(ctx, projectID)
	if err != nil {
		return 0, fmt.Errorf("load symbols: %w", err)
	}

	files, err := e.store.ListFilesByProject(ctx, projectID)
	if err != nil {
		return 0, fmt.Errorf("load files: %w", err)
	}

	table := newSymbolTable()

	for _, f := range files {
		table.FileByPath[f.Path] = f.ID
	}

	for _, sym := range symbols {
		table.ByFQN[sym.QualifiedName] = sym.ID
		shortName := shortNameOf(sym.QualifiedName)
		table.ByShortName[shortName] = append(table.ByShortName[shortName], sym.ID)
		table.ByFile[sym.FileID] = append(table.ByFile[sym.FileID], sym.ID)
		table.ByLang[sym.QualifiedName] = sym.Language
	}

	// Build file-local symbol sets for scope resolution
	fileSymbols := make(map[uuid.UUID]map[string]uuid.UUID) // fileID → qname → symID
	for _, sym := range symbols {
		if fileSymbols[sym.FileID] == nil {
			fileSymbols[sym.FileID] = make(map[string]uuid.UUID)
		}
		fileSymbols[sym.FileID][sym.QualifiedName] = sym.ID
		fileSymbols[sym.FileID][sym.Name] = sym.ID
	}

	created := 0

	// For each file's unresolved references, attempt cross-file resolution
	for _, fr := range parseResults {
		fileID, ok := table.FileByPath[fr.Path]
		if !ok {
			continue
		}

		localScope := fileSymbols[fileID]

		for _, ref := range fr.References {
			sourceID, ok := localScope[ref.FromSymbol]
			if !ok {
				// Source symbol not in this file's scope — try project-wide
				sourceID, ok = table.ByFQN[ref.FromSymbol]
			}
			// When FromSymbol is empty but ToName is set (e.g. C# [Table("X")] fallback), infer source from this file's symbols
			if !ok && ref.FromSymbol == "" && ref.ToName != "" && ref.ReferenceType == "uses_table" {
				sourceID = inferSourceFromFileSymbols(fileID, table)
			}
			if sourceID == uuid.Nil {
				continue
			}

			// Try to resolve the target
			result := resolveTarget(ref, localScope, table, e.crossLang, fr.Language)
			if !result.Resolved {
				continue
			}

			// Skip self-references
			if sourceID == result.TargetID {
				continue
			}

			// Determine confidence: use ref's confidence if set, otherwise from resolution
			confidence := result.Confidence
			if ref.Confidence > 0 && confidence > 0 {
				// Multiply parser confidence with resolution confidence
				confidence = ref.Confidence * confidence
			} else if ref.Confidence > 0 {
				confidence = ref.Confidence
			}

			// Use CreateSymbolEdgeWithMetadata for cross-language edges with confidence
			if result.CrossLang {
				meta := map[string]interface{}{
					"confidence":     confidence,
					"match_strategy": result.Strategy,
					"bridge":         result.Bridge,
				}
				metaJSON, _ := json.Marshal(meta)
				_, err := e.store.CreateSymbolEdgeWithMetadata(ctx, postgres.CreateSymbolEdgeWithMetadataParams{
					ProjectID: projectID,
					SourceID:  sourceID,
					TargetID:  result.TargetID,
					EdgeType:  ref.ReferenceType,
					Metadata:  metaJSON,
				})
				if err != nil {
					continue
				}
			} else {
				_, err := e.store.CreateSymbolEdge(ctx, postgres.CreateSymbolEdgeParams{
					ProjectID: projectID,
					SourceID:  sourceID,
					TargetID:  result.TargetID,
					EdgeType:  ref.ReferenceType,
				})
				if err != nil {
					continue
				}
			}
			created++
		}
	}

	e.logger.Info("cross-file resolution complete",
		slog.Int("edges_created", created),
		slog.Int("symbols_indexed", len(symbols)))

	return created, nil
}

// resolveResult holds the outcome of target resolution.
type resolveResult struct {
	TargetID   uuid.UUID
	Confidence float64 // 0 = not set (same-language, treated as 1.0)
	Strategy   string
	Bridge     string
	CrossLang  bool
	Resolved   bool
}

// resolveTarget attempts to find the target symbol for a reference.
// Resolution order: qualified name → file-local scope → project-wide short name → case-insensitive → cross-language.
func resolveTarget(ref parser.RawReference, localScope map[string]uuid.UUID, table *SymbolTable, crossLang *CrossLangResolver, sourceLang string) resolveResult {
	// 1. Try fully qualified name
	if ref.ToQualified != "" {
		if id, ok := table.ByFQN[ref.ToQualified]; ok {
			return resolveResult{TargetID: id, Confidence: 1.0, Resolved: true}
		}
	}

	// 2. Try the target name in local scope (already resolved in parse stage, but try anyway)
	if id, ok := localScope[ref.ToName]; ok {
		return resolveResult{TargetID: id, Confidence: 1.0, Resolved: true}
	}
	if ref.ToQualified != "" {
		if id, ok := localScope[ref.ToQualified]; ok {
			return resolveResult{TargetID: id, Confidence: 1.0, Resolved: true}
		}
	}

	// 3. Try project-wide by short name (if unambiguous)
	candidates := table.ByShortName[ref.ToName]
	if len(candidates) == 1 {
		return resolveResult{TargetID: candidates[0], Confidence: 1.0, Resolved: true}
	}

	// 4. Try case-insensitive FQN match (SQL is often case-insensitive)
	lowerTarget := strings.ToLower(ref.ToName)
	for fqn, id := range table.ByFQN {
		if strings.ToLower(shortNameOf(fqn)) == lowerTarget {
			return resolveResult{TargetID: id, Confidence: 1.0, Resolved: true}
		}
	}

	// 5. Try cross-language resolution
	if crossLang != nil && sourceLang != "" {
		if match, ok := crossLang.Resolve(ref, sourceLang, table); ok {
			return resolveResult{
				TargetID:   match.TargetID,
				Confidence: match.Confidence,
				Strategy:   match.Strategy,
				Bridge:     match.Bridge,
				CrossLang:  true,
				Resolved:   true,
			}
		}
	}

	return resolveResult{}
}

// shortNameOf extracts the short name from a qualified name.
// e.g., "dbo.Customers" → "Customers", "schema.proc" → "proc"
func shortNameOf(qualifiedName string) string {
	parts := strings.Split(qualifiedName, ".")
	return parts[len(parts)-1]
}

// inferSourceFromFileSymbols returns one symbol ID from the file when refs have no FromSymbol (e.g. C# uses_table).
// Used so that [Table("X")] or inline SQL refs can still create an edge from the enclosing type.
func inferSourceFromFileSymbols(fileID uuid.UUID, table *SymbolTable) uuid.UUID {
	ids := table.ByFile[fileID]
	if len(ids) == 0 {
		return uuid.Nil
	}
	return ids[0]
}
