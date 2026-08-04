-- +goose Up
UPDATE scored_transactions
SET triggered_rules = array_replace(triggered_rules, 'north_korea', 'sanctioned_jurisdiction')
WHERE triggered_rules @> ARRAY['north_korea'];

-- +goose Down
UPDATE scored_transactions
SET triggered_rules = array_replace(triggered_rules, 'sanctioned_jurisdiction', 'north_korea')
WHERE triggered_rules @> ARRAY['sanctioned_jurisdiction'];
