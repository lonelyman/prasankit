# Prasankit

Prasankit is a multi-tenant project/workspace management platform.

## Current Status

MVP-0 backend is in progress. The implementation target is end-to-end MVP-0 before any out-of-scope feature is added.

MVP-0 scope, hard rules, locked decisions, and stack are documented in **[docs/00-context-guard.md](docs/00-context-guard.md)** — read it first.

For current state ("what's built, what's next"), see **[docs/99-current-handoff.md](docs/99-current-handoff.md)**.

## Stack (snapshot)

- Frontend: Next.js 16.x + React 19.x + TypeScript 6.x (`app-web/`, not yet implemented)
- Backend: Go 1.26.x + Fiber v3.x (`app-api/`)
- ORM / Driver: GORM v2, pgx v5
- Database: PostgreSQL 18.x
- Cache / Session: Redis 8.x
- Object Storage: MinIO
- Reverse Proxy: Traefik 3.6.x
- Deployment: Docker Compose on Ubuntu Server 24.04 LTS

Authoritative version baselines live in [docs/00-tech-stack-architecture-baseline.md](docs/00-tech-stack-architecture-baseline.md).

## Local Dev Bootstrap

Create runtime env and start the current backend foundation:

```bash
make env-init
make dev-up
```

This starts PostgreSQL, Redis, and MinIO, runs goose migrations, starts the API, then checks `/api/v1/health/ready`.

For a new server, follow the same order:

```text
env -> infrastructure -> migrations -> API -> health check
```

`.env.example` is a template only. Runtime services read `.env`, which is intentionally ignored by git.

Auth register sends a verification email through real SMTP. Set `MAIL_*` values in `.env` before testing `POST /api/v1/auth/register`.

## Documentation Map

- **Rules and scope (canonical)**: [docs/00-context-guard.md](docs/00-context-guard.md)
- **Stack/version baseline**: [docs/00-tech-stack-architecture-baseline.md](docs/00-tech-stack-architecture-baseline.md)
- **Tenant model**: [docs/06-tenant-data-isolation.md](docs/06-tenant-data-isolation.md)
- **API design (spec)**: [docs/09-api-design.md](docs/09-api-design.md)
- **MVP-0 user flow + build plan**: [docs/14-mvp0-user-flow.md](docs/14-mvp0-user-flow.md), [docs/15-mvp0-build-plan.md](docs/15-mvp0-build-plan.md)
- **Current source tree + module status**: [docs/13-project-source-tree.md](docs/13-project-source-tree.md) (target), code in [app-api/](app-api/) is authoritative for current state
- **API examples for implemented endpoints**: [docs/18-api-test-examples.md](docs/18-api-test-examples.md) + `docs/18.x-*.md`
- **Current handoff (what's built, what's next)**: [docs/99-current-handoff.md](docs/99-current-handoff.md)

Other docs (01, 03–05, 07, 10, 11) describe MVP-1+ design intent and are clearly labelled at the top of each file.

## Important Rule

Do not change architecture, API format, tenant rule, file security rule, auth strategy, deployment proxy, app folder names, or MVP phase without documenting the reason first. See [docs/00-context-guard.md](docs/00-context-guard.md) for the binding rules.
