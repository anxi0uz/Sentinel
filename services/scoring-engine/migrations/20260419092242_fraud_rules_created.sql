-- +goose Up
CREATE TABLE fraud_rules (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name        text NOT NULL,
    field       text NOT NULL,
    operator    text NOT NULL,
    threshold   float8,
    values      text[],
    score_delta float8 NOT NULL,
    active      bool DEFAULT true
);

INSERT INTO fraud_rules (name, field, operator, threshold, values, score_delta) VALUES
    ('high_amount', 'amount', 'gt', 50000, null, 40),
    ('north_korea', 'country', 'eq', null, '{"KP"}', 60);


-- +goose Down
DROP TABLE fraud_rules;
