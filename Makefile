ifneq (,$(wildcard .env))
include .env
export
endif

POSTGRES_PRIMARY_USER ?= prasankit
POSTGRES_PRIMARY_PASSWORD ?= change_me
POSTGRES_PRIMARY_NAME ?= prasankit
POSTGRES_EXTERNAL_PORT ?= 15432
POSTGRES_SSL_MODE ?= disable
API_EXTERNAL_PORT ?= 18080

GOOSE_DSN ?= postgres://$(POSTGRES_PRIMARY_USER):$(POSTGRES_PRIMARY_PASSWORD)@localhost:$(POSTGRES_EXTERNAL_PORT)/$(POSTGRES_PRIMARY_NAME)?sslmode=$(POSTGRES_SSL_MODE)

.PHONY: env-init compose-config infra-up db-migrate api-up web-up dev-up health ps logs-api down

env-init:
	@test -f .env || cp .env.example .env

compose-config: env-init
	docker compose config >/dev/null

infra-up: env-init
	docker compose up -d --wait prasankit-pgsql prasankit-redis prasankit-minio

db-migrate: env-init
	$(MAKE) -C app-api db-migrate GOOSE_DSN="$(GOOSE_DSN)"

api-up: env-init
	docker compose up -d --build prasankit-api

web-up: env-init
	docker compose up -d --build prasankit-web

dev-up: infra-up db-migrate api-up web-up health

health: env-init
	curl -fsS "http://localhost:$(API_EXTERNAL_PORT)/api/v1/health/ready"

ps:
	docker compose ps

logs-api:
	docker compose logs -f prasankit-api

down:
	docker compose down
