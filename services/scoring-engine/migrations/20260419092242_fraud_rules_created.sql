-- +goose Up
CREATE TABLE IF NOT EXISTS fraud_rules (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name        text NOT NULL,
    field       text NOT NULL,
    operator    text NOT NULL,
    threshold   float8,
    values      text[],
    score_delta float8 NOT NULL,
    active      bool DEFAULT true
);

INSERT INTO fraud_rules (name, field, operator, threshold, values, score_delta)
SELECT seed.name, seed.field, seed.operator, seed.threshold, seed.values, seed.score_delta
FROM (VALUES
    ('high_amount', 'amount', 'gt', 50000::float8, null::text[], 40::float8),
    ('north_korea', 'country', 'eq', null::float8, '{"KP"}'::text[], 60::float8)
) AS seed(name, field, operator, threshold, values, score_delta)
WHERE NOT EXISTS (
    SELECT 1 FROM fraud_rules existing WHERE existing.name = seed.name
);


-- +goose Down
DROP TABLE fraud_rules;
