package config

import (
	"context"
	"testing"
)

func TestNewConfigLoadsEmbeddedBaseConfigFromEnvironment(t *testing.T) {
	t.Setenv("SENTINEL_DATABASE_HOST", "postgres")
	t.Setenv("SENTINEL_DATABASE_PORT", "5432")
	t.Setenv("SENTINEL_DATABASE_USER", "sentinel")
	t.Setenv("SENTINEL_DATABASE_PASSWORD", "secret")
	t.Setenv("SENTINEL_DATABASE_NAME", "sentinel")
	t.Setenv("SENTINEL_DATABASE_SSLMODE", "disable")
	t.Setenv("SENTINEL_KAFKA_BROKERS", "kafka-1:9092,kafka-2:9092")

	cfg, err := NewConfig(context.Background(), t.TempDir()+"/missing.toml")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Database.Host != "postgres" || cfg.Database.Port != 5432 {
		t.Fatalf("database config was not loaded: %+v", cfg.Database)
	}
	if len(cfg.Kafka.Brokers) != 2 || cfg.Kafka.Brokers[0] != "kafka-1:9092" {
		t.Fatalf("kafka brokers were not loaded: %v", cfg.Kafka.Brokers)
	}
}
