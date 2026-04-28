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
- app-api/database/migrations/000004_create_auth_identities.sql
- app-api/database/migrations/000005_create_workspace_core_tables.sql
- app-api/cmd/api/main.go
- app-api/internal/config/config.go
- app-api/internal/bootstrap/app.go
- app-api/internal/bootstrap/db.go
- app-api/internal/bootstrap/http.go
- app-api/internal/bootstrap/redis.go
- app-api/internal/bootstrap/storage.go
- app-api/internal/adapters/cache/redis/ratelimit/limiter.go
- app-api/internal/adapters/cache/redis/sessionstore/store.go
- app-api/internal/adapters/email/smtpemail/sender.go
- app-api/internal/transport/http/router.go
- app-api/internal/transport/http/authhttp/auth_handler.go
- app-api/internal/transport/http/authhttp/auth_handler_test.go
- app-api/internal/transport/http/workspacehttp/workspace_handler.go
- app-api/internal/transport/http/workspacehttp/workspace_handler_test.go
- app-api/internal/transport/http/health/health_handler.go
- app-api/internal/transport/http/presenter/presenter.go
- app-api/internal/transport/http/middlewares/error_handler.go
- app-api/internal/transport/http/middlewares/request_id.go
- app-api/internal/modules/auth/auth_entity.go
- app-api/internal/modules/auth/auth_repository.go
- app-api/internal/modules/auth/authsvc/auth_service.go
- app-api/internal/modules/auth/authsvc/auth_service_test.go
- app-api/internal/modules/workspace/workspace_entity.go
- app-api/internal/modules/workspace/workspace_repository.go
- app-api/internal/modules/workspace/workspacesvc/workspace_service.go
- app-api/internal/modules/workspace/workspacesvc/workspace_service_test.go
- app-api/internal/adapters/database/postgres/authrepo/auth_repository.go
- app-api/internal/adapters/database/postgres/authrepo/auth_repository_integration_test.go
- app-api/internal/adapters/database/postgres/workspacerepo/workspace_repository.go
- app-api/internal/adapters/database/postgres/workspacerepo/workspace_repository_integration_test.go
- app-api/internal/bootstrap/http_test.go
- app-api/pkg/ids/ids.go
- app-api/pkg/ids/ids_test.go
- app-api/pkg/dbtypes/jsonb.go
- app-api/pkg/dbtypes/jsonb_test.go
- app-api/pkg/passwordhash/bcrypt.go
- app-api/pkg/passwordhash/bcrypt_test.go
- app-api/pkg/securetoken/token.go
- app-api/pkg/securetoken/token_test.go

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
- ปรับ migration `000003_drop_auth_uuid_v4_defaults.sql` ทั้ง Up/Down ไม่ให้พา UUID v4 default กลับมาใน dev rollback/redo flow
- เพิ่มและ apply migration `000004_create_auth_identities.sql` เพื่อแยก account owner (`user_accounts`) ออกจาก login methods (`auth_identities`)
- เพิ่มและ apply migration `000005_create_workspace_core_tables.sql` สำหรับ `workspaces` และ `workspace_memberships`
- ปรับ fresh install migration `000002_create_auth_core_tables.sql` ให้มี `auth_identities` ตั้งแต่แรก
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
- Workspace core tables ที่ยืนยันแล้ว:
  - `workspaces`
  - `workspace_memberships`
- เพิ่ม Auth module skeleton แล้ว:
  - domain entity
  - repository interface
  - service constructor
  - postgres adapter placeholder
  - HTTP handler/routes placeholder
- Auth register route เริ่มใช้งานจริงแล้ว:
  - `POST /api/v1/auth/register`
- Auth forgot/reset password route เริ่มใช้งานจริงแล้ว:
  - `POST /api/v1/auth/forgot-password`
  - `POST /api/v1/auth/reset-password`
