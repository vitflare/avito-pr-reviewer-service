package config

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func MustInitDB(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.PostgresHost, cfg.PostgresPort, cfg.PostgresUser, cfg.PostgresPass, cfg.PostgresDatabase, cfg.PostgresSSLMode,
	)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pool config: %w", err)
	}

	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = cfg.HealthCheckPeriod

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	pingAttemptsLeft := 3
	var pingErr error

	for i := 0; i < pingAttemptsLeft; i++ {
		pingCtx, pingCancel := context.WithTimeout(ctx, 5*time.Second)
		pingErr = pool.Ping(pingCtx)
		pingCancel()

		if pingErr == nil {
			break
		}

		slog.Warn("failed to ping database",
			slog.Int("attempt", i+1),
			slog.String("error", pingErr.Error()),
		)

		if i < pingAttemptsLeft-1 {
			time.Sleep(500 * time.Millisecond)
		}
	}

	if pingErr != nil {
		return nil, fmt.Errorf("failed to ping pool: %w", pingErr)
	}

	return pool, nil
}
