ifneq (,$(wildcard .env))
include .env
export
endif

.PHONY: generate test test-race vet smoke acceptance infra-up infra-down up down seed run-gateway run-scoring run-alert run-generator

generate:
	cd services/transaction-gateway && oapi-codegen --config configs/oapi-cfg.yaml internal/api/api.swagger.yaml

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

smoke:
	bash scripts/smoke.sh

acceptance: smoke

infra-up:
	podman-compose up -d kafka postgres

infra-down:
	podman-compose down

up:
	podman-compose build
	podman-compose up -d kafka postgres
	podman-compose up -d --no-deps --force-recreate transaction-gateway scoring-engine alert-service event-generator web

down:
	podman-compose down

seed:
	podman exec -i sentinel-postgres sh -c 'psql -v ON_ERROR_STOP=1 -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"' < scripts/seed-dev.sql

run-gateway:
	cd services/transaction-gateway && go run ./cmd

run-scoring:
	cd services/scoring-engine && go run ./cmd

run-alert:
	cd services/alert-service && go run ./cmd

run-generator:
	go run ./services/event-generator/cmd
