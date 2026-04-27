# Foundation File Plan

เอกสารนี้บอกไฟล์แรกที่ต้องสร้างเมื่อเริ่มเขียนโค้ดจริง

ยังไม่ใช่ feature plan และยังไม่ลง business logic

## เป้าหมายของ Foundation

ทำให้ระบบพื้นฐานรันได้ก่อน:

```text
Frontend เปิดได้
Backend health check ได้
PostgreSQL ต่อได้
Redis ต่อได้
MinIO ต่อได้
docker compose รันได้
```

เมื่อ Foundation เสร็จแล้ว ค่อยเริ่ม Auth / Session

## Root Files

ไฟล์ที่ควรมีที่ root:

```text
README.md
AGENTS.md
.gitignore
.env.example
docker-compose.yml
```

มีแล้ว:

```text
README.md
AGENTS.md
.gitignore
```

ยังต้องสร้าง:

```text
.env.example
docker-compose.yml
```

## app-api/ Files

โครงแรกของ backend:

```text
app-api/
├─ Dockerfile
├─ README.md
├─ go.mod
├─ go.sum
├─ cmd/
│  └─ api/
│     └─ main.go
└─ internal/
   ├─ bootstrap/
   │  ├─ app.go
   │  ├─ db.go
   │  ├─ redis.go
   │  ├─ storage.go
   │  └─ http.go
   ├─ config/
   │  └─ config.go
   └─ transport/
      └─ http/
         ├─ router.go
         ├─ health/
         │  └─ health_handler.go
         ├─ middlewares/
         │  └─ request_id.go
         └─ presenter/
            └─ presenter.go
```

## app-api/ Foundation Scope

Backend Foundation ทำแค่นี้:

- load config จาก environment
- connect PostgreSQL
- connect Redis
- create MinIO client
- start Fiber server
- expose health endpoint
- add request_id middleware
- return response format มาตรฐาน

ยังไม่ทำ:

- auth
- tenant context
- permission guard
- migrations จริง
- business modules

## Backend Health Endpoint

Endpoint พื้นฐาน:

```text
GET /api/v1/health
GET /api/v1/health/live
GET /api/v1/health/ready
```

`/api/v1/health` คงไว้เป็น legacy/basic alias ของ live health เพื่อให้ integration เดิมไม่พัง

Live response:

```json
{
  "data": {
    "status": "ok"
  }
}
```

Ready response ตอนที่ยังไม่ได้ wire dependency client:

```json
{
  "data": {
    "status": "ok",
    "checks": {
      "api": "ok"
    }
  }
}
```

ภายหลังเมื่อ app-api มี PostgreSQL / Redis / MinIO client แล้ว ให้เพิ่ม dependency check ใน `/api/v1/health/ready` ไม่ใช่ live health

## app-web/ Files

โครงแรกของ frontend:

```text
app-web/
├─ Dockerfile
├─ README.md
├─ package.json
├─ package-lock.json
├─ next.config.ts
├─ tsconfig.json
├─ eslint.config.mjs
├─ postcss.config.mjs
├─ tailwind.config.ts
├─ .env.example
└─ src/
   ├─ app/
   │  ├─ layout.tsx
   │  ├─ page.tsx
   │  ├─ globals.css
   │  ├─ not-found.tsx
   │  └─ error.tsx
   ├─ components/
   │  └─ ui/
   ├─ features/
   ├─ lib/
   │  ├─ api/
   │  │  └─ client.ts
   │  └─ utils/
   └─ types/
```

## app-web/ Foundation Scope

Frontend Foundation ทำแค่นี้:

- Next.js app เปิดหน้าแรกได้
- มี global style
- มี API client กลาง
- มี error page / not found page
- เตรียม folder structure ตาม docs/13

ยังไม่ทำ:

- login page
- workspace page
- project page
- task board
- file upload

## Docker Files

ต้องมี:

```text
docker-compose.yml
app-api/Dockerfile
app-web/Dockerfile
.env.example
```

Services ใน docker-compose:

```text
prasankit-web
prasankit-api
prasankit-pgsql
prasankit-redis
prasankit-minio
```

Traefik อาจใส่เป็น placeholder ก่อนใน local foundation ถ้ายังไม่ต้องใช้ subdomain จริง

## Environment Variables แรก

ขั้นต่ำใน `.env.example`:

```text
API_ENV=development
API_PORT=8080
API_TIMEZONE=Asia/Bangkok

WEB_PORT=3000
NEXT_PUBLIC_API_URL=http://localhost:8080
INTERNAL_API_URL=http://prasankit-api:8080

TENANT_RESOLUTION_MODE=path
DEV_TENANT_PATH_PREFIX=/w

POSTGRES_PRIMARY_HOST=prasankit-pgsql
POSTGRES_PRIMARY_PORT=5432
POSTGRES_PRIMARY_USER=prasankit
POSTGRES_PRIMARY_PASSWORD=change_me
POSTGRES_PRIMARY_NAME=prasankit
POSTGRES_SSL_MODE=disable

REDIS_HOST=prasankit-redis
REDIS_PORT=6379
REDIS_PASSWORD=change_me
REDIS_DB=0

MINIO_ROOT_USER=prasankit
MINIO_ROOT_PASSWORD=change_me
STORAGE_ENDPOINT=http://prasankit-minio:9000
STORAGE_BUCKET=prasankit
STORAGE_PRESIGN_TTL=15m

SESSION_COOKIE_NAME=prasankit_session
SESSION_SECRET=change_me
SESSION_TTL=24h
COOKIE_SECURE=false
COOKIE_SAMESITE=Lax
```

## Foundation Definition of Done

Foundation ถือว่าเสร็จเมื่อ:

1. `docker compose up -d` รัน service หลักได้
2. backend เปิด `GET /api/v1/health` ได้
3. frontend เปิดหน้าแรกได้
4. backend connect PostgreSQL ได้
5. backend connect Redis ได้
6. backend สร้าง MinIO client ได้
7. response health ใช้ root key `data`
8. ยังไม่มี business feature ปนเข้ามา

Current implementation note:

- Health check backend ใช้ Fiber v3
- app-api build/run ผ่าน Docker
- Dockerfile ใช้ Go 1.25 เป็นขั้นต่ำสำหรับ Fiber v3 ระหว่างรอขยับตาม baseline Go 1.26.x

## Step หลัง Foundation

เมื่อ Foundation เสร็จ ขั้นถัดไปคือ:

```text
Auth / Session
```

ยังไม่ข้ามไป Project หรือ Task จนกว่า Auth และ Workspace จะพร้อม
