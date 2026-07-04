// Package clickhouse provides ClickHouse connectivity and event writing for DSP analytics.
package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"
)

// Client wraps a ClickHouse connection.
type Client struct {
	db *sql.DB
}

// Config holds ClickHouse connection settings.
type Config struct {
	Host     string
	Port     int
	Database string
	Username string
	Password string
}

// NewClient creates a ClickHouse client and verifies connectivity.
func NewClient(cfg Config) (*Client, error) {
	dsn := fmt.Sprintf("clickhouse://%s:%s@%s:%d/%s",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	db, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return nil, fmt.Errorf("clickhouse open: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("clickhouse ping: %w", err)
	}

	return &Client{db: db}, nil
}

// DB returns the underlying sql.DB for direct queries.
func (c *Client) DB() *sql.DB {
	return c.db
}

// Close closes the ClickHouse connection.
func (c *Client) Close() error {
	return c.db.Close()
}
