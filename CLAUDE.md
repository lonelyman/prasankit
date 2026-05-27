# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Context to Read First

Documentation is organized so each topic has a single canonical home:

| Topic | Canonical home |
| --- | --- |
| Hard rules + MVP-0 scope + stack snapshot | `docs/00-context-guard.md` |
| Stack version baselines | `docs/00-tech-stack-architecture-baseline.md` |
| Tenant / data isolation model | `docs/06-tenant-data-isolation.md` |
| API design (spec + implemented/planned per section) | `docs/09-api-design.md` |
| MVP-0 scope (user flow + build plan) | `docs/14-mvp0-user-flow.md`, `docs/15-mvp0-build-plan.md` |
| Source tree (target + current-state note) | `docs/13-project-source-tree.md` |
| API examples for implemented endpoints | `docs/18-api-test-examples.md` + `docs/18.x-*.md` |
| Current state (what's built, what's next) | `docs/99-current-handoff.md` |

`README.md` and `AGENTS.md` at the repo root link to these and add only what isn't covered elsewhere (commands, composition rule, review checklist). They no longer restate the hard rules.

Docs **01, 03, 04, 05, 07, 10, 11** describe MVP-1+ design intent and have a clear status banner at the top. Doc **08** (database design) is mixed — it includes future tables; trust the migrations under `app-api/database/migrations/` for current schema. Docs **16, 17** are stubs (superseded; pointers to 99).

**Trust the code over the docs when they disagree.** When extending an existing module, also read the matching `docs/18.x-...-test-examples.md` so request/response shape stays consistent with completed endpoints.

## Common Commands

Bootstrap and run the full stack via Docker Compose from the repo root:

```bash
make env-init      # copies .env.example -> .env if missing
make dev-up        # infra-up + db-migrate + api-up + health
make health        # GET /api/v1/health/ready
make logs-api      # docker compose logs -f prasankit-api
make down          # docker compose down
```

Database migrations (goose, run from `app-api/`):

```bash
make db-migrate                              # goose up + status
make goose-status
make goose-up
make goose-down
make goose-create name=create_task_tables    # new migration
```

The root `make db-migrate` delegates to `app-api/`'s Makefile using a `GOOSE_DSN` built from `.env`. Override DSN when needed: `make goose-status GOOSE_DSN=postgres://...`.

Go tests (from `app-api/`):

```bash
GOTOOLCHAIN=auto go test ./...
GOTOOLCHAIN=auto go test ./internal/modules/auth/authsvc -run TestRegisterEmailPassword -count=1
```

Repository integration tests require a running PostgreSQL and `PRASANKIT_TEST_DB_DSN`:

```bash
PRASANKIT_TEST_DB_DSN='postgres://prasankit:change_me@localhost:15432/prasankit?sslmode=disable' \
  GOTOOLCHAIN=auto go test ./internal/adapters/database/postgres/authrepo -run TestRepositoryIntegration -count=1
```

`GOTOOLCHAIN=auto` is required because the local machine is on Go 1.23.5 but the module declares `go 1.25` (Fiber v3 needs 1.25+). The Dockerfile builds with Go 1.25.

## Backend Architecture

`app-api/` is Go Fiber v3 + GORM + pgx, organized as Pragmatic Hexagonal Architecture. Layer boundaries are strict — do not invert them.

Modules currently implemented (only these four): **auth, workspace, project, task**. Each follows the same shape:

```
cmd/api/main.go                              loads config, calls bootstrap.InitializeApp
internal/bootstrap/                          composition root: app.go, db.go, http.go, redis.go, storage.go
                                             (no mailer.go — SMTP sender is wired directly in app.go)
internal/config/config.go                    env loader; fail-fast on missing required env
internal/modules/{m}/                        domain entity + repository INTERFACE
internal/modules/{m}/{m}svc/                 business service / use case (logic lives here, not in handlers)
internal/modules/workspace/workspaceperm/    role → permission map (Can(role, permission))
internal/adapters/database/postgres/{m}repo/ GORM repository IMPLEMENTATION
internal/adapters/cache/redis/ratelimit/     Redis rate limiter
internal/adapters/cache/redis/sessionstore/  Redis session store
internal/adapters/storage/minioattachment/   MinIO presigned URL signer (signer.go)
internal/adapters/email/smtpemail/           SMTP sender (sender.go)
internal/transport/http/{m}http/             Fiber handler + routes; parse/format only
internal/transport/http/presenter/           response envelopes (data/error/items+pagination)
internal/transport/http/middlewares/         request_id.go, error_handler.go (that's all)
internal/transport/http/health/              health/live/ready handlers
pkg/ids                                      UUID v7 generator
pkg/dbtypes                                  JSONB GORM value type
pkg/passwordhash                             bcrypt adapter
pkg/securetoken                              random token + hash for verification/reset tokens
```

Wiring order in `bootstrap.InitializeApp`: config → infrastructure clients (Postgres, Redis, MinIO) → repositories → services → HTTP handlers → router. `App` owns every resource that needs shutdown and closes it in `Close()`. Keep `bootstrap` small — do not pre-wire placeholder repos/services/handlers for modules that aren't being implemented yet.

Handlers parse and format; they never run business logic. Services orchestrate use cases and call repositories. Repositories own persistence — they set timestamps and generate UUID v7 IDs in Go before insert; do not rely on database defaults for primary keys (no `gen_random_uuid()`) or for important timestamps.

GORM is for query/ORM only. Schema is managed exclusively by SQL migrations under `app-api/database/migrations/` via goose. Do not use `AutoMigrate` for schema changes. goose is invoked via `go run github.com/pressly/goose/v3/cmd/goose@vX.Y.Z` and is intentionally not an app runtime dependency.

