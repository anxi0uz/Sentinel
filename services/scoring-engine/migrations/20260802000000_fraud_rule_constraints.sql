-- +goose Up
UPDATE fraud_rules SET threshold = 0 WHERE threshold IS NULL;
UPDATE fraud_rules SET values = '{}' WHERE values IS NULL;
UPDATE fraud_rules SET active = true WHERE active IS NULL;

ALTER TABLE fraud_rules
    ALTER COLUMN threshold SET DEFAULT 0,
    ALTER COLUMN threshold SET NOT NULL,
    ALTER COLUMN values SET DEFAULT '{}',
    ALTER COLUMN values SET NOT NULL,
    ALTER COLUMN active SET NOT NULL;

-- +goose Down
ALTER TABLE fraud_rules
    ALTER COLUMN threshold DROP NOT NULL,
    ALTER COLUMN threshold DROP DEFAULT,
    ALTER COLUMN values DROP NOT NULL,
    ALTER COLUMN values DROP DEFAULT,
    ALTER COLUMN active DROP NOT NULL;
