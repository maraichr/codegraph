package graph

import (
	"context"
	"fmt"

	"github.com/maraichr/codegraph/internal/config"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Client wraps the Neo4j driver and provides graph operations.
type Client struct {
	driver neo4j.DriverWithContext
}

// NewClient creates a new Neo4j client from configuration.
func NewClient(cfg config.Neo4jConfig) (*Client, error) {
	driver, err := neo4j.NewDriverWithContext(cfg.URI, neo4j.BasicAuth(cfg.User, cfg.Password, ""))
	if err != nil {
		return nil, fmt.Errorf("create neo4j driver: %w", err)
	}
	return &Client{driver: driver}, nil
}

// EnsureIndexes creates uniqueness constraints on Symbol(id) and File(id) if they do not exist.
// These constraints create indexes that make MERGE/MATCH by id fast; without them, sync can take many minutes.
func (c *Client) EnsureIndexes(ctx context.Context) error {
	session := c.Session(ctx)
	defer session.Close(ctx)
	_, err := neo4j.ExecuteWrite(ctx, session, func(tx neo4j.ManagedTransaction) (any, error) {
		if _, err := tx.Run(ctx, CreateConstraintSymbolID, nil); err != nil {
			return struct{}{}, fmt.Errorf("create symbol id constraint: %w", err)
		}
		if _, err := tx.Run(ctx, CreateConstraintFileID, nil); err != nil {
			return struct{}{}, fmt.Errorf("create file id constraint: %w", err)
		}
		return struct{}{}, nil
	})
	return err
}

// Close releases the Neo4j driver resources.
func (c *Client) Close(ctx context.Context) error {
	return c.driver.Close(ctx)
}

// Verify checks connectivity to Neo4j.
func (c *Client) Verify(ctx context.Context) error {
	return c.driver.VerifyConnectivity(ctx)
}

// Session returns a new Neo4j session.
func (c *Client) Session(ctx context.Context) neo4j.SessionWithContext {
	return c.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
}
