package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/codegraph-labs/codegraph/internal/store/postgres"
)

type Store struct {
	*postgres.Queries
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{
		Queries: postgres.New(pool),
		pool:    pool,
	}
}

func (s *Store) Pool() *pgxpool.Pool {
	return s.pool
}

func (s *Store) WithTx(ctx context.Context, fn func(*postgres.Queries) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := fn(s.Queries.WithTx(tx)); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
