package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	appkafka "github.com/anxi0uz/sentinel/pkg/kafka"
	"github.com/anxi0uz/sentinel/services/event-generator/internal/generator"
	"github.com/segmentio/kafka-go"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	brokers := envList("SENTINEL_KAFKA_BROKERS", []string{"localhost:9092"})
	interval := envDuration("SENTINEL_GENERATOR_INTERVAL", 2*time.Second)
	count := envInt("SENTINEL_GENERATOR_COUNT", 0)
	scenario := generator.ParseScenario(os.Getenv("SENTINEL_GENERATOR_SCENARIO"))

	writer := appkafka.NewWriter(brokers)
	defer func() {
		if err := writer.Close(); err != nil {
			slog.Error("cannot close Kafka writer", slog.String("error", err.Error()))
		}
	}()

	slog.Info("event generator started",
		slog.Any("brokers", brokers),
		slog.Duration("interval", interval),
		slog.Int("count", count),
		slog.String("scenario", string(scenario)),
	)

	published := 0
	for {
		event := generator.NewEvent(scenario, time.Now().UTC())
		payload, err := json.Marshal(event)
		if err != nil {
			slog.Error("cannot marshal generated event", slog.String("error", err.Error()))
			os.Exit(1)
		}
		if err := writer.WriteMessages(ctx, kafka.Message{
			Topic: "transactions",
			Key:   []byte(event.Transaction.ID.String()),
			Value: payload,
		}); err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("cannot publish generated event", slog.String("error", err.Error()))
			os.Exit(1)
		}

		published++
		slog.Info("event generated",
			slog.String("transaction_id", event.Transaction.ID.String()),
			slog.Float64("amount", event.Transaction.Amount),
			slog.String("country", event.Transaction.Country),
		)
		if count > 0 && published >= count {
			return
		}

		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func envList(name string, fallback []string) []string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			result = append(result, part)
		}
	}
	if len(result) == 0 {
		return fallback
	}
	return result
}

func envDuration(name string, fallback time.Duration) time.Duration {
	value, err := time.ParseDuration(strings.TrimSpace(os.Getenv(name)))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func envInt(name string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv(name)))
	if err != nil || value < 0 {
		return fallback
	}
	return value
}
