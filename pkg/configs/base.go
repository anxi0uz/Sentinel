package configs

import (
	"log/slog"
	"net"
	"net/url"
	"strconv"
)

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

func (c DatabaseConfig) URL() string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.User, c.Password),
		Host:   net.JoinHostPort(c.Host, strconv.Itoa(c.Port)),
		Path:   c.Name,
	}
	query := u.Query()
	query.Set("sslmode", c.SSLMode)
	u.RawQuery = query.Encode()
	return u.String()
}
