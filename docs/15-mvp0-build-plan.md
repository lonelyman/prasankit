# MVP-0 Build Plan

เอกสารนี้แตก MVP-0 User Flow ออกเป็นงานที่ต้องสร้างจริง

## หลักคิด

MVP-0 ต้องทำระบบให้เดิน end-to-end ได้ก่อน ไม่เน้นความสวยหรือ feature ครบทุก module

ลำดับทำงาน:

```text
Foundation
-> Auth / Session
-> Workspace
-> Tenant Context
-> Project
-> Project Team
-> Task
-> File
-> Activity Log
```

## 1. Foundation

เป้าหมาย:

เตรียมโครง backend, frontend, database, docker ให้พร้อมรัน local

Backend:

- Go module
- Fiber server
- config loader
- PostgreSQL connection
- Redis connection
- MinIO connection
- health endpoint
- response presenter
- request_id middleware

Frontend:

- Next.js app
- Tailwind / shadcn setup
- base layout
- API client
- route groups
- error pages

Infra:

- docker-compose.yml
- .env.example
- PostgreSQL
- Redis
- MinIO
- Traefik placeholder / local proxy config ภายหลัง

เสร็จเมื่อ:

- เปิด backend health check ได้
- เปิด frontend home page ได้
- docker compose รัน service พื้นฐานได้

## 2. Auth / Session

เป้าหมาย:

ผู้ใช้สมัคร, ยืนยัน email, login, logout ได้

Tables:

- user_accounts
- auth_sessions
- auth_login_attempts
- auth_email_verification_tokens
- auth_password_reset_tokens
- security_events

API:

- POST /api/v1/auth/register
- POST /api/v1/auth/verify-email
- POST /api/v1/auth/login
- POST /api/v1/auth/logout
- POST /api/v1/auth/logout-all
- POST /api/v1/auth/forgot-password
- POST /api/v1/auth/reset-password
- GET /api/v1/auth/me

UI:

- Login
- Verify Email
- Forgot Password
- Reset Password

เสร็จเมื่อ:

- login แล้วได้ httpOnly Cookie
- session อยู่ใน Redis
- logout แล้ว session ใช้ไม่ได้

## 3. Workspace

เป้าหมาย:

ผู้ใช้สร้าง workspace และเข้า workspace ของตัวเองได้

Tables:

- workspaces
- workspace_memberships
- workspace_configs
- workspace_master_items
- master_seed_logs

API:

- POST /api/v1/workspaces/register
- GET /api/v1/workspaces/check-slug
- GET /api/v1/workspaces/me
- GET /api/v1/workspace
- PATCH /api/v1/workspace

UI:

- Create Workspace
- Find Workspace
- Workspace Dashboard แบบพื้นฐาน

เสร็จเมื่อ:

- สมัคร workspace ได้
- owner ถูกสร้างเป็น workspace member
- user เห็น workspace ของตัวเอง

## 4. Tenant Context

เป้าหมาย:

ทุก request ภายใน workspace ต้องรู้ว่าอยู่ tenant ไหน โดย backend resolve เอง

Backend:

- tenant middleware
- workspace slug resolver
- development path mode: /w/{workspaceSlug}
- production subdomain mode ภายหลัง
- membership guard
- permission guard เบื้องต้น

Rules:

- ห้ามรับ tenant_id จาก frontend เป็นค่าที่เชื่อถือได้
- Workspace API ห้ามใส่ tenant_id ใน URL
- ทุก query ของ workspace data ต้อง filter tenant_id

เสร็จเมื่อ:

- user เข้า workspace ที่ตัวเองเป็น member ได้
- user เข้า workspace คนอื่นไม่ได้
- API ใช้ tenant_id จาก context

## 5. Project

เป้าหมาย:

สร้างและเปิด project ได้

Tables:

- projects
- project_members
- project_code_counters
- project_master_items

API:

- GET /api/v1/workspace/projects
- POST /api/v1/workspace/projects
- GET /api/v1/workspace/projects/{project_id}
- PATCH /api/v1/workspace/projects/{project_id}
- POST /api/v1/workspace/projects/{project_id}/archive
- POST /api/v1/workspace/projects/{project_id}/restore

UI:

- Project List
- Create Project
- Project Detail Overview

เสร็จเมื่อ:

