# API Test Examples

เอกสารนี้ใช้เก็บตัวอย่างการทดสอบ API ที่ implement เสร็จจริงแล้วเท่านั้น

สถานะ flow จริงของ Auth ดูที่ `docs/18.0-auth-flow-status.md`

กติกา:

- Endpoint ที่ยังเป็น `501 NOT_IMPLEMENTED` ไม่ต้องใส่เป็นเส้นที่เสร็จ
- ทุก endpoint ที่เสร็จแล้วต้องมีตัวอย่าง request, success response และ error สำคัญ
- ตัวอย่างต้องรันกับ Docker Compose local ได้
- Base URL local ตอนนี้คือ `http://localhost:8080`
- API ทุกเส้นต้องอยู่ใต้ `/api/v1`
- Success response ต้องอยู่ใต้ root key `data`
- Error response ต้องอยู่ใต้ root key `error`

## Before Testing

เริ่มระบบ local:

```bash
make dev-up
```

หรือถ้า infrastructure รันอยู่แล้วและแก้ API code:

```bash
docker compose up -d --build prasankit-api
```

ตรวจสถานะ container:

```bash
docker compose ps
```

## Completed Endpoints

| Method | Endpoint | Status |
| --- | --- | --- |
| GET | `/api/v1/health` | Done |
| GET | `/api/v1/health/live` | Done |
| GET | `/api/v1/health/ready` | Done |
| POST | `/api/v1/auth/register` | Done, requires SMTP env, see `docs/18.1-auth-register-test-examples.md` |
| POST | `/api/v1/auth/verify-email` | Done, see `docs/18.2-auth-verify-email-test-examples.md` |
| POST | `/api/v1/auth/resend-verification-email` | Done, see `docs/18.3-auth-resend-verification-email-test-examples.md` |
| POST | `/api/v1/auth/login` | Done, sets httpOnly cookie, see `docs/18.4-auth-login-test-examples.md` |
| GET | `/api/v1/auth/me` | Done, validates Redis session cookie, see `docs/18.5-auth-me-test-examples.md` |
| POST | `/api/v1/auth/logout` | Done, revokes current session and clears cookie, see `docs/18.6-auth-logout-test-examples.md` |
| POST | `/api/v1/auth/logout-all` | Done, revokes all account sessions, see `docs/18.7-auth-logout-all-test-examples.md` |
| POST | `/api/v1/auth/forgot-password` | Done, see `docs/18.8-auth-password-reset-test-examples.md` |
| POST | `/api/v1/auth/reset-password` | Done, see `docs/18.8-auth-password-reset-test-examples.md` |
| GET | `/api/v1/workspaces/check-slug` | Done, see `docs/18.9-workspace-registration-test-examples.md` |
| POST | `/api/v1/workspaces/register` | Done, requires session cookie, see `docs/18.9-workspace-registration-test-examples.md` |
| GET | `/api/v1/workspaces/me` | Done, requires session cookie, see `docs/18.9-workspace-registration-test-examples.md` |
| GET | `/api/v1/workspaces/current` | Done, resolves Tenant Context from session + `X-Workspace-Slug`, see `docs/18.10-workspace-current-test-examples.md` |
| GET | `/api/v1/workspace/projects` | Done, requires session cookie + `X-Workspace-Slug`, see `docs/18.11-project-foundation-test-examples.md` |
| POST | `/api/v1/workspace/projects` | Done, requires session cookie + `X-Workspace-Slug` + `workspace.manage`, see `docs/18.11-project-foundation-test-examples.md` |
| GET | `/api/v1/workspace/projects/{project_id}` | Done, requires session cookie + `X-Workspace-Slug`, see `docs/18.11-project-foundation-test-examples.md` |
| PATCH | `/api/v1/workspace/projects/{project_id}` | Done, requires session cookie + `X-Workspace-Slug` + `workspace.manage`, see `docs/18.11-project-foundation-test-examples.md` |
| GET | `/api/v1/workspace/projects/{project_id}/members` | Done, requires session cookie + `X-Workspace-Slug`, see `docs/18.12-project-team-test-examples.md` |
| POST | `/api/v1/workspace/projects/{project_id}/members` | Done, requires session cookie + `X-Workspace-Slug` + `workspace.manage`, see `docs/18.12-project-team-test-examples.md` |
| GET | `/api/v1/workspace/projects/{project_id}/tasks` | Done, requires session cookie + `X-Workspace-Slug`, see `docs/18.13-task-foundation-test-examples.md` |
| GET | `/api/v1/workspace/projects/{project_id}/tasks/summary` | Done, requires session cookie + `X-Workspace-Slug`, see `docs/18.13-task-foundation-test-examples.md` |
| POST | `/api/v1/workspace/projects/{project_id}/tasks` | Done, requires session cookie + `X-Workspace-Slug` + `workspace.manage`, see `docs/18.13-task-foundation-test-examples.md` |
| GET | `/api/v1/workspace/projects/{project_id}/tasks/{task_id}` | Done, requires session cookie + `X-Workspace-Slug`, see `docs/18.13-task-foundation-test-examples.md` |
| PATCH | `/api/v1/workspace/projects/{project_id}/tasks/{task_id}` | Done, requires session cookie + `X-Workspace-Slug` + `workspace.manage`, see `docs/18.13-task-foundation-test-examples.md` |
| PATCH | `/api/v1/workspace/projects/{project_id}/tasks/{task_id}/status` | Done, requires session cookie + `X-Workspace-Slug` + `workspace.manage`, see `docs/18.13-task-foundation-test-examples.md` |
| DELETE | `/api/v1/workspace/projects/{project_id}/tasks/{task_id}` | Done, requires session cookie + `X-Workspace-Slug` + `workspace.manage`, see `docs/18.13-task-foundation-test-examples.md` |

