# ชุดที่ 12: README / Context Guard + AI Coding Prompts

Prasankit Project Platform

## เป้าหมายของชุดนี้

ชุดที่ 12 ใช้เป็นเอกสารควบคุม AI / Developer / Coding Agent เพื่อให้เขียนโค้ดตามสเปกที่วางไว้ ลดปัญหา context หลุด แชทรวน หรือ AI สร้างระบบนอกสโคป

## 12.1 Context Guard 4 ชั้น

Decision Final: ชุดที่ 12 ต้องไม่พึ่ง AGENTS.md อย่างเดียว

- README.md
- AGENTS.md
- docs/00-context-guard.md
- Prompt Header ที่แปะซ้ำในทุกคำสั่ง AI
เหตุผล: AI / Coding Agent บางตัวอาจไม่อ่าน AGENTS.md หรืออ่านไม่ครบ ดังนั้นกฎสำคัญต้องถูกย้ำหลายจุด โดยเฉพาะ Prompt Header ที่แปะไปกับคำสั่งโดยตรง

## 12.2 README.md

README.md ใช้สำหรับอธิบายภาพรวมระบบให้ developer และ AI เข้าใจ โดยควรมีข้อมูล Project Name, Project Goal, Tech Stack, Architecture Principles, Tenant Rules, API Rules, File Security Rules, Deploy / Backup Rules และ Important Do / Don’t

### Rule ที่ต้องอยู่ใน README

- Multi-tenant ใช้ Shared Database + tenant_id
- tenant_id ต้องมาจาก backend Tenant Context
- Frontend ไม่ควรส่ง tenant_id เป็นค่าที่เชื่อถือได้
- Workspace API ห้ามใส่ tenant_id ใน URL
- API prefix ใช้ /api/v1
- Success response ใช้ data
- Error response ใช้ error
- List response ใช้ data.items
- Pagination อยู่ใน data.pagination
- File ห้ามเก็บ Full URL จริงใน database
- เปิดไฟล์ผ่าน backend + signed URL
- ต้องมี Permission Guard
- List API ต้องมี pagination
- Soft delete / audit log ต้องใช้กับข้อมูลสำคัญ
- ห้ามเพิ่ม scope เองโดยไม่ระบุเหตุผล
### README.md Template

```text
# Prasankit

Prasankit is a multi-tenant project/workspace management platform.

## Main Stack
- Frontend: Next.js + TypeScript
- Backend: Go + Fiber
- ORM: GORM
- Database: PostgreSQL
- Cache / temporary state: Redis
- Object Storage: MinIO
- Deployment: Docker Compose

## Architecture Principles
- Multi-tenant architecture uses shared database + tenant_id.
- tenant_id must be resolved by backend Tenant Context.
- Frontend must not send tenant_id as a trusted value.
- Workspace API must not include tenant_id in URL.
- API prefix must use /api/v1.
- API success response must use root key data.
- API error response must use root key error.
- List response must use data.items and data.pagination.
- All Workspace / Project APIs must pass Permission Guard.
- Files must not store full public URL in database.
- File access must go through backend permission check and signed URL.
- Demo and Production environments must be separated.

## Important Rule
Do not change architecture, API format, tenant rule, file security rule, or response standard without explicitly documenting the reason.
```

## 12.3 AGENTS.md

AGENTS.md ใช้สำหรับ AI Agent ที่รองรับไฟล์นี้ โดยต้องระบุ Required Context Documents, Stack, Hard Rules, Coding Style, Scope Control และ Review Checklist

ข้อความสำคัญใน AGENTS.md: ถ้า AI Agent ไม่อ่าน AGENTS.md ให้ copy section “Prompt Header” ไปใส่ใน prompt ทุกครั้ง

```text
# AGENTS.md

You are working on Prasankit, a multi-tenant project/workspace management platform.

Before generating or modifying code, follow these rules strictly.

## Required Context
Read these documents first:
- README.md
- docs/00-context-guard.md
- docs/08-database-design.md
- docs/09-api-design.md
- docs/10-ui-page-structure.md
- docs/11-docker-deploy-domain-ssl.md

## Hard Rules
1. Do not put tenant_id in Workspace API URL.
2. tenant_id must come from backend Tenant Context only.
3. Every workspace/project query must filter by tenant_id.
4. API version prefix must be /api/v1.
5. Success response root key must be data.
6. Error response root key must be error.
7. List response must use data.items and data.pagination.
8. All sensitive actions must pass Permission Guard.
9. Do not store full file URLs in database.
10. File opening must go through backend permission check and signed URL.
11. All list APIs must have pagination.
12. Do not introduce out-of-scope features unless explicitly requested.
```

