package configs

import (
	"net/url"
	"testing"
)

func TestDatabaseConfigURL(t *testing.T) {
	cfg := DatabaseConfig{
		Host:     "postgres.internal",
		Port:     5432,
		User:     "sentinel@app",
		Password: "s3cr:t/p@ss",
		Name:     "sentinel",
		SSLMode:  "require",
	}

	dsn := cfg.URL()
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse database URL: %v", err)
	}
	if parsed.Scheme != "postgres" {
		t.Fatalf("scheme = %q, want postgres", parsed.Scheme)
	}
	if parsed.Host != "postgres.internal:5432" {
		t.Fatalf("host = %q, want postgres.internal:5432", parsed.Host)
	}
	if parsed.User.Username() != cfg.User {
		t.Fatalf("user = %q, want %q", parsed.User.Username(), cfg.User)
	}
	password, ok := parsed.User.Password()
	if !ok || password != cfg.Password {
		t.Fatalf("password = %q, %v; want %q, true", password, ok, cfg.Password)
	}
	if parsed.Path != "/sentinel" {
		t.Fatalf("path = %q, want /sentinel", parsed.Path)
	}
	if parsed.Query().Get("sslmode") != "require" {
		t.Fatalf("sslmode = %q, want require", parsed.Query().Get("sslmode"))
	}
}
