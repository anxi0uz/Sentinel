package config

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/anxi0uz/sentinel/pkg/configs"
	"github.com/joho/godotenv"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Config struct {
	configs.BaseConfig
	Server ServerConfig `koanf:"server"`
}

type ServerConfig struct {
	Host            string `koanf:"host"`
	Port            int    `koanf:"port"`
	ReadTimeout     string `koanf:"readTimeout"`
	WriteTimeout    string `koanf:"writeTimeout"`
	IdleTimeout     string `koanf:"idleTimeout"`
	ReadTimeoutDur  time.Duration
	WriteTimeoutDur time.Duration
	IdleTimeoutDur  time.Duration
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

	if err := cfg.parseDurations(); err != nil {
		return nil, fmt.Errorf("ошибка парсинга длительностей: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("конфигурация невалидна: %w", err)
	}

	return &cfg, nil
}

func (c *Config) parseDurations() error {
	parse := func(name, s string) (time.Duration, error) {
		d, e := time.ParseDuration(s)
		if e != nil {
			return 0, fmt.Errorf("%s %q: %w", name, s, e)
		}
		return d, nil
	}

	var err error
	c.Server.ReadTimeoutDur, err = parse("readTimeout", c.Server.ReadTimeout)
	if err != nil {
		return err
	}
	c.Server.WriteTimeoutDur, err = parse("writeTimeout", c.Server.WriteTimeout)
	if err != nil {
		return err
	}
	c.Server.IdleTimeoutDur, err = parse("idleTimeout", c.Server.IdleTimeout)
	if err != nil {
		return err
	}
	return nil
}

func (c *Config) setDefaults() {
	if c.Server.Host == "" {
		c.Server.Host = "0.0.0.0"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Server.ReadTimeout == "" {
		c.Server.ReadTimeout = "10s"
	}
	if c.Server.WriteTimeout == "" {
		c.Server.WriteTimeout = "30s"
	}
	if c.Server.IdleTimeout == "" {
		c.Server.IdleTimeout = "60s"
	}
	if len(c.Kafka.Brokers) == 0 {
		c.Kafka.Brokers = []string{"localhost:9092"}
	}
}

func (c *Config) validate() error {
	if len(c.Kafka.Brokers) == 0 {
		return fmt.Errorf("kafka.brokers обязателен")
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
