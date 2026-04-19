# Sentinel

Real-time fraud detection system built with Go. Event-driven architecture using Apache Kafka as the message bus, PostgreSQL for persistence, and LLM-powered analytics for pattern insights.

## Architecture

```
┌─────────────────┐     ┌──────────────────┐     ┌───────────────┐
│ transaction-    │────▶│  scoring-engine  │────▶│ alert-service │
│ gateway         │     │  (worker pool)   │     │               │
└─────────────────┘     └──────────────────┘     └───────┬───────┘
        │                        │                        │
        │ enriches with user      │ rule-based scoring     │ persists
        ▼                        ▼                        ▼
   PostgreSQL              fraud_rules DB           fraud_events DB
                           (RuleCache)                     │
                                                           ▼
                                                  ┌─────────────────┐
                                                  │ notification-   │
                                                  │ service         │
                                                  └─────────────────┘

┌─────────────────┐     ┌──────────────────┐
│  analytics-api  │     │ insight-service  │
│  (HTTP API)     │     │ (LLM scheduler)  │
└─────────────────┘     └──────────────────┘
```

**Pipeline:**
```
POST /transactions → Kafka [transactions] → scoring-engine → Kafka [scored] → alert-service → Kafka [alerts] → notification-service
```

## Services

| Service | Responsibility |
|---------|---------------|
| `transaction-gateway` | HTTP intake, enriches transaction with user data, publishes to Kafka |
| `scoring-engine` | Worker pool, rule-based scoring, emits scored transactions |
| `alert-service` | Consumes scored transactions, persists fraud events |
| `notification-service` | Consumes alerts, rate-limited notifications |
| `analytics-api` | HTTP API over fraud data |
| `insight-service` | Periodic LLM analysis of fraud patterns |

## Tech Stack

- **Go** — all services
- **Apache Kafka** — inter-service messaging
- **PostgreSQL** — persistence
- **Chi** — HTTP router
- **pgxpool** — PostgreSQL connection pooling
- **oapi-codegen** — OpenAPI-first, generated Chi handlers
- **koanf** — configuration (TOML + ENV)
- **Ollama / OpenAI** — LLM analytics

## Fraud Rules

Rules are stored in PostgreSQL and cached in-memory with periodic reload:

```sql
CREATE TABLE fraud_rules (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name        text NOT NULL,
    field       text NOT NULL,       -- amount, country
    operator    text NOT NULL,       -- gt, lt, eq, not_in
    threshold   float8,
    score_delta float8 NOT NULL,
    active      bool DEFAULT true
);
```

Transactions scoring above **80 points** trigger an alert.

Example rules:
- `amount gt 50000` → +40 pts
- `country eq "KP"` → +60 pts
- `impossible_travel` → +80 pts (country change within impossible timeframe)

## Getting Started

**Prerequisites:** Go 1.25+, Podman

```bash
# Create network
podman network create SentinelNetwork

# Start infrastructure
podman-compose up -d

# Run transaction-gateway
cd services/transaction-gateway
go run cmd/main.go
```

**Submit a transaction:**
```bash
curl -X POST http://localhost:8080/transactions \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "amount": 99999.99,
    "currency": "USD",
    "ip": "1.2.3.4",
    "country": "KP"
  }'
```

## Project Structure

```
sentinel/
├── pkg/
│   ├── models/       — shared domain models
│   ├── kafka/        — Kafka reader/writer wrappers
│   └── configs/      — shared base config
├── services/
│   ├── transaction-gateway/
│   ├── scoring-engine/
│   ├── alert-service/
│   ├── notification-service/
│   ├── analytics-api/
│   └── insight-service/
├── podman-compose.yml
└── Makefile
```

---

# Sentinel (RU)

Система обнаружения мошеннических транзакций в реальном времени на Go. Event-driven архитектура с Apache Kafka в качестве шины событий, PostgreSQL для хранения данных и LLM-аналитикой для выявления паттернов.

## Архитектура

```
┌─────────────────┐     ┌──────────────────┐     ┌───────────────┐
│ transaction-    │────▶│  scoring-engine  │────▶│ alert-service │
│ gateway         │     │  (worker pool)   │     │               │
└─────────────────┘     └──────────────────┘     └───────┬───────┘
        │                        │                        │
        │ обогащает данными юзера │ скоринг по правилам   │ сохраняет
        ▼                        ▼                        ▼
   PostgreSQL              fraud_rules DB           fraud_events DB
                           (RuleCache)                     │
                                                           ▼
                                                  ┌─────────────────┐
                                                  │ notification-   │
                                                  │ service         │
                                                  └─────────────────┘

┌─────────────────┐     ┌──────────────────┐
│  analytics-api  │     │ insight-service  │
│  (HTTP API)     │     │ (LLM планировщик)│
└─────────────────┘     └──────────────────┘
```

**Pipeline:**
```
POST /transactions → Kafka [transactions] → scoring-engine → Kafka [scored] → alert-service → Kafka [alerts] → notification-service
```

## Сервисы

| Сервис | Ответственность |
|--------|----------------|
| `transaction-gateway` | Приём HTTP запросов, обогащение транзакции данными юзера, публикация в Kafka |
| `scoring-engine` | Worker pool, скоринг по правилам, публикация результатов |
| `alert-service` | Потребление scored транзакций, сохранение fraud событий |
| `notification-service` | Потребление алертов, отправка уведомлений с rate limiting |
| `analytics-api` | HTTP API для аналитики по fraud данным |
| `insight-service` | Периодический LLM-анализ паттернов мошенничества |

## Стек

- **Go** — все сервисы
- **Apache Kafka** — коммуникация между сервисами
- **PostgreSQL** — хранение данных
- **Chi** — HTTP роутер
- **pgxpool** — пул соединений с PostgreSQL
- **oapi-codegen** — OpenAPI-first, генерация Chi хендлеров
- **koanf** — конфигурация (TOML + ENV)
- **Ollama / OpenAI** — LLM аналитика

## Правила скоринга

Правила хранятся в PostgreSQL и кешируются в памяти с периодическим обновлением:

```sql
CREATE TABLE fraud_rules (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name        text NOT NULL,
    field       text NOT NULL,       -- amount, country
    operator    text NOT NULL,       -- gt, lt, eq, not_in
    threshold   float8,
    score_delta float8 NOT NULL,
    active      bool DEFAULT true
);
```

Транзакции набравшие более **80 баллов** получают статус фрода.

Примеры правил:
- `amount gt 50000` → +40 баллов
- `country eq "KP"` → +60 баллов
- `impossible_travel` → +80 баллов (смена страны за физически невозможное время)

## Запуск

**Требования:** Go 1.25+, Podman

```bash
# Создать сеть
podman network create SentinelNetwork

# Поднять инфраструктуру
podman-compose up -d

# Запустить transaction-gateway
cd services/transaction-gateway
go run cmd/main.go
```

**Отправить транзакцию:**
```bash
curl -X POST http://localhost:8080/transactions \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "amount": 99999.99,
    "currency": "USD",
    "ip": "1.2.3.4",
    "country": "KP"
  }'
```

## Структура проекта

```
sentinel/
├── pkg/
│   ├── models/       — общие доменные модели
│   ├── kafka/        — обёртки над Kafka reader/writer
│   └── configs/      — базовый конфиг
├── services/
│   ├── transaction-gateway/
│   ├── scoring-engine/
│   ├── alert-service/
│   ├── notification-service/
│   ├── analytics-api/
│   └── insight-service/
├── podman-compose.yml
└── Makefile
```
