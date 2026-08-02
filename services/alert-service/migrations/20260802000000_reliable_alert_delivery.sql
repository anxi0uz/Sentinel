-- +goose Up
-- Older versions could persist the same transaction more than once. Prefer
-- the newest result which already has an alert; otherwise keep the newest
-- result. Move historical alerts to that survivor before removing duplicates.
CREATE TEMP TABLE scored_transaction_survivors AS
SELECT id,
       first_value(id) OVER (
           PARTITION BY transaction_id
           ORDER BY EXISTS (
               SELECT 1 FROM alerts WHERE scored_transaction_id = scored_transactions.id
           ) DESC,
           processed_at DESC,
           id
       ) AS survivor_id
FROM scored_transactions;

UPDATE alerts AS alert
SET scored_transaction_id = survivor.survivor_id
FROM scored_transaction_survivors AS survivor
WHERE alert.scored_transaction_id = survivor.id
  AND survivor.id <> survivor.survivor_id;

WITH duplicate_alerts AS (
    SELECT id,
           row_number() OVER (
               PARTITION BY scored_transaction_id
               ORDER BY created_at DESC, id
           ) AS position
    FROM alerts
)
DELETE FROM alerts
WHERE id IN (SELECT id FROM duplicate_alerts WHERE position > 1);

DELETE FROM scored_transactions AS scored
USING scored_transaction_survivors AS survivor
WHERE scored.id = survivor.id
  AND survivor.id <> survivor.survivor_id;

DROP TABLE scored_transaction_survivors;

CREATE UNIQUE INDEX IF NOT EXISTS scored_transactions_transaction_id_uidx
    ON scored_transactions (transaction_id);

CREATE UNIQUE INDEX IF NOT EXISTS alerts_scored_transaction_id_uidx
    ON alerts (scored_transaction_id);

CREATE TABLE outbox_events (
    id              uuid PRIMARY KEY,
    topic           text NOT NULL,
    event_key       text NOT NULL,
    payload         jsonb NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    next_attempt_at timestamptz NOT NULL DEFAULT now(),
    published_at    timestamptz,
    attempts        integer NOT NULL DEFAULT 0,
    last_error      text,
    UNIQUE (topic, event_key)
);

CREATE INDEX outbox_events_pending_idx
    ON outbox_events (next_attempt_at, created_at)
    WHERE published_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS outbox_events;
DROP INDEX IF EXISTS alerts_scored_transaction_id_uidx;
DROP INDEX IF EXISTS scored_transactions_transaction_id_uidx;
