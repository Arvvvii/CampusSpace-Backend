package database

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/Arvvvii/CampusSpace-Backend/config"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	defaultPingTimeout     = 5 * time.Second
	defaultMaxOpenConns    = 25
	defaultMaxIdleConns    = 5
	defaultConnMaxLifetime = 30 * time.Minute
	defaultConnMaxIdleTime = 5 * time.Minute
)

// NewPostgresDB opens a PostgreSQL connection pool using the pgx stdlib driver,
// configures connection pooling, and verifies connectivity with PingContext.
func NewPostgresDB(cfg config.DatabaseConfig) (*sql.DB, error) {
	dsn := buildPostgresDSN(cfg)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgresql connection: %w", err)
	}

	// Configure connection pool to bound resource usage and recycle stale connections.
	db.SetMaxOpenConns(defaultMaxOpenConns)
	db.SetMaxIdleConns(defaultMaxIdleConns)
	db.SetConnMaxLifetime(defaultConnMaxLifetime)
	db.SetConnMaxIdleTime(defaultConnMaxIdleTime)

	ctx, cancel := context.WithTimeout(context.Background(), defaultPingTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping postgresql database: %w", err)
	}

	return db, nil
}

func buildPostgresDSN(cfg config.DatabaseConfig) string {
	dsn := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.User, cfg.Password),
		Host:   net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
		Path:   cfg.Name,
	}

	query := dsn.Query()
	if cfg.SSLMode != "" {
		query.Set("sslmode", cfg.SSLMode)
	}
	dsn.RawQuery = query.Encode()

	return dsn.String()
}

