-- +goose Up
ALTER TABLE scored_transactions
    ADD COLUMN IF NOT EXISTS transaction_payload jsonb;

-- +goose Down
ALTER TABLE scored_transactions
    DROP COLUMN IF EXISTS transaction_payload;
