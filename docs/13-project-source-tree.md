# Prasankit Project Platform

ชุดที่ 13: Project Source Tree / Folder Structure

Root Project Tree, API Tree, Web Tree, Package Naming Rules

## เป้าหมายของชุดนี้

- วางโครงสร้าง Source Code ของระบบ Prasankit
- กำหนด Root project tree, app-api/ tree และ app-web/ tree
- กำหนดแนวทาง package naming เพื่อไม่ให้ import ชนกันใน Go
- แยก context ของ frontend ให้ชัดตาม UI Template ที่ล็อกไว้
- ใช้เป็น guideline ให้ developer และ AI Coding Agent ทำงานต่อได้ไม่หลุดโครงสร้าง
## 13.1 Root Project Tree

### Decision Final

```text
Root folder ใช้ชื่อ prasankit
ภายในใช้ app-prefixed folder เพื่อให้อ่านเรียงกันชัดเจน:
- app-api
- app-web
```

### Root Tree

```text
prasankit/
├─ README.md
├─ AGENTS.md
├─ docker-compose.yml
├─ .env.example
├─ .gitignore
│
├─ docs/
│ ├─ 00-context-guard.md
│ ├─ 01-system-overview.md
│ ├─ 02-workspace-registration.md
│ ├─ 03-project-structure.md
│ ├─ 04-notification-calendar-reminder.md
│ ├─ 05-master-config-permission.md
│ ├─ 06-tenant-data-isolation.md
│ ├─ 07-module-map-system-structure.md
│ ├─ 08-database-design.md
│ ├─ 09-api-design.md
│ ├─ 10-ui-page-structure.md
│ ├─ 11-docker-deploy-domain-ssl.md
│ ├─ 12-ai-coding-prompts.md
│ └─ 13-project-source-tree.md
│
├─ app-api/
├─ app-web/
│
├─ deploy/
│ ├─ traefik/
│ │ └─ conf.d/
│ └─ scripts/
│ ├─ deploy.sh
│ ├─ backup-postgres.sh
│ ├─ backup-minio.sh
│ └─ restore-notes.md
│
├─ data/
│ ├─ postgres/
│ ├─ redis/
│ └─ minio/
│
├─ logs/
│ ├─ app-api/
│ ├─ app-web/
│ └─ proxy/
│
└─ backups/
├─ postgres/
└─ minio/
```

### ความหมายของ Folder หลัก

| Folder | หน้าที่ |
| --- | --- |
| app-api/ | Go Fiber backend |
| app-web/ | Next.js frontend |
| docs/ | เอกสารคุม context และ requirement |
| deploy/ | Traefik config + deploy / backup scripts |
| data/ | bind mount ข้อมูลสำคัญ ห้ามลบมั่ว |
| logs/ | log ที่แยกออกมาให้อ่านง่าย |
| backups/ | output backup local ก่อน sync ไปที่อื่น |

Docker Service / Container Name

```text
app-api/ → service/container: prasankit-api
app-web/ → service/container: prasankit-web
```

## 13.2 app-api/ Tree

### Decision Final

```text
app-api/ ใช้ Pragmatic Hexagonal Architecture
```

- ถูกหลัก Go มากกว่าใช้ชื่อ generic
- เหมาะกับ Hexagonal Architecture
- แยก domain / service / adapter / transport ชัด
- VS Code auto import ได้ง่าย
- ลดปัญหา package ชน และไม่ต้องตั้ง alias เองบ่อย
### app-api/ Tree

