package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/anxi0uz/sentinel/pkg/database"
	"github.com/anxi0uz/sentinel/pkg/kafka"
	"github.com/anxi0uz/sentinel/services/transaction-gateway/internal/api"
	"github.com/anxi0uz/sentinel/services/transaction-gateway/internal/config"
	"github.com/anxi0uz/sentinel/services/transaction-gateway/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-cz/devslog"
	slogchi "github.com/samber/slog-chi"
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
		slog.ErrorContext(ctx, "cant load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	slog.SetLogLoggerLevel(cfg.LogLevel)

	writer := kafka.NewWriter(cfg.Kafka.Brokers, "transactions")
	defer func() {
		if err := writer.Close(); err != nil {
			slog.ErrorContext(ctx, "error while closing kafka writer", slog.String("error", err.Error()))
		}
	}()

	pool, err := database.NewConnectionPool(ctx, cfg.DatabaseURL())
	if err != nil {
		slog.ErrorContext(ctx, "error while creating postgres pool", slog.String("error", err.Error()))
		os.Exit(1)
	}

	producer := service.NewProducer(writer)
	server := api.NewServer(producer, pool)

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(slogchi.NewWithConfig(slog.Default(), slogchi.Config{
		DefaultLevel:     slog.LevelInfo,
		ClientErrorLevel: slog.LevelWarn,
		ServerErrorLevel: slog.LevelError,
		Filters: []slogchi.Filter{
			slogchi.IgnorePath("/health"),
		},
	}))

	h := api.HandlerFromMux(server, r)
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      h,
		ReadTimeout:  cfg.Server.ReadTimeoutDur,
		IdleTimeout:  cfg.Server.IdleTimeoutDur,
		WriteTimeout: cfg.Server.WriteTimeoutDur,
	}

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("starting transaction-gateway", slog.String("addr", srv.Addr))
		serverErr <- srv.ListenAndServe()
	}()

	select {
	case sig := <-sigchan:
		slog.Info("received signal", slog.String("signal", sig.String()))
	case err := <-serverErr:
		slog.ErrorContext(ctx, "server error", slog.String("error", err.Error()))
		os.Exit(1)
	}
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("shutdown error", slog.String("error", err.Error()))
	}

	slog.Info("transaction-gateway stopped")
}
