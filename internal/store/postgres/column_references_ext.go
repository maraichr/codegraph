package postgres

// column_references_ext.go provides hand-written DB access for the column_references
// table introduced in migration 007, supplementing SQLC-generated code.

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ColumnReference is the DB model for the column_references table.
type ColumnReference struct {
	ID             uuid.UUID  `json:"id"`
	ProjectID      uuid.UUID  `json:"project_id"`
	IndexRunID     uuid.UUID  `json:"index_run_id"`
	SourceColumn   string     `json:"source_column"`
	TargetColumn   string     `json:"target_column"`
	DerivationType string     `json:"derivation_type"`
	Expression     *string    `json:"expression"`
	Context        *string    `json:"context"`
	Line           *int32     `json:"line"`
	CreatedAt      time.Time  `json:"created_at"`
}

// InsertColumnReferenceParams holds the fields for inserting a column reference.
type InsertColumnReferenceParams struct {
	ProjectID      uuid.UUID
	IndexRunID     uuid.UUID
	SourceColumn   string
	TargetColumn   string
	DerivationType string
	Expression     *string
	Context        *string
	Line           *int32
}

// InsertColumnReference inserts a single column reference into the DB.
func (q *Queries) InsertColumnReference(ctx context.Context, arg InsertColumnReferenceParams) error {
	_, err := q.db.Exec(ctx,
		`INSERT INTO column_references
		   (project_id, index_run_id, source_column, target_column, derivation_type, expression, context, line)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		arg.ProjectID, arg.IndexRunID, arg.SourceColumn, arg.TargetColumn,
		arg.DerivationType, arg.Expression, arg.Context, arg.Line)
	return err
}

// ListColumnReferencesByIndexRun returns all column references for an index run.
func (q *Queries) ListColumnReferencesByIndexRun(ctx context.Context, indexRunID uuid.UUID) ([]ColumnReference, error) {
	rows, err := q.db.Query(ctx,
		`SELECT id, project_id, index_run_id, source_column, target_column,
		        derivation_type, expression, context, line, created_at
		 FROM column_references
		 WHERE index_run_id = $1`,
		indexRunID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ColumnReference
	for rows.Next() {
		var i ColumnReference
		if err := rows.Scan(
			&i.ID, &i.ProjectID, &i.IndexRunID, &i.SourceColumn, &i.TargetColumn,
			&i.DerivationType, &i.Expression, &i.Context, &i.Line, &i.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

// DeleteColumnReferencesByIndexRun removes all column references for an index run.
// Called after the lineage stage completes to free up storage.
func (q *Queries) DeleteColumnReferencesByIndexRun(ctx context.Context, indexRunID uuid.UUID) error {
	_, err := q.db.Exec(ctx,
		`DELETE FROM column_references WHERE index_run_id = $1`,
		indexRunID)
	return err
}