```text
app-api/
├─ Dockerfile
├─ README.md
├─ go.mod
├─ go.sum
│
├─ cmd/
│ └─ api/
│ └─ main.go
│
├─ internal/
│ ├─ bootstrap/
│ │ ├─ app.go
│ │ ├─ db.go
│ │ ├─ http.go
│ │ ├─ redis.go
│ │ ├─ storage.go
│ │ └─ mailer.go
│ │
│ ├─ config/
│ │ └─ config.go
│ │
│ ├─ modules/
│ │ ├─ auth/
│ │ │ ├─ auth_entity.go
│ │ │ ├─ auth_repository.go
│ │ │ └─ authsvc/
│ │ │ └─ auth_service.go
│ │ │
│ │ ├─ user/
│ │ │ ├─ user_entity.go
│ │ │ ├─ profile_entity.go
│ │ │ ├─ user_repository.go
│ │ │ └─ usersvc/
│ │ │ └─ user_service.go
│ │ │
│ │ ├─ workspace/
│ │ │ ├─ workspace_entity.go
│ │ │ ├─ workspace_repository.go
│ │ │ └─ workspacesvc/
│ │ │ └─ workspace_service.go
│ │ │
│ │ ├─ member/
│ │ ├─ project/
│ │ ├─ task/
│ │ ├─ timeline/
│ │ ├─ finance/
│ │ ├─ file/
│ │ ├─ notification/
│ │ ├─ report/
│ │ ├─ audit/
│ │ └─ platform/
│ │
│ ├─ adapters/
│ │ ├─ database/
│ │ │ └─ postgres/
│ │ │ ├─ authrepo/
│ │ │ ├─ userrepo/
│ │ │ ├─ workspacerepo/
│ │ │ ├─ memberrepo/
│ │ │ ├─ projectrepo/
│ │ │ ├─ taskrepo/
│ │ │ ├─ timelinerepo/
│ │ │ ├─ financerepo/
│ │ │ ├─ filerepo/
│ │ │ ├─ notificationrepo/
│ │ │ ├─ reportrepo/
│ │ │ ├─ auditrepo/
│ │ │ └─ platformrepo/
│ │ │
│ │ ├─ cache/
│ │ │ └─ redis/
│ │ │ └─ redis_client.go
│ │ │
│ │ ├─ storage/
│ │ │ └─ minio/
│ │ │ ├─ minio_client.go
│ │ │ └─ signed_url.go
│ │ │
│ │ └─ mailer/
│ │ └─ smtp/
│ │ └─ smtp_mailer.go
│ │
│ └─ transport/
│ └─ http/
│ ├─ router.go
│ ├─ presenter/
│ │ └─ presenter.go
│ ├─ middlewares/
│ │ ├─ auth.go
│ │ ├─ tenant.go
│ │ ├─ permission.go
│ │ ├─ request_id.go
│ │ ├─ rate_limit.go
│ │ └─ logger.go
│ │
│ ├─ health/
│ │ └─ health_handler.go
│ ├─ authhttp/
│ ├─ userhttp/
│ ├─ workspacehttp/
│ ├─ memberhttp/
│ ├─ projecthttp/
│ ├─ taskhttp/
│ ├─ timelinehttp/
│ ├─ financehttp/
│ ├─ filehttp/
│ ├─ notificationhttp/
│ ├─ reporthttp/
│ ├─ audithttp/
│ └─ platformhttp/
│
├─ migrations/
├─ pkg/
└─ tests/
```

### app-api/ Layer Rule

```text
internal/modules
= domain + repository interface

internal/modules/{module}/{module}svc
= business service / use case

internal/adapters/database/postgres/{module}repo
= GORM / PostgreSQL repository implementation

internal/transport/http/{module}http
= HTTP handler / request / response / routes

internal/transport/http/presenter
= response format กลาง

pkg
= shared utility ที่ไม่ผูก business module เช่น `pkg/ids` สำหรับ UUID v7 generator กลาง
```

### Package Naming Rule

```text
Domain package:
project, task, workspace

Service package:
projectsvc, tasksvc, workspacesvc

Repository implementation:
projectrepo, taskrepo, workspacerepo

HTTP transport:
projecthttp, taskhttp, workspacehttp
```

### ตัวอย่าง Import ที่ต้องการ

```text
import (
"prasankit-api/internal/modules/project/projectsvc"
"prasankit-api/internal/transport/http/presenter"
"prasankit-api/pkg/auth"
"prasankit-api/pkg/custom_errors"
"prasankit-api/pkg/logger"
"prasankit-api/pkg/validation"

"github.com/gofiber/fiber/v3"
"github.com/google/uuid"
)
```

## 13.3 app-web/ Tree

### Decision Final

```text
app-web/ ใช้ Next.js App Router แบบสากล
แยก route ด้วย Route Group ตาม UI Context
แยก business logic ด้วย features/
แยก reusable UI ด้วย components/
แยก API client ด้วย lib/api/
แยก type กลางด้วย types/
และใช้ middleware.ts สำหรับ auth/tenant routing เบื้องต้น
```

