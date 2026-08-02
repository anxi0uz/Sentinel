# Sentinel

Sentinel is a small real-time fraud detection pipeline written in Go. It accepts
transactions over HTTP, evaluates database-backed fraud rules and persists
alerts using an event-driven architecture.

## Architecture

```text
POST /transactions
        │
        ▼
transaction-gateway ── Kafka: transactions ──▶ scoring-engine
                                                    │
                                                    ▼
                                             Kafka: scored
                                                    │
                                                    ▼
                                              alert-service
                                                    │
                                      PostgreSQL + transactional outbox
                                                    │
                                                    ▼
                                              Kafka: alerts
```

The repository currently contains three services:

| Service | Responsibility |
| --- | --- |
| `transaction-gateway` | Validates HTTP requests, enriches transactions with an existing user profile and publishes them to Kafka |
| `scoring-engine` | Caches fraud rules, scores transactions in a partition-ordered worker pool and publishes the result |
| `alert-service` | Idempotently persists scoring results and alerts, then publishes alerts through a transactional outbox |

## Delivery guarantees

- Kafka producers wait for acknowledgements from all in-sync replicas.
- The scoring worker preserves processing and offset commit order within each
  Kafka partition.
- `alert-service` commits the consumed Kafka message only after its PostgreSQL
  transaction succeeds.
- `scored_transactions.transaction_id` makes redelivery idempotent.
- Alert persistence and outbox enqueue happen in the same database transaction.
- Outbox publication is at-least-once. Consumers of the `alerts` topic should
  deduplicate by the Kafka message key (the alert ID).

## Run locally

Requirements: Go 1.26+, Podman and `podman-compose`.

```bash
cp .env.example .env
make up
make seed
```

`make up` builds all three services and starts Kafka, PostgreSQL and Zookeeper.
The development seed creates the user referenced by the example request.

Submit a transaction:

```bash
curl -i http://localhost:8080/transactions \
  -H 'Content-Type: application/json' \
  -d '{
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "amount": 99999.99,
    "currency": "EUR",
    "ip": "1.2.3.4",
    "country": "KP"
  }'
```

The endpoint returns `202 Accepted`. The default rules produce a high-risk score
and `alert-service` persists an alert before publishing it to the `alerts` topic.

Stop the stack with:

```bash
make down
```

To run only the infrastructure and start Go services from separate terminals:

```bash
make infra-up
make run-gateway
make run-scoring
make run-alert
```

Each service owns its Goose version table, so their migrations can run in any
startup order while sharing the development database.

## Development

```bash
make test
make test-race
make vet
make acceptance
```

The focused tests cover fraud rule evaluation, impossible-travel edge cases,
Kafka partition ordering, publish/commit failure semantics, request validation
and alert severity boundaries.

`make acceptance` expects the Compose stack to be running. It seeds the demo
user, submits a transaction and waits until the score, alert and published
outbox record are visible in PostgreSQL.

## Project layout

```text
pkg/
  configs/      shared configuration types
  database/     PostgreSQL pool and Goose helpers
  kafka/        Kafka reader/writer constructors
  models/       shared event and persistence models
  outbox/       PostgreSQL transactional outbox publisher
  storage/      generic PostgreSQL helpers
services/
  transaction-gateway/
  scoring-engine/
  alert-service/
scripts/
  seed-dev.sql
podman-compose.yml
Makefile
```
