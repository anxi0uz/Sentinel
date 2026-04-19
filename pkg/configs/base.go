package configs

import "log/slog"

type BaseConfig struct {
	LogLevel slog.Level     `koanf:"logLevel"`
	Kafka    KafkaConfig    `koanf:"kafka"`
	Database DatabaseConfig `koanf:"database"`
}
type KafkaConfig struct {
	Brokers []string `koanf:"brokers"`
}
type DatabaseConfig struct {
	Host     string `koanf:"host"`
	Port     int    `koanf:"port"`
	User     string `koanf:"user"`
	Password string `koanf:"password"`
	Name     string `koanf:"name"`
	SSLMode  string `koanf:"sslmode"`
}