## 12.4 docs/00-context-guard.md

เอกสารนี้ต้องสั้น ชัด และใช้กันหลุดสเปก เป็นไฟล์ที่ AI และ Developer ควรอ่านก่อนเริ่มงานทุกครั้ง

```text
# Prasankit Context Guard

Project: Prasankit

Stack:
- Next.js + TypeScript
- Go Fiber + GORM
- PostgreSQL
- Redis
- MinIO
- Docker Compose

Hard Rules:
1. tenant_id ต้องมาจาก backend context
2. ห้ามใส่ tenant_id ใน Workspace API URL
3. API ใช้ /api/v1
4. success response ใช้ data
5. error response ใช้ error
6. list ใช้ data.items + data.pagination
7. file ห้ามเก็บ full URL
8. เปิดไฟล์ผ่าน signed URL จาก backend
9. ทุก Workspace/Project API ต้องผ่าน Permission Guard
10. list ต้อง pagination
11. ห้ามเพิ่ม scope เอง
```

## 12.5 Prompt Header

Prompt Header เป็นส่วนที่ต้องแปะก่อนสั่ง AI ทุกครั้ง เพราะเป็นชั้นที่กันพลาดได้ดีที่สุด

```text
Context Guard:
You are working on Prasankit.

Stack:
- Next.js + TypeScript
- Go Fiber + GORM
- PostgreSQL
- Redis
- MinIO
- Docker Compose

Hard Rules:
- tenant_id must come from backend Tenant Context only.
- Do not put tenant_id in Workspace API URL.
- API prefix must be /api/v1.
- Success response root key is data.
- Error response root key is error.
- List response uses data.items and data.pagination.
- File records must not store full URL.
- File access must use backend permission check + signed URL.
- Every Workspace/Project API must use Permission Guard.
- Every list API must have pagination.
- Do not add out-of-scope features.
```

## 12.6 Global Coding Prompt

```text
You are helping build Prasankit, a multi-tenant project/workspace management platform.

Use this stack:
- Frontend: Next.js + TypeScript
- Backend: Go + Fiber
- ORM: GORM
- Database: PostgreSQL
- Redis
- MinIO
- Docker Compose

Follow these architecture rules:
- Shared database + tenant_id.
- tenant_id must be resolved by backend Tenant Context.
- Do not accept tenant_id from frontend as trusted input.
- Workspace API must not include tenant_id in URL.
- API prefix must be /api/v1.
- Success response must use root key data.
- Error response must use root key error.
- List response must use data.items.
- Pagination must be inside data.pagination.
- All Workspace / Project endpoints must pass Permission Guard.
- All important actions must write Activity Log or Security Event.
- File database records must not store full URL.
- Store only bucket_key, object_key, file_name, mime_type, size.
- File access must go through backend permission check and signed URL.
- All list APIs must have pagination.
- Avoid LIKE '%keyword%' on large tables unless specifically justified.
- Use soft delete for important business entities.
- Unique constraints must consider soft delete.
- Do not introduce out-of-scope features.

When coding:
1. Explain the intended change briefly.
2. Show files to create/update.
3. Provide code in complete usable form.
4. Include validation.
5. Include error handling.
6. Keep naming consistent with Prasankit.
7. Do not change architecture unless explicitly asked.
```

## 12.7 Backend Prompt

