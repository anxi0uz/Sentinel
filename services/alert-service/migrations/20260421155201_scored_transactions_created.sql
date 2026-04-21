-- +goose Up
CREATE TABLE IF NOT EXISTS scored_transactions (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id  uuid NOT NULL,
    score           int NOT NULL,
    triggered_rules text[],
    processed_at    timestamptz NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS scored_transactions;
