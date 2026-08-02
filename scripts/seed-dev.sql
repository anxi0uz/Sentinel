INSERT INTO users (id, country, last_ip, last_country, last_seen_at)
VALUES (
    '550e8400-e29b-41d4-a716-446655440000',
    'FI',
    '127.0.0.1',
    'FI',
    now() - interval '30 minutes'
)
ON CONFLICT (id) DO UPDATE SET
    country = EXCLUDED.country,
    last_ip = EXCLUDED.last_ip,
    last_country = EXCLUDED.last_country,
    last_seen_at = EXCLUDED.last_seen_at;