```text
You are implementing backend code for Prasankit.

Stack:
- Go
- Fiber
- GORM
- PostgreSQL
- Redis where needed
- MinIO where needed

Architecture:
- handler/controller layer handles HTTP.
- service layer handles business logic.
- repository layer handles database queries.
- middleware handles auth, tenant context, permission guard.
- response package standardizes output.

API rules:
- Prefix: /api/v1
- Workspace API path must not contain tenant_id.
- tenant_id must come from Tenant Context middleware.
- Every workspace/project query must include tenant_id.
- Never query project_id, task_id, file_id alone without tenant_id.
- Success response root key = data.
- List response = data.items + data.pagination.
- Error response root key = error.

Security rules:
- Validate session from httpOnly Cookie / Redis session store.
- Check user account status.
- Check password_changed_at against session creation time if applicable.
- Check active workspace membership.
- Check Permission Guard for the action.
- Do not log secrets.
- Write activity_logs for business actions.
- Write security_events for security actions.

Task:
Implement [FEATURE_NAME].

Required output:
1. Route definitions.
2. Handler/controller.
3. Request/response structs.
4. Service.
5. Repository.
6. Validation.
7. Error handling.
8. Activity/security log where needed.
9. Notes for migration if schema changes.
```

## 12.8 Frontend Prompt

```text
You are implementing frontend code for Prasankit.

Stack:
- Next.js
- TypeScript
- App Router
- Tailwind CSS
- API integration with /api/v1 backend

UI contexts:
1. Public App Template
2. Owner / Business Admin Template
3. Workspace Portal Template
4. Project Detail Template
5. Platform Admin Template

Rules:
- Keep visual context clear using header label, sidebar, breadcrumb, badge, or accent.
- Workspace Portal must clearly show current workspace.
- Project Detail must clearly show current project.
- Do not mix Owner Console, Workspace Portal, and Platform Admin UI.
- All data visibility depends on role/permission from backend.
- Do not trust frontend-only permission checks.
- API success root is data.
- API error root is error.
- List response uses data.items and data.pagination.
- Do not hardcode tenant_id.
- Local dev tenant path uses /w/:slug.
- Production/demo tenant uses subdomain.
- File open must use backend download-url endpoint, not stored public URL.

Task:
Build [PAGE_OR_COMPONENT_NAME].

Required output:
1. Route structure.
2. Component structure.
3. TypeScript types.
4. API client function.
5. Loading state.
6. Error state.
7. Empty state.
8. Permission-aware UI behavior.
9. Responsive layout.
```

## 12.9 Database Migration Prompt

```text
You are creating database migrations for Prasankit.

Database:
- PostgreSQL
- Shared database + tenant_id

Rules:
- Use UUID v7 primary keys generated by the Go application.
- Use PostgreSQL UUID columns, but do not rely on `gen_random_uuid()` defaults for primary keys.
- Workspace data tables must include tenant_id.
- Most workspace data tables also include workspace_id.
- Project data tables include tenant_id, workspace_id, project_id.
- Important tables must include audit fields.
- Use soft delete for important business data.
- Unique constraints must consider soft delete.
- If deleted records can allow value reuse, use partial unique index WHERE deleted_at IS NULL.
- Add indexes for tenant_id + common filters.
- Add indexes for join columns.
- Avoid creating bloated main tables.
- Use child tables for optional/multi-value/history data.
- Do not store full file URL.
- Store bucket_key and object_key only.

Required output:
1. Migration SQL or migration file.
2. Indexes.
3. Constraints.
4. Soft delete considerations.
5. Rollback migration if applicable.
6. Notes about high-volume tables.
```

## 12.10 Docker / Deploy Prompt

```text
You are writing Docker/deploy configuration for Prasankit.

Service names:
- prasankit-web
- prasankit-api
- prasankit-pgsql
- prasankit-redis
- prasankit-minio
- prasankit-proxy

Rules:
- Use Docker Compose for MVP/Demo.
- Use bind mount or clearly defined volumes for data safety.
- Important data must not live only in container layer.
- PostgreSQL data path must be persistent.
- MinIO data path must be persistent.
- Do not use docker compose down -v in demo/production.
- Do not use docker system prune --volumes in demo/production.
- Use .env for secrets/config.
- Do not commit real secrets.
- Add healthcheck for pgsql, redis, minio.
- api depends on healthy pgsql/redis/minio.
- web depends on api.
- Use deploy script to reduce manual command mistakes.

Generate:
1. docker-compose.yml
2. .env.example
3. deploy.sh
4. backup script skeleton
5. notes for data safety
```

## 12.11 Review Prompt

