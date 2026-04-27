# Prasankit API

Go backend for Prasankit.

## Current Scope

Foundation backend skeleton:

- Fiber v3 API
- Health endpoints
- PostgreSQL readiness
- Redis readiness
- MinIO readiness
- SQL migration baseline

Current endpoints:

```text
GET /api/v1/health
GET /api/v1/health/live
GET /api/v1/health/ready
```

Ready response:

```json
{
  "data": {
    "status": "ok",
    "checks": {
      "api": "ok",
      "minio": "ok",
      "postgres": "ok",
      "redis": "ok"
    }
  }
}
```

## Docker

Run through Docker Compose from the repository root:

```text
docker compose up --build prasankit-api
```

## Database Migrations

Schema is managed by SQL migrations with goose.

GORM is used for query/ORM only. Do not use `AutoMigrate` for production schema changes.

goose is run through `go run ...@version` and is not added as an app runtime dependency.

Primary keys use PostgreSQL `UUID` columns, but the Go application generates UUID v7 values before insert. Do not add database-side `gen_random_uuid()` defaults for new primary keys.

Run from `app-api/`:

```bash
export PRASANKIT_DB_DSN="postgres://prasankit:change_me@localhost:15432/prasankit?sslmode=disable"

GOTOOLCHAIN=auto go run github.com/pressly/goose/v3/cmd/goose@v3.27.1 -dir database/migrations postgres "$PRASANKIT_DB_DSN" status
GOTOOLCHAIN=auto go run github.com/pressly/goose/v3/cmd/goose@v3.27.1 -dir database/migrations postgres "$PRASANKIT_DB_DSN" up
```

Or use Make targets from `app-api/`:

```bash
make db-migrate
make goose-status
make goose-up
make goose-down
make goose-create name=create_auth_tables
```

Override the DSN when needed:

```bash
make goose-status GOOSE_DSN="postgres://user:pass@host:5432/dbname?sslmode=require"
```

## Note

This skeleton uses Fiber v3. The Dockerfile builds with Go 1.25 because Fiber v3 requires Go 1.25+ and the local machine currently has Go 1.23.5.

The project baseline can be moved to Go 1.26.x by changing the Dockerfile build arg after the target image is available in the local Docker environment.
