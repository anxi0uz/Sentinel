-- +goose Up
UPDATE fraud_rules
SET name = 'sanctioned_jurisdiction'
WHERE name = 'north_korea'
  AND NOT EXISTS (
      SELECT 1 FROM fraud_rules WHERE name = 'sanctioned_jurisdiction'
  );

DELETE FROM fraud_rules WHERE name = 'north_korea';

-- +goose Down
UPDATE fraud_rules
SET name = 'north_korea'
WHERE name = 'sanctioned_jurisdiction'
  AND NOT EXISTS (
      SELECT 1 FROM fraud_rules WHERE name = 'north_korea'
  );

DELETE FROM fraud_rules WHERE name = 'sanctioned_jurisdiction';
