package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func NewConnectionPool(ctx context.Context, connectionString string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, connectionString)
	if err != nil {
		slog.ErrorContext(ctx, "Cant create pool of connection", slog.String("Error", err.Error()))
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		slog.ErrorContext(ctx, "Cant ping connection", slog.String("Error", err.Error()))
		return nil, err
	}

	return pool, nil
}
func RunMigrations(ctx context.Context, dbURL string, migrationsDir string) error {
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return fmt.Errorf("open db for migrations: %w", err)
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("goose dialect: %w", err)
	}

	if err := goose.Up(db, migrationsDir); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}

	slog.InfoContext(ctx, "migrations applied", slog.String("dir", migrationsDir))
	return nil
}
