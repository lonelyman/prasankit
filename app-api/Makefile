GOOSE_VERSION ?= v3.27.1
GOOSE_DIR ?= database/migrations
GOOSE_DRIVER ?= postgres
GOOSE_DSN ?= postgres://prasankit:change_me@localhost:15432/prasankit?sslmode=disable
GOOSE_CMD = GOTOOLCHAIN=auto go run github.com/pressly/goose/v3/cmd/goose@$(GOOSE_VERSION)

.PHONY: goose-status goose-up goose-down goose-redo goose-create db-migrate

goose-status:
	$(GOOSE_CMD) -dir $(GOOSE_DIR) $(GOOSE_DRIVER) "$(GOOSE_DSN)" status

goose-up:
	$(GOOSE_CMD) -dir $(GOOSE_DIR) $(GOOSE_DRIVER) "$(GOOSE_DSN)" up

db-migrate: goose-up goose-status

goose-down:
	$(GOOSE_CMD) -dir $(GOOSE_DIR) $(GOOSE_DRIVER) "$(GOOSE_DSN)" down

goose-redo:
	$(GOOSE_CMD) -dir $(GOOSE_DIR) $(GOOSE_DRIVER) "$(GOOSE_DSN)" redo

goose-create:
	@test -n "$(name)" || (echo "usage: make goose-create name=create_table_name" && exit 1)
	$(GOOSE_CMD) -dir $(GOOSE_DIR) create "$(name)" sql