- Auth postgres repository implementation เริ่มแล้ว:
  - `FindUserAccountByEmail`
  - `CreateUserAccount` โดยสร้าง primary key เป็น UUID v7 จาก Go application, set status default และ set `created_at`/`updated_at` ใน app
  - `CreateAuthIdentity` โดยสร้าง primary key เป็น UUID v7 จาก Go application และ default เป็น `email_password/email`
  - `CreateAuthSession` โดยสร้าง primary key เป็น UUID v7 จาก Go application, set `created_at` ใน app และบังคับ `expires_at`
  - `CreateLoginAttempt` โดยสร้าง primary key เป็น UUID v7 จาก Go application และ set `created_at` ใน app หาก caller ไม่ส่งมา
  - `CreateSecurityEvent` โดยสร้าง primary key เป็น UUID v7 จาก Go application และใช้ `pkg/dbtypes.JSONB` สำหรับ `metadata_json`
  - `FindUserAccountByID`
  - `FindActiveAuthSessionByHash`
  - `RevokeAuthSessionByHash`
- Auth repository integration test ใช้ `PRASANKIT_TEST_DB_DSN` และผ่านกับ Docker PostgreSQL แล้ว
- ทดสอบ fresh install migration กับ database ใหม่ `prasankit_identity_install_check` แล้วผ่านถึง version 4
- ตรวจ fresh install schema แล้ว auth primary key defaults เป็น `<null>` ทั้งหมด
- ตรวจ fresh install schema แล้ว `user_accounts` ใช้ `primary_email` และมี `auth_identities`
- รัน Auth repository integration test กับ fresh install database แล้วผ่าน
- รัน `goose-redo` สำหรับ `000003_drop_auth_uuid_v4_defaults.sql` แล้วตรวจ primary key defaults ยังเป็น `<null>` ทั้งหมด
- เพิ่ม `pkg/ids` เป็น UUID v7 generator กลางสำหรับ primary keys
- เพิ่ม `pkg/dbtypes.JSONB` เป็น JSONB type กลางสำหรับ GORM row model เพื่อลดการใช้ raw SQL/Exec แบบเฉพาะกิจ
- เพิ่ม `pkg/passwordhash` เป็น bcrypt password hasher กลาง โดยใช้ cost default 12
- เพิ่ม `pkg/securetoken` สำหรับสร้าง token ลับและเก็บเฉพาะ hash ใน database
- เพิ่ม SMTP email sender adapter สำหรับส่ง verification email จริง
- เพิ่ม Redis rate limiter adapter สำหรับคุมจำนวน verification email ต่อ IP
- เพิ่ม transaction boundary ใน Auth repository interface และ PostgreSQL adapter ผ่าน `WithinTransaction`
- เพิ่มการ map PostgreSQL unique violation ของ `auth_identities` เป็น `auth.ErrEmailAlreadyRegistered` เพื่อกัน race ตอนสมัคร email ซ้ำ
- เริ่ม Auth service flow สำหรับ email/password registration แล้ว:
  - normalize email
  - validate password length ขั้นต่ำ
  - reject duplicate email
  - rate limit verification email ต่อ IP
  - hash password ด้วย bcrypt
  - create `user_accounts`
  - create `auth_identities` แบบ `email_password/email`
  - create `auth_email_verification_tokens`
  - send verification email ผ่าน SMTP จริง
  - create `security_events` สำหรับ `auth.account_registered`
  - create `security_events` สำหรับ `auth.email_verification_sent`
