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