## Health

### GET /api/v1/health

ใช้เช็ก health แบบ legacy/simple

Request:

```bash
curl -i http://localhost:8080/api/v1/health
```

Success response:

```http
HTTP/1.1 200 OK
```

```json
{
  "data": {
    "status": "ok"
  }
}
```

### GET /api/v1/health/live

ใช้เป็น liveness check ของ API container

Request:

```bash
curl -i http://localhost:8080/api/v1/health/live
```

Success response:

```http
HTTP/1.1 200 OK
```

```json
{
  "data": {
    "status": "ok"
  }
}
```

### GET /api/v1/health/ready

ใช้เป็น readiness check โดยตรวจ API process, PostgreSQL, Redis และ MinIO

Request:

```bash
curl -i http://localhost:8080/api/v1/health/ready
```

Success response:

```http
HTTP/1.1 200 OK
```

```json
{
  "data": {
    "checks": {
      "api": "ok",
      "minio": "ok",
      "postgres": "ok",
      "redis": "ok"
    },
    "status": "ok"
  }
}
```

Service unavailable response:

```http
HTTP/1.1 503 Service Unavailable
```

```json
{
  "error": {
    "code": "SERVICE_UNAVAILABLE",
    "message": "Service is not ready",
    "details": {
      "checks": {
        "api": "ok",
        "minio": "error",
        "postgres": "ok",
        "redis": "ok"
      }
    }
  }
}
```

หมายเหตุ: ค่าใน `checks` จะต่างกันตาม service ที่ล่มจริง

## Endpoint Example Files

| File | Endpoint |
| --- | --- |
| `docs/18.1-auth-register-test-examples.md` | `POST /api/v1/auth/register` |
| `docs/18.2-auth-verify-email-test-examples.md` | `POST /api/v1/auth/verify-email` |
| `docs/18.3-auth-resend-verification-email-test-examples.md` | `POST /api/v1/auth/resend-verification-email` |
| `docs/18.4-auth-login-test-examples.md` | `POST /api/v1/auth/login` |
| `docs/18.5-auth-me-test-examples.md` | `GET /api/v1/auth/me` |
| `docs/18.6-auth-logout-test-examples.md` | `POST /api/v1/auth/logout` |
| `docs/18.7-auth-logout-all-test-examples.md` | `POST /api/v1/auth/logout-all` |
| `docs/18.8-auth-password-reset-test-examples.md` | `POST /api/v1/auth/forgot-password`, `POST /api/v1/auth/reset-password` |
| `docs/18.9-workspace-registration-test-examples.md` | `GET /api/v1/workspaces/check-slug`, `POST /api/v1/workspaces/register`, `GET /api/v1/workspaces/me` |
| `docs/18.10-workspace-current-test-examples.md` | `GET /api/v1/workspaces/current` |
| `docs/18.11-project-foundation-test-examples.md` | `GET /api/v1/workspace/projects`, `POST /api/v1/workspace/projects`, `GET /api/v1/workspace/projects/{project_id}`, `PATCH /api/v1/workspace/projects/{project_id}` |
| `docs/18.12-project-team-test-examples.md` | `GET /api/v1/workspace/projects/{project_id}/members`, `POST /api/v1/workspace/projects/{project_id}/members`, `PATCH /api/v1/workspace/projects/{project_id}/members/{member_id}`, `DELETE /api/v1/workspace/projects/{project_id}/members/{member_id}`, `GET /api/v1/workspace/projects/{project_id}/positions`, `PUT /api/v1/workspace/projects/{project_id}/members/{member_id}/positions` |
| `docs/18.13-task-foundation-test-examples.md` | `GET /api/v1/workspace/projects/{project_id}/tasks`, `GET /api/v1/workspace/projects/{project_id}/tasks/summary`, `POST /api/v1/workspace/projects/{project_id}/tasks`, `GET /api/v1/workspace/projects/{project_id}/tasks/{task_id}`, `PATCH /api/v1/workspace/projects/{project_id}/tasks/{task_id}`, `PATCH /api/v1/workspace/projects/{project_id}/tasks/{task_id}/status`, `DELETE /api/v1/workspace/projects/{project_id}/tasks/{task_id}` |

## Not Done Yet

ยังไม่มี Auth route ที่เป็น placeholder ในชุด backend auth foundation ปัจจุบัน

Workspace routes ถัดไปที่ยังไม่ทำ:

| Method | Endpoint | Purpose |
| --- | --- | --- |
| - | - | Project/Team/Task routes ที่ต้องใช้ Tenant Context |
