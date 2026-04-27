# Foundation Progress

เอกสารนี้ใช้บอกว่าตอนนี้ Foundation ทำถึงขั้นไหนแล้ว

## Completed

Foundation Step 1: Local Infrastructure

สถานะ: Done

สร้างแล้ว:

- Makefile
- .env.example
- docker-compose.yml

Services ที่มีใน docker-compose ตอนนี้:

- prasankit-pgsql
- prasankit-redis
- prasankit-minio

ใส่แล้ว:

- prasankit-api

ยังไม่ใส่:

- prasankit-web
- prasankit-proxy

เหตุผล:

app-api/ มี Dockerfile แล้ว ส่วน app-web/ ยังไม่มี Dockerfile จึงยังไม่ใส่ web service ใน docker-compose

## Verified

ตรวจ `make compose-config` ผ่านแล้ว โดยใช้ `.env` เป็น runtime env

## Current Step

Foundation Step 2: Backend Skeleton

สถานะ: Done เฉพาะ backend skeleton และ health split

สร้างแล้ว:

- app-api/go.mod
- app-api/go.sum
- app-api/Dockerfile
- app-api/Makefile
- app-api/database/README.md
- app-api/database/migrations/000001_enable_extensions.sql
- app-api/database/migrations/000002_create_auth_core_tables.sql
- app-api/database/migrations/000003_drop_auth_uuid_v4_defaults.sql
- app-api/cmd/api/main.go
- app-api/internal/config/config.go
- app-api/internal/bootstrap/app.go
- app-api/internal/bootstrap/db.go
- app-api/internal/bootstrap/http.go
- app-api/internal/bootstrap/redis.go
- app-api/internal/bootstrap/storage.go
- app-api/internal/transport/http/router.go
- app-api/internal/transport/http/authhttp/auth_handler.go
- app-api/internal/transport/http/health/health_handler.go
- app-api/internal/transport/http/presenter/presenter.go
- app-api/internal/transport/http/middlewares/error_handler.go
- app-api/internal/transport/http/middlewares/request_id.go
- app-api/internal/modules/auth/auth_entity.go
- app-api/internal/modules/auth/auth_repository.go
- app-api/internal/modules/auth/authsvc/auth_service.go
- app-api/internal/adapters/database/postgres/authrepo/auth_repository.go
- app-api/internal/adapters/database/postgres/authrepo/auth_repository_integration_test.go
- app-api/internal/bootstrap/http_test.go
- app-api/pkg/ids/ids.go
- app-api/pkg/ids/ids_test.go

เป้าหมาย:

เปิด health endpoint ให้ได้:

```text
GET /api/v1/health
GET /api/v1/health/live
GET /api/v1/health/ready
```

Response:

```json
{
  "data": {
    "status": "ok"
  }
}
```

Ready response ปัจจุบัน:

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

Verified:

