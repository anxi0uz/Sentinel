#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

required_containers=(
  sentinel-kafka
  sentinel-postgres
  sentinel-transaction-gateway
  sentinel-scoring-engine
  sentinel-alert-service
  sentinel-event-generator
  sentinel-web
)

for container in "${required_containers[@]}"; do
  if [[ "$(podman inspect --format '{{.State.Running}}' "$container" 2>/dev/null || true)" != "true" ]]; then
    echo "smoke failed: container $container is not running" >&2
    exit 1
  fi
done

if ! podman exec sentinel-kafka kafka-metadata-quorum \
  --bootstrap-server 127.0.0.1:9092 describe --status >/dev/null 2>&1; then
  echo "smoke failed: Kafka KRaft metadata quorum is not ready" >&2
  exit 1
fi

health=""
for _ in $(seq 1 30); do
  health="$(curl -fsS --connect-timeout 2 --max-time 5 http://localhost:8080/health 2>/dev/null || true)"
  if [[ "$health" == '{"status":"ok"}' ]]; then
    break
  fi
  sleep 1
done
if [[ "$health" != '{"status":"ok"}' ]]; then
  echo "smoke failed: gateway did not become healthy; last response: $health" >&2
  exit 1
fi

web_health="$(curl -fsS --connect-timeout 2 --max-time 5 http://localhost:3000/health 2>/dev/null || true)"
if [[ "$web_health" != '{"status":"ok"}' ]]; then
  echo "smoke failed: web did not become healthy; last response: $web_health" >&2
  exit 1
fi
web_index="$(curl -fsS --connect-timeout 2 --max-time 5 http://localhost:3000/)"
if [[ "$web_index" != *'<title>Sentinel — Risk Monitor</title>'* ]]; then
  echo "smoke failed: web index is invalid" >&2
  exit 1
fi

new_uuid() {
  if command -v uuidgen >/dev/null 2>&1; then
    uuidgen | tr '[:upper:]' '[:lower:]'
  else
    tr -d '\n' </proc/sys/kernel/random/uuid
  fi
}

sql() {
  podman exec -i sentinel-postgres sh -c \
    'psql -v ON_ERROR_STOP=1 -At -F "|" -U "$POSTGRES_USER" -d "$POSTGRES_DB"' \
    <<<"$1"
}

user_id="$(new_uuid)"
transaction_id=""
response_file="$(mktemp)"

cleanup() {
  rm -f "$response_file"
  sql "
    BEGIN;
    DELETE FROM outbox_events
    WHERE event_key IN (
      SELECT alert.id::text
      FROM alerts AS alert
      JOIN scored_transactions AS scored ON scored.id = alert.scored_transaction_id
      WHERE scored.transaction_id = NULLIF('$transaction_id', '')::uuid
    );
    DELETE FROM alerts
    WHERE scored_transaction_id IN (
      SELECT id FROM scored_transactions
      WHERE transaction_id = NULLIF('$transaction_id', '')::uuid
    );
    DELETE FROM scored_transactions
    WHERE transaction_id = NULLIF('$transaction_id', '')::uuid;
    DELETE FROM users WHERE id = '$user_id';
    COMMIT;
  " >/dev/null 2>&1 || true
}
trap cleanup EXIT

sql "
  INSERT INTO users (id, country, last_ip, last_country, last_seen_at)
  VALUES ('$user_id', 'FI', '127.0.0.1', 'FI', now() - interval '30 minutes');
" >/dev/null

http_status="$(curl -sS -o "$response_file" -w '%{http_code}' \
  --connect-timeout 2 --max-time 15 \
  http://localhost:3000/api/transactions \
  -H 'Content-Type: application/json' \
  -d "{\"user_id\":\"$user_id\",\"amount\":99999.99,\"currency\":\"EUR\",\"ip\":\"1.2.3.4\",\"country\":\"KP\"}")"
response="$(<"$response_file")"

if [[ "$http_status" != "202" ]]; then
  echo "smoke failed: gateway returned HTTP $http_status: $response" >&2
  exit 1
