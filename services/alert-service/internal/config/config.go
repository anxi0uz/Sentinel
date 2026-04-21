package config

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/anxi0uz/sentinel/pkg/configs"
	"github.com/joho/godotenv"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Config struct {
	configs.BaseConfig
}

func NewConfig(ctx context.Context, configPath string) (*Config, error) {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		slog.WarnContext(ctx, "не удалось загрузить .env", "error", err)
	}

	k := koanf.New(".")

	if err := k.Load(file.Provider(configPath), toml.Parser()); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("не удалось прочитать config.toml: %w", err)
		}
		slog.InfoContext(ctx, "config.toml не найден — используем только ENV")
	}

	if err := k.Load(env.Provider("SENTINEL_", ".", func(s string) string {
		return strings.ReplaceAll(strings.ToLower(strings.TrimPrefix(s, "SENTINEL_")), "_", ".")
	}), nil); err != nil {
		return nil, fmt.Errorf("ошибка загрузки ENV: %w", err)
	}

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("не удалось размапить конфигурацию: %w", err)
	}

	cfg.setDefaults()

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("конфигурация невалидна: %w", err)
	}

	return &cfg, nil
}

func (c *Config) setDefaults() {
	if len(c.Kafka.Brokers) == 0 {
		c.Kafka.Brokers = []string{"localhost:9092"}
	}
}

func (c *Config) validate() error {
	if len(c.Kafka.Brokers) == 0 {
		return fmt.Errorf("kafka.brokers обязателен")
	}
	if c.Database.Host == "" {
		return fmt.Errorf("database.host обязателен")
	}
	return nil
}

func (c *Config) DatabaseURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.Database.User,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.Name,
		c.Database.SSLMode,
	)
}