- `go test ./...` ผ่านใน app-api
- `make compose-config` ผ่านหลังเปลี่ยน API healthcheck ไปใช้ `/api/v1/health/live`
- สร้าง `.env` local จาก `.env.example` แล้ว เพื่อให้ `docker compose up -d` โหลดค่า env โดยอัตโนมัติ
- `docker compose up -d` ผ่านหลัง health split
- `docker compose ps` แสดง PostgreSQL, Redis, MinIO และ API เป็น healthy
- `GET /api/v1/health` ตอบ `{"data":{"status":"ok"}}`
- `GET /api/v1/health/live` ตอบ `{"data":{"status":"ok"}}`
- `GET /api/v1/health/ready` ตอบ `{"data":{"checks":{"api":"ok","minio":"ok","postgres":"ok","redis":"ok"},"status":"ok"}}`
- log ของ container แสดง Fiber v3.2.0
- API config fail fast หากไม่มี required env เช่น API_ENV หรือ API_PORT
- API config fail fast สำหรับ PostgreSQL env รวมถึง `POSTGRES_SSL_MODE`
- API config fail fast สำหรับ Redis env รวมถึง `REDIS_DB`
- API config fail fast สำหรับ Storage/MinIO env รวมถึง `STORAGE_ENDPOINT`, `STORAGE_PUBLIC_ENDPOINT`, `STORAGE_BUCKET`, `STORAGE_PRESIGN_TTL`
- app-api ต่อ PostgreSQL ผ่าน GORM v2 + pgx แล้ว
- app-api ต่อ Redis ผ่าน go-redis v9 แล้ว
- app-api สร้าง MinIO client ผ่าน minio-go v7 แล้ว
- presenter มี RenderItem, RenderList, RenderError และ OffsetPagination เบื้องต้นตาม response standard
- เพิ่ม app runner พร้อม graceful shutdown สำหรับ Fiber app แล้ว
- เพิ่ม basic Fiber error handler ให้ error response อยู่ใต้ root key `error`
- เพิ่ม `/api/v1/health/live` สำหรับ liveness
- เพิ่ม `/api/v1/health/ready` สำหรับ readiness และเช็ก PostgreSQL/Redis/MinIO แล้ว
- เพิ่ม test สำหรับ health endpoints และ error envelope
- เปลี่ยน Docker healthcheck ของ `prasankit-api` ไปใช้ `/api/v1/health/live`
- `docker compose up -d --build prasankit-api` ผ่านหลังเพิ่ม PostgreSQL connection
- `docker compose up -d --build prasankit-api` ผ่านหลังเพิ่ม Redis connection
- `docker compose up -d --build prasankit-api` ผ่านหลังเพิ่ม MinIO client
- เลือก goose สำหรับ SQL migration แล้ว
- เพิ่ม Makefile เฉพาะ goose targets แล้ว รวมถึง `make db-migrate` สำหรับ `goose-up` แล้วตามด้วย `goose-status`
- เพิ่ม root `Makefile` สำหรับ dev bootstrap:
  - `make env-init`
  - `make infra-up`
  - `make db-migrate`
  - `make api-up`
  - `make dev-up`
  - `make health`
- เปลี่ยน `docker-compose.yml` ให้ API container อ่าน runtime env จาก `.env` แทน `.env.example`
- apply migration `000001_enable_extensions.sql` ผ่าน Docker PostgreSQL แล้ว
- apply migration `000002_create_auth_core_tables.sql` ผ่าน Docker PostgreSQL แล้ว
- ปรับ migration `000002_create_auth_core_tables.sql` ให้ fresh install สร้าง auth primary key columns โดยไม่มี database default v4 ตั้งแต่แรก
- apply migration `000003_drop_auth_uuid_v4_defaults.sql` ผ่าน Docker PostgreSQL แล้ว เพื่อถอด default `gen_random_uuid()` ออกจากฐานที่เคย apply migration รุ่นก่อนหน้า
- PostgreSQL extensions ที่ยืนยันแล้ว:
  - `citext`
  - `pgcrypto`
- Auth core tables ที่ยืนยันแล้ว:
  - `user_accounts`
  - `auth_sessions`
  - `auth_login_attempts`
  - `auth_email_verification_tokens`
  - `auth_password_reset_tokens`
  - `security_events`
- เพิ่ม Auth module skeleton แล้ว:
  - domain entity
  - repository interface
  - service constructor
  - postgres adapter placeholder
  - HTTP handler/routes placeholder
- Auth routes ตอบ `501 NOT_IMPLEMENTED` ตามที่ตั้งใจ:
  - `POST /api/v1/auth/register`
  - `POST /api/v1/auth/verify-email`
  - `POST /api/v1/auth/login`
  - `POST /api/v1/auth/logout`
  - `POST /api/v1/auth/logout-all`
  - `POST /api/v1/auth/forgot-password`
  - `POST /api/v1/auth/reset-password`
  - `GET /api/v1/auth/me`
