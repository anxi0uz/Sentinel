-- +goose Up
CREATE TABLE IF NOT EXISTS alerts (
    id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    scored_transaction_id uuid NOT NULL REFERENCES scored_transactions(id),
    severity              text NOT NULL,
    created_at            timestamptz NOT NULL DEFAULT now()
);
-- +goose Down
DROP TABLE IF EXISTS alerts;
