package ingestion

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/google/uuid"

	"github.com/codegraph-labs/codegraph/internal/parser"
	"github.com/codegraph-labs/codegraph/internal/store"
	"github.com/codegraph-labs/codegraph/internal/store/postgres"
)

// PersistResults writes parsed file results to PostgreSQL.
// Returns counts of files, symbols, and edges persisted.
func PersistResults(ctx context.Context, s *store.Store, results []parser.FileResult) (files, symbols, edges int, err error) {
	for _, fr := range results {
		// Upsert file
		hash := fmt.Sprintf("%x", sha256.Sum256([]byte(fr.Path)))
		if fr.Hash != "" {
			hash = fr.Hash
		}

		dbFile, err := s.UpsertFile(ctx, postgres.UpsertFileParams{
			ProjectID: fr.ProjectID,
			SourceID:  fr.SourceID,
			Path:      fr.Path,
			Language:  fr.Language,
			SizeBytes: fr.SizeBytes,
			Hash:      hash,
		})
		if err != nil {
			return files, symbols, edges, fmt.Errorf("upsert file %s: %w", fr.Path, err)
		}
		files++

		// Delete existing symbols for this file (re-index)
		_ = s.DeleteSymbolsByFile(ctx, dbFile.ID)

		// Insert symbols, tracking qualified_name -> ID for edge resolution
		symbolIDs := make(map[string]uuid.UUID)

		for _, sym := range fr.Symbols {
			created, err := createSymbol(ctx, s, fr.ProjectID, dbFile.ID, sym)
			if err != nil {
				return files, symbols, edges, fmt.Errorf("create symbol %s: %w", sym.QualifiedName, err)
			}
			symbolIDs[sym.QualifiedName] = created.ID
			symbols++

			// Also insert child symbols (e.g., columns)
			for _, child := range sym.Children {
				childCreated, err := createSymbol(ctx, s, fr.ProjectID, dbFile.ID, child)
				if err != nil {
					return files, symbols, edges, fmt.Errorf("create child symbol %s: %w", child.QualifiedName, err)
				}
				symbolIDs[child.QualifiedName] = childCreated.ID
				symbols++
			}
		}

		// Insert edges (best-effort: skip if source or target symbol not found)
		for _, ref := range fr.References {
			sourceID, ok := symbolIDs[ref.FromSymbol]
			if !ok {
				continue
			}
			targetID, ok := symbolIDs[ref.ToQualified]
			if !ok {
				// Try unqualified name
				targetID, ok = symbolIDs[ref.ToName]
				if !ok {
					continue
				}
			}

			_, err := s.CreateSymbolEdge(ctx, postgres.CreateSymbolEdgeParams{
				ProjectID: fr.ProjectID,
				SourceID:  sourceID,
				TargetID:  targetID,
				EdgeType:  ref.ReferenceType,
			})
			if err != nil {
				// ON CONFLICT DO NOTHING means this is ok
				continue
			}
			edges++
		}
	}

	return files, symbols, edges, nil
}

func createSymbol(ctx context.Context, s *store.Store, projectID, fileID uuid.UUID, sym parser.Symbol) (postgres.Symbol, error) {
	var startCol, endCol *int32
	if sym.StartCol > 0 {
		v := int32(sym.StartCol)
		startCol = &v
	}
	if sym.EndCol > 0 {
		v := int32(sym.EndCol)
		endCol = &v
	}
	var sig, doc *string
	if sym.Signature != "" {
		sig = &sym.Signature
	}
	if sym.DocComment != "" {
		doc = &sym.DocComment
	}

	return s.CreateSymbol(ctx, postgres.CreateSymbolParams{
		ProjectID:     projectID,
		FileID:        fileID,
		Name:          sym.Name,
		QualifiedName: sym.QualifiedName,
		Kind:          sym.Kind,
		Language:      sym.Language,
		StartLine:     int32(sym.StartLine),
		EndLine:       int32(sym.EndLine),
		StartCol:      startCol,
		EndCol:        endCol,
		Signature:     sig,
		DocComment:    doc,
	})
}