- เพิ่ม unit test สำหรับ Auth registration service แล้ว
- เพิ่ม HTTP handler สำหรับ `POST /api/v1/auth/register` แล้ว โดย response สำเร็จเป็น `201 Created` ใต้ root key `data`
- เพิ่ม HTTP handler สำหรับ `POST /api/v1/auth/verify-email` แล้ว โดย response สำเร็จอยู่ใต้ root key `data`
- เพิ่ม HTTP handler สำหรับ `POST /api/v1/auth/resend-verification-email` แล้ว โดย response เป็น generic success ใต้ root key `data`
- เพิ่ม HTTP handler สำหรับ `POST /api/v1/auth/login` แล้ว โดย set httpOnly cookie และ response อยู่ใต้ root key `data`
- เพิ่ม HTTP handler สำหรับ `GET /api/v1/auth/me` แล้ว โดยตรวจ Redis session cookie และ response อยู่ใต้ root key `data`
- เพิ่ม HTTP handler สำหรับ `POST /api/v1/auth/logout` แล้ว โดย revoke current session, ลบ Redis session และ clear cookie
- เพิ่ม HTTP handler สำหรับ `POST /api/v1/auth/logout-all` แล้ว โดย revoke session ทุกอุปกรณ์ของ account และ clear cookie ปัจจุบัน
- เพิ่ม HTTP handler สำหรับ `POST /api/v1/auth/forgot-password` แล้ว โดยตอบ generic success เพื่อไม่เปิดเผย account existence
- เพิ่ม HTTP handler สำหรับ `POST /api/v1/auth/reset-password` แล้ว โดยเปลี่ยน password และ revoke active sessions ของ account
- เพิ่ม Redis session store adapter แล้ว เพื่อเก็บ session record จาก login
- เพิ่ม Redis session store read/delete แล้ว เพื่อใช้ validate session และ cleanup session หมดอายุ
- เพิ่ม Auth service flow สำหรับ email/password login แล้ว:
  - normalize email
  - ตรวจ password ด้วย bcrypt
  - require email verified
  - require account status `active`
  - สร้าง session token แบบ random
  - hash session token ด้วย `SESSION_SECRET`
  - เก็บ session metadata ใน `auth_sessions`
  - เก็บ session record ใน Redis
  - เขียน `auth_login_attempts`
  - เขียน `security_events` สำหรับ `auth.login_success` และ `auth.login_failed`
- เพิ่ม Auth service flow สำหรับ current account/session validation แล้ว:
  - require session token จาก cookie
  - hash session token ด้วย `SESSION_SECRET`
  - อ่าน session record จาก Redis
  - reject missing/invalid/expired session
  - โหลด `user_accounts` จาก database
  - require account status `active`
- เพิ่ม Auth service flow สำหรับ logout current session แล้ว:
  - require session token จาก cookie
  - hash session token ด้วย `SESSION_SECRET`
  - อ่าน session record จาก Redis
  - update `auth_sessions.status = revoked`
  - เขียน `security_events` สำหรับ `auth.logout`
  - ลบ session record จาก Redis
- เพิ่ม Auth service flow สำหรับ forgot/reset password แล้ว:
  - rate limit reset email ต่อ IP และต่อ email
  - สร้าง password reset token และเก็บเฉพาะ hash
  - ส่ง reset email ผ่าน SMTP จริงไปที่ frontend URL
  - reset password ด้วย token ที่ active และยังไม่หมดอายุ
  - mark token เป็น used
  - update password hash
  - revoke active sessions ทั้งหมดของ account ด้วย reason `password_reset`
  - ลบ Redis session records ของ active sessions ที่ถูก revoke
  - เขียน `security_events` สำหรับ `auth.password_reset_requested` และ `auth.password_reset_success`
