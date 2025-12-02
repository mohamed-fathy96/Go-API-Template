package db

import (
	"context"
	"database/sql"
	"fmt"

	entsql "entgo.io/ent/dialect/sql"

	"kabsa/ent"
	"kabsa/internal/config"
	"kabsa/internal/logging"

	_ "github.com/jackc/pgx/v5/stdlib" // register pgx as a database/sql driver
)

type Client struct {
	ent    *ent.Client
	db     *sql.DB
	logger logging.Logger
}

// NewClient creates an Ent client backed by database/sql using the pgx driver.
func NewClient(ctx context.Context, cfg config.PostgresConfig, logger logging.Logger) (*Client, error) {
	dsn := cfg.EffectiveDSN()

	// database/sql connection pool using pgx
	dbStd, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	// Verify connectivity
	if err := dbStd.PingContext(ctx); err != nil {
		_ = dbStd.Close()
		return nil, fmt.Errorf("db ping: %w", err)
	}

	// Ent driver from *sql.DB
	drv := entsql.OpenDB("pgx", dbStd)
	entClient := ent.NewClient(ent.Driver(drv))

	return &Client{
		ent:    entClient,
		db:     dbStd,
		logger: logger.With("component", "db_client"),
	}, nil
}

// Ent returns the underlying Ent client.
func (c *Client) Ent() *ent.Client {
	return c.ent
}

// Close closes both the Ent client and the underlying DB pool.
func (c *Client) Close() error {
	if err := c.ent.Close(); err != nil {
		_ = c.db.Close()
		return err
	}
	return c.db.Close()
}

// Ping is used by health checks.
func (c *Client) Ping(ctx context.Context) error {
	return c.db.PingContext(ctx)
}