JSONB fields in row models use `pkg/dbtypes.JSONB`; do not drop to raw SQL/`Exec` when GORM `Create` works.

## Per-Handler Auth/Tenant/Permission Pattern

There is **no shared auth/tenant/permission middleware**. Each HTTP handler implements its own gate methods on the `Handler` receiver — `requireSession`, `requireTenantContext`, and `requireWorkspacePermission(perm)` — and applies them per route. The pattern is duplicated across `workspacehttp`, `projecthttp`, and `taskhttp` with the same shape:

```go
group.Get("/path",
    h.requireSession,
    h.requireTenantContext,
    h.requireWorkspacePermission(workspaceperm.PermissionWorkspaceView),
    h.Handler)
```

Each handler caches the resolved account and tenant context on `c.Locals(...)` using handler-scoped keys (e.g. `task.account`, `task.tenant_context`). Tenant resolution reads the `X-Workspace-Slug` header, calls `workspacesvc.ResolveTenantContext`, and rejects the request if the header is missing or the user has no membership.

When adding a new module that needs auth/tenant/permission, copy this pattern verbatim rather than trying to extract a shared middleware. If a refactor to shared middleware is ever proposed, do not start without explicit user approval — it touches every route.

## Auth Model

Session-based Auth with Redis session store and httpOnly cookie — locked decision, do not introduce JWT as the primary auth strategy. Raw session tokens are never returned in response bodies and never stored plaintext; only the hash is persisted. Session secret hashes the token before lookup.

Account is split from login methods: `user_accounts` is the account owner, `auth_identities` stores credentials per provider (currently `email_password/email`; OAuth providers later use `provider + provider_user_id`). Do not treat email as the only identity source.

Password hashing only through `pkg/passwordhash` (bcrypt). Verification and reset tokens use `pkg/securetoken` (random token → hash stored). Verification/reset email sends are rate-limited per IP via Redis.

`POST /api/v1/auth/logout` revokes only the current session (multi-device stays alive); `POST /api/v1/auth/logout-all` revokes all sessions for the account. `POST /api/v1/auth/forgot-password` returns generic success regardless of whether the email exists. `POST /api/v1/auth/reset-password` revokes all active sessions for the account.

## Multi-Tenant Model

Shared database + `tenant_id` column on every workspace/project-scoped row. **`tenant_id` is resolved only by backend Tenant Context** — frontend must not send it, and workspace API URLs must not include it. Every workspace/project query must filter by `tenant_id`. Workspace identity in URLs is the slug; the frontend sends `X-Workspace-Slug` and the backend resolves it to `tenant_id`/`workspace_id` per request.

Permission Guard sits at `internal/modules/workspace/workspaceperm` and gates sensitive actions. `GET /api/v1/workspaces/current` uses `workspace.view`; project and task routes all pass through Permission Guard. Adding a sensitive action without a permission check is a defect.

## API Response Envelope and Route Prefixes

Locked format — handlers go through `presenter` and never write raw envelopes:

- Success: `{ "data": ... }`
- Error: `{ "error": { "code", "message", "details?" } }`
- List: `{ "data": { "items": [...], "pagination": {...} } }`
- API prefix: `/api/v1`
- Every list endpoint must paginate (use `presenter.ParseOffsetQuery` + `presenter.NewOffsetPagination`; default limit 10, max 100).

Route groups in the running code are inconsistent on plural vs singular and this is intentional — match the existing pattern, do not "fix" it without approval:

- Auth: `/api/v1/auth/...`
- Workspace-level: `/api/v1/workspaces/...` (plural) — `check-slug`, `me`, `current`, `register`
- Project- and task-level: `/api/v1/workspace/projects/...` (singular) — projects, project members/positions, tasks and all task sub-resources

## Files / Object Storage

Files use MinIO via presigned PUT/GET URLs. Database stores object key + metadata only — never a full public URL. Upload flow for task attachments: create metadata as `pending` → client uploads directly to MinIO via presigned PUT → `PATCH .../complete` marks `uploaded` → download via presigned GET only when `uploaded` → delete is soft delete on metadata (object retention/cleanup is a future concern). All file access goes through a backend permission check before any signed URL is returned.

## Module Conventions

Package naming (must not collide): domain package is `project`, service is `projectsvc`, repository impl is `projectrepo`, HTTP transport is `projecthttp`. Repeat this pattern for each new module — and copy the per-handler auth/tenant/permission methods from an existing module (see section above).

When a new endpoint ships, add a runnable example to `docs/18-api-test-examples.md`, or split into a new `docs/18.x-{feature}-test-examples.md` if the section is long.

Task activity entries are written in the same DB transaction as the task create/update/status/delete that produced them — do not write activity asynchronously or out of transaction. Comments, checklist items, attachments, tags, and relations all use soft delete.

## Frontend

`app-web/` is the planned Next.js + React + TypeScript frontend (App Router with `(public)/(owner)/(admin)/(workspace)` route groups). The directory is currently empty — frontend work is not part of the active phase.

## Git / Workflow

Working branch is `dev`; merge to `main` manually after review. The user has authorized committing and pushing completed logical chunks to `dev` unless instructed otherwise — still confirm before pushing if the change touches architecture, locked decisions, or anything risky.

## Scope Guard

MVP-0 is the only active phase. Do **not** start Finance, Deliverable full workflow, Notification engine, Reports, Billing, custom domain, SSO, mobile, AI assistant, e-signature, OCR, or advanced approval workflows. Do not change architecture, API format, tenant rule, file security rule, auth strategy, deployment proxy, app folder names, or MVP phase without documenting the reason first (see "Important Rule" in `README.md`).
