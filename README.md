# Prasankit

**Multi-Tenant Project Management Platform** สำหรับทีม/องค์กรที่ทำงานแบบโครงการ — รองรับทั้งงานภายในและงานลูกค้า/งานรับจ้างในระบบเดียว. wedge ที่ต่างจาก Jira/Notion คือ **งวดงาน + ตรวจรับ** และ **การเงินโครงการ** อยู่ที่เดียวกัน (ดู [docs/01-vision.md](docs/01-vision.md)).

> **เอกสาร = source of truth.** ก่อนเริ่มทำงานอ่าน [docs/99-handoff.md](docs/99-handoff.md) (สถานะล่าสุด + ก้าวต่อไป) และ [docs/00-workflow.md](docs/00-workflow.md) (วิธีทำงานร่วมกัน / gate). README นี้คุมแค่ "วิธีรันโปรเจกต์" — ไม่ใช่ design.

## Stack

| ฝั่ง | เทคโนโลยี |
| --- | --- |
| Backend (`app-api/`) | Go 1.26 + Fiber v3, hexagonal (ports & adapters), PostgreSQL 18 (pgx/GORM), Redis 8 (session), MinIO (storage), Mailpit (dev mail), goose (migration) |
| Frontend (`app-web/`) | Next.js (App Router) + TypeScript + Tailwind, pnpm, Vitest |
| Infra | docker-compose (api / web / pgsql / redis / minio / mailpit) |

รายละเอียด architecture: [docs/02-architecture.md](docs/02-architecture.md) · schema: [docs/04-data-model.md](docs/04-data-model.md)

## Prerequisites

- **Docker + Docker Compose v2** — รันทุก service (api/web/infra) ใน container
- **Go** — สำหรับ `make test` และ goose CLI ที่รันบน host (`GOTOOLCHAIN=auto` จะ pull toolchain ให้ แต่ต้องมี go ฐานก่อน)
- **pnpm** — เฉพาะตอน dev FE นอก container (`pnpm dev`); ปกติ web รันใน container อยู่แล้ว

## Quickstart

```bash
make env-init     # คัดลอก .env.example -> .env (แก้พอร์ตได้ถ้าชนเครื่อง)
make dev-up       # infra (pg/redis/minio/mailpit) -> migrate -> build+run api -> web -> health
```

เปิด:
- Web — http://localhost:13000
- API — http://localhost:18080 (health: `GET /api/v1/health/ready`)
- Mailpit (ดูอีเมล dev) — http://localhost:18025

ปิด: `make down` · ดู logs api: `make logs-api` · rebuild เฉพาะ api/web: `make api-up` / `make web-up`

### Host ports (default จาก `.env.example`)

| Service | Host port | หมายเหตุ |
| --- | --- | --- |
| API | `18080` | container listen 8080 |
| Web | `13000` | container listen 3000 · CORS อนุญาตเฉพาะ origin นี้ |
| PostgreSQL | `15432` | |
| Redis | `16379` | |
| MinIO | `19000` (API) / `19001` (console) | |
| Mailpit | `11025` (SMTP) / `18025` (Web UI) | |

> พอร์ตเป็น high-port กันชน (D23). ถ้าชนเครื่อง dev แก้ใน `.env` (`*_EXTERNAL_PORT`) — ค่าใน `.env` override ทั้ง docker-compose และ Makefile.

## Tests

```bash
cd app-api && make test    # go test -p 1 ./...  (serialize: integration tests แชร์ pg/redis เดียว)
cd app-api && make seed    # กู้ dev login fixtures (จำเป็นหลัง test — ดูคำเตือนล่าง)
```

> ⚠️ **`make test` ล้าง dev DB** — integration tests ใช้ Postgres ตัวเดียวกับแอปแล้ว `TRUNCATE` ตอน teardown → ข้อมูล dev (users / workspace / projects) หายทุกครั้ง. รัน `make seed` ตามเพื่อกู้คืน.
> ใช้ `make test` เสมอ — `go test ./...` เปล่า ๆ จะ flake (parallel packages ตีกันบน DB ที่แชร์).

**Dev login fixtures** (หลัง `make seed`): `owner@prasankit.local` (owner) + `bob@prasankit.local` (user) บน workspace `prasankit` · รหัส `demopass123`.

## Migrations (goose)

```bash
cd app-api
make db-migrate     # apply ทั้งหมด (รวมใน make dev-up อยู่แล้ว)
make goose-status   # ดูสถานะ
make goose-down     # ถอย 1 step
```

> Makefile ของ `app-api` ใช้ `GOOSE_DSN` default พอร์ต `15432`. ถ้า `.env` ตั้งพอร์ต Postgres อื่น ให้ override: `make goose-up GOOSE_DSN="postgres://prasankit:change_me@localhost:<port>/prasankit?sslmode=disable"` (หรือใช้ `make db-migrate` จาก root ที่ derive พอร์ตจาก `.env` ให้อัตโนมัติ).

## Frontend dev (`pnpm dev`)

web รันใน container ผ่าน `make dev-up` อยู่แล้ว. ถ้าจะรัน FE นอก container:

```bash
cd app-web
cp .env.example .env.local    # ตั้ง NEXT_PUBLIC_API_URL=http://localhost:18080
pnpm install && pnpm dev
```

> ⚠️ `pnpm dev` รันที่พอร์ต `3000` ซึ่ง **ไม่อยู่ใน `API_CORS_ALLOWED_ORIGINS`** (อนุญาตแค่ `:13000`) → request ไป API จะโดน CORS block. ใช้ผ่าน container (`:13000`) หรือเพิ่ม `:3000` เข้า `API_CORS_ALLOWED_ORIGINS` ใน `.env` ตอน dev.

## Repo layout

```
app-api/        Go backend (hexagonal)
  cmd/{api,seed}              entrypoint + dev seeder
  internal/
    modules/                 domain + service (auth, workspace, project, ...)
    adapters/                driven adapters (database, cache, email, storage)
    transport/http/          driving adapter (handlers, middlewares, presenter)
    bootstrap/               composition root (wiring)
  database/migrations/       goose SQL migrations
app-web/        Next.js frontend (App Router, Tailwind, pnpm)
docs/           เอกสาร design (source of truth) + DECISIONS + handoff
docker-compose.yml · Makefile · .env.example
```

## เอกสาร

| ไฟล์ | เนื้อหา |
| --- | --- |
| [docs/99-handoff.md](docs/99-handoff.md) | **อ่านก่อน** — อยู่ตรงไหน / ทำอะไรต่อ (living doc) |
| [docs/00-workflow.md](docs/00-workflow.md) | role / loop / gate (contract วิธีทำงาน) |
| [docs/01-vision.md](docs/01-vision.md) | vision / scope / north star |
| [docs/02-architecture.md](docs/02-architecture.md) | backend architecture / tenant model / auth |
| [docs/03-build-plan.md](docs/03-build-plan.md) | scope first cut + ลำดับ milestone |
| [docs/04-data-model.md](docs/04-data-model.md) | schema จริงต่อ milestone (M1–M2) |
| [docs/DECISIONS.md](docs/DECISIONS.md) | append-only decision log (D1–D57) |
