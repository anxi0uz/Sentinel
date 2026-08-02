-- +goose Up
CREATE TABLE IF NOT EXISTS users (
    id           uuid PRIMARY KEY,
    country      text NOT NULL,
    last_ip      text NOT NULL,
    last_country text NOT NULL,
    last_seen_at timestamptz NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
-- This migration may adopt a users table created before gateway-specific
-- Goose history existed. Never drop user data during rollback.
SELECT 1;
