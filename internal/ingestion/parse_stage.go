package ingestion

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/codegraph-labs/codegraph/internal/parser"
	"github.com/codegraph-labs/codegraph/internal/store"
	"github.com/codegraph-labs/codegraph/internal/store/postgres"
)

// ParseStage walks the work directory, parses SQL files, and persists results.
type ParseStage struct {
	registry *parser.Registry
	store    *store.Store
}

func NewParseStage(registry *parser.Registry, store *store.Store) *ParseStage {
	return &ParseStage{registry: registry, store: store}
}

func (s *ParseStage) Name() string { return "parse" }

func (s *ParseStage) Execute(ctx context.Context, rc *IndexRunContext) error {
	if rc.WorkDir == "" {
		return nil // no files to parse (e.g., no clone stage ran)
	}

	// Handle incremental: delete symbols for removed files
	if rc.Incremental && len(rc.DeletedFiles) > 0 {
		for _, delPath := range rc.DeletedFiles {
			file, err := s.store.GetFileByPath(ctx, postgres.GetFileByPathParams{
				ProjectID: rc.ProjectID,
				SourceID:  rc.SourceID,
				Path:      delPath,
			})
			if err != nil {
				continue // file may not exist
			}
			_ = s.store.DeleteSymbolsByFileID(ctx, file.ID)
		}
	}

	var results []parser.FileResult

	if rc.Incremental && len(rc.ChangedFiles) > 0 {
		// Incremental: only parse changed files
		for _, relPath := range rc.ChangedFiles {
			absPath := filepath.Join(rc.WorkDir, relPath)
			info, err := os.Stat(absPath)
			if err != nil {
				continue // file might not exist
			}
			fr := s.parseFile(rc, absPath, relPath, info)
			if fr != nil {
				results = append(results, *fr)
			}
		}
	} else {
		// Full scan
		err := filepath.Walk(rc.WorkDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}

			relPath, _ := filepath.Rel(rc.WorkDir, path)
			fr := s.parseFile(rc, path, relPath, info)
			if fr != nil {
				results = append(results, *fr)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("walk work dir: %w", err)
		}
	}

	files, symbols, edges, err := PersistResults(ctx, s.store, results)
	if err != nil {
		return fmt.Errorf("persist results: %w", err)
	}

	rc.FilesProcessed = files
	rc.SymbolsFound = symbols
	rc.EdgesFound = edges
	rc.ParseResults = results

	return nil
}

func (s *ParseStage) parseFile(rc *IndexRunContext, absPath, relPath string, info os.FileInfo) *parser.FileResult {
	p := s.registry.ForFile(absPath)
	if p == nil {
		return nil
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil
	}

	// Detect SQL dialect for .sql files
	language := "sql"
	if filepath.Ext(absPath) == ".sql" {
		language = parser.DetectDialect(content)
	}

	input := parser.FileInput{
		Path:     relPath,
		Content:  content,
		Language: language,
	}

	result, err := p.Parse(input)
	if err != nil {
		return nil
	}

	hash := fmt.Sprintf("%x", sha256.Sum256(content))

	return &parser.FileResult{
		ProjectID:        rc.ProjectID,
		SourceID:         rc.SourceID,
		Path:             relPath,
		Language:         language,
		SizeBytes:        info.Size(),
		Hash:             hash,
		Symbols:          result.Symbols,
		References:       result.References,
		ColumnReferences: result.ColumnReferences,
	}
}