fi
if [[ "$response" =~ \"id\":\"([0-9a-f-]{36})\" ]]; then
  transaction_id="${BASH_REMATCH[1]}"
else
  echo "smoke failed: gateway response has no transaction id: $response" >&2
  exit 1
fi

result=""
for _ in $(seq 1 30); do
  result="$(sql "
    SELECT scored.score, alert.severity, outbox.published_at IS NOT NULL
    FROM scored_transactions AS scored
    JOIN alerts AS alert ON alert.scored_transaction_id = scored.id
    JOIN outbox_events AS outbox ON outbox.event_key = alert.id::text
    WHERE scored.transaction_id = '$transaction_id'
      AND outbox.published_at IS NOT NULL;
  ")"
  if [[ -n "$result" ]]; then
    break
  fi
  sleep 1
done

if [[ -z "$result" ]]; then
  echo "smoke failed: pipeline timed out for transaction $transaction_id" >&2
  exit 1
fi

IFS='|' read -r score severity published <<<"$result"
case "$score" in
  ''|*[!0-9]*)
    echo "smoke failed: invalid score in pipeline result: $result" >&2
    exit 1
    ;;
esac

if (( score >= 120 )); then
  expected_severity="CRITICAL"
elif (( score >= 90 )); then
  expected_severity="HIGH"
elif (( score >= 80 )); then
  expected_severity="MEDIUM"
else
  echo "smoke failed: expected an alerting score, got $score" >&2
  exit 1
fi

if [[ "$severity" != "$expected_severity" ]]; then
  echo "smoke failed: score $score has severity $severity, want $expected_severity" >&2
  exit 1
fi
if [[ "$published" != "t" && "$published" != "true" ]]; then
  echo "smoke failed: outbox event was not published: $result" >&2
  exit 1
fi

detail="$(curl -fsS --connect-timeout 2 --max-time 10 "http://localhost:3000/api/transactions/$transaction_id")"
if [[ "$detail" != *"\"transaction_id\":\"$transaction_id\""* ]]; then
  echo "smoke failed: read API returned the wrong transaction: $detail" >&2
  exit 1
fi
if [[ "$detail" != *"\"score\":$score"* || "$detail" != *"\"severity\":\"$severity\""* ]]; then
  echo "smoke failed: read API score or severity mismatch: $detail" >&2
  exit 1
fi
if [[ "$detail" != *'"delivery_status":"PUBLISHED"'* || "$detail" != *'"amount":99999.99'* ]]; then
  echo "smoke failed: read API snapshot or delivery status mismatch: $detail" >&2
  exit 1
fi
if [[ "$detail" != *'"sanctioned_jurisdiction"'* || "$detail" == *'"north_korea"'* ]]; then
  echo "smoke failed: read API did not use the canonical sanctioned-jurisdiction rule: $detail" >&2
  exit 1
fi

list="$(curl -fsS --connect-timeout 2 --max-time 10 "http://localhost:3000/api/transactions?severity=$severity&min_score=80&limit=10")"
if [[ "$list" != *"\"transaction_id\":\"$transaction_id\""* || "$list" != *'"pagination"'* ]]; then
  echo "smoke failed: transaction list does not contain the smoke event: $list" >&2
  exit 1
fi

stats="$(curl -fsS --connect-timeout 2 --max-time 10 http://localhost:3000/api/stats)"
if [[ "$stats" != *'"processed"'* || "$stats" != *'"by_severity"'* || "$stats" != *'"top_rules"'* ]]; then
  echo "smoke failed: stats API returned an invalid response: $stats" >&2
  exit 1
fi
if [[ "$stats" != *'"sanctioned_jurisdiction"'* || "$stats" == *'"north_korea"'* || "$stats" == *'"name":"KP"'* ]]; then
  echo "smoke failed: stats API did not aggregate by canonical rule name: $stats" >&2
  exit 1
fi

echo "smoke passed: transaction=$transaction_id score=$score severity=$severity outbox=published web-api=verified"
