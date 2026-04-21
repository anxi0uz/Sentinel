package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/anxi0uz/sentinel/pkg/database"
	"github.com/anxi0uz/sentinel/pkg/kafka"
	"github.com/anxi0uz/sentinel/services/scoring-engine/internal/cache"
	"github.com/anxi0uz/sentinel/services/scoring-engine/internal/config"
	"github.com/anxi0uz/sentinel/services/scoring-engine/internal/worker"
	"github.com/golang-cz/devslog"
)

func NewDevLogger() {
	opts := &devslog.Options{
		MaxSlicePrintSize: 4,
		SortKeys:          true,
		TimeFormat:        "15:04:05.000",
		NewLineAfterLog:   true,
		DebugColor:        devslog.Cyan,
		StringerFormatter: true,
	}

	handler := devslog.NewHandler(os.Stdout, opts)
	logger := slog.New(handler)

	slog.SetDefault(logger)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	NewDevLogger()
	cfg, err := config.NewConfig(ctx, "configs/config.toml")
	if err != nil {
		slog.ErrorContext(ctx, "error to create config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	writer := kafka.NewWriter(cfg.Kafka.Brokers, "scored")
	reader := kafka.NewReader(cfg.Kafka.Brokers, "transactions", "scoring-engine")
	defer func() {
		if err := writer.Close(); err != nil {
			slog.Error("writer close error", slog.String("error", err.Error()))
		}
	}()
	defer func() {
		if err := reader.Close(); err != nil {
			slog.Error("reader close error", slog.String("error", err.Error()))
		}
	}()

	cache := &cache.RuleCache{}
	if err := database.RunMigrations(ctx, cfg.DatabaseURL(), "./migrations"); err != nil {
		slog.ErrorContext(ctx, "error to apply migrations", slog.String("Error", err.Error()))
		os.Exit(1)
	}

	pool, err := database.NewConnectionPool(ctx, cfg.DatabaseURL())
	if err != nil {
		slog.ErrorContext(ctx, "failed to create connection pool", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	if err := cache.Start(ctx, pool, cfg.CacheInterval); err != nil {
		slog.ErrorContext(ctx, "failed to load rule cache", slog.String("error", err.Error()))
		os.Exit(1)
	}

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	workerPool := worker.New(reader, writer, cache, cfg.Workers)

	go func() {
		workerPool.Run(ctx)
	}()
	select {
	case sig := <-sigchan:
		slog.Info("received signal", slog.String("signal", sig.String()))
	case <-ctx.Done():
	}
	cancel()
}
