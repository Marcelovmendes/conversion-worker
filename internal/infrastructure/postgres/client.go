package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/marcelovmendes/playswap/conversion-worker/internal/config"
)

type Client interface {
	Ping(ctx context.Context) error
	Close()
	GetPool() *pgxpool.Pool
}

type pgClient struct {
	pool *pgxpool.Pool
}

func NewClient(ctx context.Context, cfg config.PostgresConfig) (Client, error) {
	pool, err := pgxpool.New(ctx, cfg.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &pgClient{pool: pool}, nil
}

func (c *pgClient) Ping(ctx context.Context) error {
	return c.pool.Ping(ctx)
}

func (c *pgClient) Close() {
	c.pool.Close()
}

func (c *pgClient) GetPool() *pgxpool.Pool {
	return c.pool
}