- เพิ่ม test สำหรับ Auth register handler แล้ว
- เพิ่ม test สำหรับ Auth verify-email handler แล้ว
- เพิ่ม test สำหรับ Auth resend-verification-email handler แล้ว
- `GOTOOLCHAIN=auto go test ./...` ผ่านใน app-api หลัง wire register handler
- Auth repository integration test ผ่านกับ Docker PostgreSQL หลังเพิ่ม duplicate email mapping
- `docker compose up -d --build prasankit-api` ผ่านหลัง wire register handler
- Docker smoke test `POST /api/v1/auth/register` ผ่าน ได้ `201 Created`
- Docker smoke test สมัคร email ซ้ำผ่าน ได้ `409 Conflict` และ error code `EMAIL_ALREADY_REGISTERED`
- เพิ่ม `docs/18-api-test-examples.md` เป็นเอกสารตัวอย่างทดสอบ endpoint ที่เสร็จจริง
- เพิ่ม `docs/18.0-auth-flow-status.md` เพื่อบันทึกว่า Auth backend ทำถึงไหน อะไรยังเป็น manual/temporary และต้องกลับมาแก้ตรงไหนเมื่อเริ่ม frontend
- เพิ่ม `docs/18.1-auth-register-test-examples.md` และอัปเดตตัวอย่าง register ให้รวม verification email/rate limit/SMTP failure แล้ว
- เพิ่ม `docs/18.2-auth-verify-email-test-examples.md` สำหรับตัวอย่างทดสอบ verify email แล้ว
- เพิ่ม `docs/18.3-auth-resend-verification-email-test-examples.md` สำหรับตัวอย่างทดสอบ resend verification email แล้ว
- เพิ่ม `docs/18.4-auth-login-test-examples.md` สำหรับตัวอย่างทดสอบ login แล้ว
- เพิ่ม `docs/18.5-auth-me-test-examples.md` สำหรับตัวอย่างทดสอบ current session/me แล้ว
- เพิ่ม `docs/18.6-auth-logout-test-examples.md` สำหรับตัวอย่างทดสอบ logout แล้ว
- เพิ่ม `docs/18.7-auth-logout-all-test-examples.md` สำหรับตัวอย่างทดสอบ logout-all แล้ว
- เพิ่ม `docs/18.8-auth-password-reset-test-examples.md` สำหรับตัวอย่างทดสอบ forgot/reset password แล้ว
- เพิ่ม `docs/18.9-workspace-registration-test-examples.md` สำหรับตัวอย่างทดสอบ check slug/register workspace แล้ว
- เพิ่ม Workspace module ก้อนแรกแล้ว:
  - `GET /api/v1/workspaces/check-slug`
  - `POST /api/v1/workspaces/register`
  - สร้าง `tenant_id` จาก backend เอง
  - สร้าง owner membership จาก account ใน session
  - response ไม่ส่ง `tenant_id` ให้ frontend ใช้คุมสิทธิ์

Known note:

- ถ้ารัน Compose โดยไม่มี `.env` ค่า `${REDIS_PASSWORD}` จะว่าง ทำให้ Redis start fail ได้
- ตอนนี้แก้ด้วย `make env-init` เพื่อสร้าง `.env` local ที่ถูก `.gitignore`
- `.env.example` เป็น template เท่านั้น ส่วน runtime จริงให้ใช้ `.env` ผ่าน `make env-init`
- หลังเพิ่ม SMTP จริง ต้องเติม `MAIL_*` env ใน `.env` ก่อน rebuild/start API ไม่เช่นนั้น config จะ fail-fast

Composition note:

- `app-api/internal/bootstrap` เป็น composition root ของ backend
- ช่วง Foundation ให้เล็กไว้ก่อน ไม่สร้าง placeholder repo/service/handler ก่อนเริ่ม module จริง
- เมื่อเริ่ม module ให้ wire ตามลำดับ `config -> infrastructure clients -> repositories -> services/use cases -> HTTP handlers -> router`
- `App` ต้องถือ resource อายุยาวที่ต้องปิด เช่น PostgreSQL, Redis, MinIO client และ Fiber server
- ทุก resource ที่ `App` ถือ ต้องปิดใน graceful shutdown
- ใช้ GORM สำหรับ query/ORM เท่านั้น ห้ามใช้ AutoMigrate เปลี่ยน schema production โดยไม่มี decision ใหม่
- Schema ใช้ SQL migration ผ่าน goose; goose CLI ใช้ผ่าน `go run ...@v3.27.1` และไม่เพิ่มเป็น runtime dependency ของ app
- Primary key strategy: ใช้ PostgreSQL `UUID` columns แต่ Go application ต้องสร้าง UUID v7 ก่อน insert; ห้ามใช้ `gen_random_uuid()` เป็น default primary key ใน table ใหม่
- JSONB strategy: ใช้ `pkg/dbtypes.JSONB` ใน repository row model เมื่อ field เป็น PostgreSQL `JSONB`
- Timestamp strategy: repository/service ต้อง set timestamp สำคัญใน Go ก่อน insert ไม่พึ่ง DB default เป็น behavior หลัก
- Auth identity strategy: `user_accounts` เป็น account กลาง ส่วน `auth_identities` เป็นช่องทาง login; social login ในอนาคตต้องใช้ `provider + provider_user_id`
- Password hashing strategy: ใช้ bcrypt ผ่าน `pkg/passwordhash` เป็น adapter กลาง ไม่ hash password ใน handler หรือ repository โดยตรง
- Email verification strategy: ส่งผ่าน SMTP จริงจาก env, เก็บเฉพาะ token hash ใน `auth_email_verification_tokens`, และคุมจำนวนส่งด้วย Redis rate limit
- Completed API documentation strategy: เมื่อ endpoint ไหน implement เสร็จจริง ต้องเพิ่ม request/success/error examples ใน `docs/18-api-test-examples.md`
- Fresh server bootstrap order: create/edit `.env` -> start PostgreSQL/Redis/MinIO -> run goose migrations -> start/rebuild API
- Auth ยังล็อกเป็น Session-based Auth + Redis + httpOnly Cookie ไม่ใช้ JWT เป็น auth หลัก
- Auth/session backend หลักเสร็จถึง forgot/reset password แล้ว
- Workspace registration backend ก้อนแรกเสร็จถึง check slug/register workspace แล้ว

หมายเหตุ:

เครื่อง local ตอนนี้เป็น Go 1.23.5 แต่ backend build ผ่าน Docker ด้วย Go 1.25 เพราะ Fiber v3 ต้องใช้ Go 1.25+ เป็นอย่างน้อย ระยะยาวยังสามารถขยับ Dockerfile ไป Go 1.26.x ตาม baseline ได้

## Next Step

ขั้นถัดไปเริ่ม `GET /api/v1/workspaces/me` และ Tenant Context resolver แบบช้า ๆ โดยยังต้องรักษา Tenant Context และ Permission Guard ตาม rule เดิม

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
-> Migration 000004_create_auth_identities applied
-> Migration 000005_create_workspace_core_tables applied
-> Auth module skeleton created
-> Auth repository: FindUserAccountByID/FindUserAccountByEmail/CreateUserAccount/CreateAuthIdentity/CreateAuthSession/CreateLoginAttempt/CreateSecurityEvent implemented
-> Auth service: RegisterEmailPassword implemented
-> Auth HTTP: POST /api/v1/auth/register wired
-> Auth HTTP: POST /api/v1/auth/verify-email wired
-> Auth HTTP: POST /api/v1/auth/resend-verification-email wired
-> Auth service: LoginEmailPassword implemented
-> Auth HTTP: POST /api/v1/auth/login wired
-> Auth session store: Redis session record implemented
-> Auth service: CurrentAccount/session validation implemented
-> Auth HTTP: GET /api/v1/auth/me wired
-> Auth service: LogoutCurrentSession implemented
-> Auth HTTP: POST /api/v1/auth/logout wired
-> Auth service: LogoutAllSessions implemented
-> Auth HTTP: POST /api/v1/auth/logout-all wired
-> Auth service: ForgotPassword/ResetPassword implemented
-> Auth HTTP: POST /api/v1/auth/forgot-password wired
-> Auth HTTP: POST /api/v1/auth/reset-password wired
-> Workspace HTTP: GET /api/v1/workspaces/check-slug wired
-> Workspace HTTP: POST /api/v1/workspaces/register wired
```