```text
Review the current implementation of Prasankit.

Check:
- Is tenant_id resolved by backend context?
- Is tenant_id absent from Workspace API URL?
- Do workspace/project queries filter by tenant_id?
- Are permissions enforced by backend?
- Is /api/v1 used consistently?
- Does success response use data?
- Does error response use error?
- Do list responses use data.items and data.pagination?
- Do list APIs have pagination?
- Are file URLs stored or exposed incorrectly?
- Does file access use signed URL?
- Are sensitive values logged?
- Are activity logs written for important actions?
- Are security events written for auth/security actions?
- Are soft deletes used correctly?
- Are unique constraints compatible with soft delete?
- Are indexes adequate?
- Are dangerous docker commands avoided?
- Are backups documented?

Return:
1. Pass/fail summary.
2. Critical issues.
3. Medium issues.
4. Minor issues.
5. Recommended fixes.
6. Files likely affected.
```

## 12.12 Scope Guard Prompt

```text
Before answering, check the current scope.

Current document set:
[DOCUMENT_SET_NAME]

Current topic:
[CURRENT_TOPIC]

Allowed scope:
[ALLOWED_SCOPE]

Not allowed:
[OUT_OF_SCOPE]

If the request is outside scope:
- Do not expand deeply.
- Mark it as future/optional.
- Return to current topic.

Do not redesign previous locked decisions unless user explicitly asks.
```

## 12.13 New Chat Context Prompt

```text
Summarize the current Prasankit project context for a new chat.

Include:
1. Project goal
2. Tech stack
3. Architecture rules
4. Tenant rules
5. API rules
6. Database rules
7. File security rules
8. UI template rules
9. Docker/deploy rules
10. Current roadmap status
11. Important decisions already locked
12. What should not be changed
13. Next task to continue

Keep it structured and compact.
Do not invent new scope.
Do not change previous decisions.
```

## 12.14 Module Prompt Examples

### Project Module

```text
Implement Project module for Prasankit.

Scope:
- List Projects
- Create Project
- Get Project Detail
- Update Project
- Archive/Restore Project
- Project Members
- Project Settings basic

Rules:
- Endpoint under /api/v1/workspace/projects.
- No tenant_id in URL.
- tenant_id from Tenant Context.
- Every query uses tenant_id.
- project_id must always be paired with tenant_id.
- Create Project must add creator as Project Owner.
- Project must have at least one Project Owner.
- Project code follows workspace policy.
- Use soft delete/archive.
- Write activity logs.
- List has pagination.
```

### Task Module

```text
Implement Task module for Prasankit.

Scope:
- List/Create/Get/Update/Delete Task
- Change Status
- Reassign
- Watchers
- Checklist
- Comments
- Attachments as file relation

Rules:
- Endpoint under /api/v1/workspace/projects/{project_id}/tasks.
- No tenant_id in URL.
- tenant_id from Tenant Context.
- Query by tenant_id + project_id + task_id.
- Validate assignee is active project member.
- Reassign must write task_reassign_logs.
- Attachments must link to Project File Center.
- Write activity logs.
- List has pagination.
```

### File Module

```text
Implement Document/File module for Prasankit.

Scope:
- Folder tree
- File list
- Upload request
- Complete upload
- Download signed URL
- Update metadata
- Soft delete/restore
- External link
- File relations

Rules:
- Do not store full URL.
- Store bucket_key and object_key.
- Client must not generate object_key.
- Backend validates permission before upload/download.
- Backend returns signed URL with expiration.
- Do not log signed URL.
- Delete from UI = soft delete.
- Object hard delete only via cleanup policy.
- Attachment = relation, not copy.
```

## 12.15 Final Decision ชุดที่ 12

```text
ชุดที่ 12:
ต้องมี README / Context Guard และ Prompt Template สำหรับควบคุม AI / Developer / Coding Agent

Context Guard ต้องมี 4 ชั้น:
1. README.md
2. AGENTS.md
3. docs/00-context-guard.md
4. Prompt Header ที่แปะซ้ำในทุกคำสั่ง AI

Prompt ทุกชุดต้องย้ำ rule สำคัญ:
- tenant_id จาก backend context เท่านั้น
- Workspace API ไม่ใส่ tenant_id ใน URL
- API prefix /api/v1
- success ใช้ data
- error ใช้ error
- list ใช้ data.items + data.pagination
- file ห้ามเก็บ full URL
- signed URL ต้องผ่าน backend
- permission guard ต้องมี
- list ต้อง pagination
- soft delete / audit log ต้องใช้
- ห้ามเพิ่ม scope เอง
```