- Auth postgres repository implementation เริ่มแล้ว:
  - `FindUserAccountByEmail`
  - `CreateLoginAttempt` โดยสร้าง primary key เป็น UUID v7 จาก Go application
- Auth repository integration test ใช้ `PRASANKIT_TEST_DB_DSN` และผ่านกับ Docker PostgreSQL แล้ว
- ทดสอบ fresh install migration กับ database ใหม่ `prasankit_install_check` แล้วผ่านถึง version 3
- ตรวจ fresh install schema แล้ว auth primary key defaults เป็น `<null>` ทั้งหมด
- รัน Auth repository integration test กับ fresh install database แล้วผ่าน
- เพิ่ม `pkg/ids` เป็น UUID v7 generator กลางสำหรับ primary keys

Known note:

- ถ้ารัน Compose โดยไม่มี `.env` ค่า `${REDIS_PASSWORD}` จะว่าง ทำให้ Redis start fail ได้
- ตอนนี้แก้ด้วย `make env-init` เพื่อสร้าง `.env` local ที่ถูก `.gitignore`
- `.env.example` เป็น template เท่านั้น ส่วน runtime จริงให้ใช้ `.env` ผ่าน `make env-init`

Composition note:

- `app-api/internal/bootstrap` เป็น composition root ของ backend
- ช่วง Foundation ให้เล็กไว้ก่อน ไม่สร้าง placeholder repo/service/handler ก่อนเริ่ม module จริง
- เมื่อเริ่ม module ให้ wire ตามลำดับ `config -> infrastructure clients -> repositories -> services/use cases -> HTTP handlers -> router`
- `App` ต้องถือ resource อายุยาวที่ต้องปิด เช่น PostgreSQL, Redis, MinIO client และ Fiber server
- ทุก resource ที่ `App` ถือ ต้องปิดใน graceful shutdown
- ใช้ GORM สำหรับ query/ORM เท่านั้น ห้ามใช้ AutoMigrate เปลี่ยน schema production โดยไม่มี decision ใหม่
- Schema ใช้ SQL migration ผ่าน goose; goose CLI ใช้ผ่าน `go run ...@v3.27.1` และไม่เพิ่มเป็น runtime dependency ของ app
- Primary key strategy: ใช้ PostgreSQL `UUID` columns แต่ Go application ต้องสร้าง UUID v7 ก่อน insert; ห้ามใช้ `gen_random_uuid()` เป็น default primary key ใน table ใหม่
- Fresh server bootstrap order: create/edit `.env` -> start PostgreSQL/Redis/MinIO -> run goose migrations -> start/rebuild API
- Auth ยังล็อกเป็น Session-based Auth + Redis + httpOnly Cookie ไม่ใช้ JWT เป็น auth หลัก

หมายเหตุ:

เครื่อง local ตอนนี้เป็น Go 1.23.5 แต่ backend build ผ่าน Docker ด้วย Go 1.25 เพราะ Fiber v3 ต้องใช้ Go 1.25+ เป็นอย่างน้อย ระยะยาวยังสามารถขยับ Dockerfile ไป Go 1.26.x ตาม baseline ได้

## Next Step

Foundation Step 3: Dependency Readiness

เริ่ม wire dependency client แบบช้า ๆ:

```text
PostgreSQL connection done
-> Redis connection done
-> MinIO client done
-> Foundation dependency readiness complete
-> Migration strategy selected: goose + SQL migrations
-> Migration 000001_enable_extensions applied
-> Migration 000002_create_auth_core_tables applied
-> Migration 000003_drop_auth_uuid_v4_defaults applied
-> Auth module skeleton created
-> Auth repository: FindUserAccountByEmail/CreateLoginAttempt implemented
```

ขั้นถัดไปทำ Auth repository implementation ต่อแบบเล็ก ๆ: `CreateSecurityEvent`