### app-web/ Tree

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
│
├─ public/
│ ├─ images/
│ ├─ icons/
│ └─ favicon.ico
│
└─ src/
├─ app/
│ ├─ layout.tsx
│ ├─ globals.css
│ ├─ not-found.tsx
│ ├─ error.tsx
│
│ ├─ (public)/
│ │ ├─ layout.tsx
│ │ ├─ page.tsx
│ │ ├─ create-workspace/
│ │ ├─ find-workspace/
│ │ ├─ login/
│ │ ├─ forgot-password/
│ │ ├─ reset-password/
│ │ ├─ verify-email/
│ │ ├─ contact/
│ │ └─ faq/
│
│ ├─ (owner)/
│ │ └─ owner/
│ │ ├─ layout.tsx
│ │ ├─ login/
│ │ ├─ dashboard/
│ │ ├─ workspaces/
│ │ └─ cleanup/
│
│ ├─ (admin)/
│ │ └─ admin/
│ │ ├─ layout.tsx
│ │ ├─ login/
│ │ ├─ dashboard/
│ │ ├─ workspaces/
│ │ ├─ configs/
│ │ ├─ master-data/
│ │ ├─ reference-data/
│ │ ├─ audit-logs/
│ │ └─ security-events/
│
│ └─ (workspace)/
│ └─ w/
│ └─ [workspaceSlug]/
│ ├─ layout.tsx
│ ├─ login/
│ ├─ dashboard/
│ ├─ projects/
│ │ ├─ page.tsx
│ │ └─ [projectId]/
│ │ ├─ layout.tsx
│ │ ├─ page.tsx
│ │ ├─ tasks/
│ │ ├─ timeline/
│ │ ├─ documents/
│ │ ├─ team/
│ │ ├─ finance/
│ │ ├─ announcements/
│ │ ├─ activity/
│ │ └─ settings/
│ ├─ my-tasks/
│ ├─ calendar/
│ ├─ team/
│ ├─ reports/
│ ├─ notifications/
│ ├─ announcements/
│ └─ settings/
│
├─ components/
│ ├─ ui/
│ ├─ common/
│ ├─ public/
│ ├─ owner/
│ ├─ admin/
│ ├─ workspace/
│ ├─ project/
│ └─ forms/
│
├─ features/
│ ├─ auth/
│ ├─ workspace/
│ ├─ project/
│ ├─ task/
│ ├─ timeline/
│ ├─ finance/
│ ├─ file/
│ ├─ notification/
│ ├─ report/
│ └─ member/
│
├─ lib/
│ ├─ api/
│ ├─ auth/
│ ├─ tenant/
│ ├─ permissions/
│ ├─ constants/
│ ├─ utils/
│ └─ validators/
│
├─ hooks/
├─ types/
├─ styles/
└─ middleware.ts
```

### app-web/ หลักที่ล็อก

- ใช้ src/app ตามมาตรฐาน Next.js App Router
- ใช้ Route Group เช่น (public), (owner), (admin), (workspace) เพื่อแยก layout
- Local dev ใช้ /w/[workspaceSlug]
- Production/Demo ใช้ subdomain ได้ แต่ frontend ยัง reuse workspace context เดิม
- Project Detail อยู่ใต้ /w/[workspaceSlug]/projects/[projectId]
- components แยกตาม UI Context
- features แยกตาม business module
- lib/api เป็น API client กลาง
- types เก็บ TypeScript types กลาง
- middleware.ts ใช้ช่วยตรวจ routing/auth/tenant เบื้องต้นฝั่ง Next.js
### Frontend Rule สำคัญ

```text
Frontend ห้าม hardcode tenant_id

ใช้ได้เฉพาะ:
- workspaceSlug
- projectId

Backend เป็นฝ่าย resolve:
workspaceSlug → tenant_id / workspace_id

API ที่ frontend เรียก:
/api/v1/workspace/...

ไม่ใช้:
/api/v1/{tenant_id}/workspace/...
```

## 13.4 Decision รวม

```text
ชุดที่ 13:
ใช้ root folder prasankit
ภายในแยก app-api และ app-web เพื่อให้อ่านเรียงกันชัดเจน

app-api/ ใช้ Pragmatic Hexagonal Architecture
โดยคุม package naming เพื่อไม่ให้ import ชน

app-web/ ใช้ Next.js App Router แบบสากล
โดยแยก route group, components, features, lib/api และ types ให้ชัดเจน
```