- สร้าง project ได้
- current user เป็น Project Owner อัตโนมัติ
- project เปิดได้เฉพาะคนที่มีสิทธิ์

## 6. Project Team

เป้าหมาย:

จัดการสมาชิกใน project ได้

Tables:

- project_members
- project_member_positions

API:

- GET /api/v1/workspace/projects/{project_id}/members
- POST /api/v1/workspace/projects/{project_id}/members
- PATCH /api/v1/workspace/projects/{project_id}/members/{member_id}
- DELETE /api/v1/workspace/projects/{project_id}/members/{member_id}

UI:

- Project Team tab
- Add member modal
- Change role
- Remove member

Rules:

- Project ต้องมี Project Owner อย่างน้อย 1 คน
- ห้าม remove Project Owner คนสุดท้าย
- เพิ่มได้เฉพาะ active workspace member

เสร็จเมื่อ:

- owner เพิ่ม member เข้า project ได้
- owner เปลี่ยน role ได้
- owner คนสุดท้ายถูก remove ไม่ได้

## 7. Task

เป้าหมาย:

สร้าง task และย้าย status บน board ได้

Tables:

- tasks
- task_watchers
- task_checklists
- task_comments
- task_reassign_logs
- task_relations
- task_counters

API:

- GET /api/v1/workspace/projects/{project_id}/tasks
- POST /api/v1/workspace/projects/{project_id}/tasks
- GET /api/v1/workspace/projects/{project_id}/tasks/{task_id}
- PATCH /api/v1/workspace/projects/{project_id}/tasks/{task_id}
- DELETE /api/v1/workspace/projects/{project_id}/tasks/{task_id}
- POST /api/v1/workspace/projects/{project_id}/tasks/{task_id}/status
- POST /api/v1/workspace/projects/{project_id}/tasks/{task_id}/reassign

UI:

- Task Board
- Task List
- Create Task
- Task Detail drawer

เสร็จเมื่อ:

- สร้าง task ได้
- assign task ได้
- ย้าย status ได้
- reassign แล้วมี log

## 8. File

เป้าหมาย:

upload file และเปิดผ่าน signed URL ได้

Tables:

- project_folders
- project_files
- file_relations
- file_lifecycle_logs

API:

- GET /api/v1/workspace/projects/{project_id}/files
- POST /api/v1/workspace/projects/{project_id}/folders
- POST /api/v1/workspace/projects/{project_id}/files/upload-request
- POST /api/v1/workspace/projects/{project_id}/files/complete-upload
- GET /api/v1/workspace/projects/{project_id}/files/{file_id}/download-url
- PATCH /api/v1/workspace/projects/{project_id}/files/{file_id}
- DELETE /api/v1/workspace/projects/{project_id}/files/{file_id}
- POST /api/v1/workspace/projects/{project_id}/files/{file_id}/relations

UI:

- Documents tab
- Folder list
- File upload
- File open/download

Rules:

- ห้ามเก็บ full file URL
- backend ตรวจสิทธิ์ก่อนสร้าง signed URL
- delete เป็น soft delete

เสร็จเมื่อ:

- upload file ได้
- เปิดไฟล์ได้ผ่าน signed URL
- ผูก file กับ task ได้

## 9. Activity Log

เป้าหมาย:

เห็นประวัติ action สำคัญใน project ได้

Tables:

- activity_logs
- security_events

API:

- GET /api/v1/workspace/projects/{project_id}/activity

UI:

- Project Activity tab

ต้อง log:

- workspace created
- login success / failed เป็น security event
- project created
- project member added
- task created
- task status changed
- task reassigned
- file uploaded

เสร็จเมื่อ:

- activity tab แสดงประวัติ project ได้
- security event แยกจาก activity log

## Build Order

ลำดับที่ควรทำจริง:

1. Foundation
2. Auth / Session
3. Workspace Registration
4. Tenant Context
5. Project
6. Project Team
7. Task
8. File
9. Activity Log

## First Coding Step

เมื่อเริ่มเขียนโค้ด ให้เริ่มจาก Foundation เท่านั้น:

- สร้าง app-api/ Go Fiber server
- สร้าง app-web/ Next.js app
- สร้าง docker-compose.yml
- ทำ health check

ยังไม่เขียน business feature จนกว่า foundation จะรันได้
