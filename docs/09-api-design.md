# ชุดที่ 9: API Design

Project Platform Specification

| รายการ | รายละเอียด |
| --- | --- |
| Version | Draft 1.0 |
| Scope | API Design only |
| API Base | /api/v1 |
| Style | REST API เป็นหลัก |
| Tenant Rule | Workspace API ไม่ใส่ tenant_id ใน URL; ใช้ Tenant Context จาก backend |
| Response Note | รูปแบบ response เป็นแนวทางเบื้องต้น รอยืนยันรายละเอียดตอน dev จริง |

เอกสารชุดนี้เป็นการออกแบบ API ระดับภาพรวมและ endpoint สำคัญตาม Module ที่ล็อกไว้ก่อนหน้า โดยยังไม่ลงรายละเอียด UI, Docker, Deploy หรือ code จริง

## 9.1 API Design Principle

- ใช้ REST API เป็นหลัก
- API ทุกตัวใช้ prefix /api/v1
- แยกกลุ่ม API เป็น Auth API, Platform API และ Workspace API
- Workspace API ทุกตัวต้องผ่าน Tenant Context + Permission Guard
- ไม่รับ tenant_id จาก frontend เป็นหลัก
- ใช้ Session-based Auth ผ่าน httpOnly Cookie
- List API ต้องมี pagination เสมอ
- Search ต้องระวัง ไม่ใช้ LIKE %% ตรง ๆ บน table ใหญ่
- Action สำคัญต้องเขียน Activity Log / Security Event ตามกรณี

| API Group | Base Path | ใช้สำหรับ |
| --- | --- | --- |
| Auth API | /api/v1/auth/... | login, token, session, password, email verification |
| Platform API | /api/v1/platform/... | ผู้ดูแลระบบกลาง จัดการ workspace, config, cleanup, audit |
| Workspace API | /api/v1/workspace/... | ข้อมูลภายใน workspace หลัง resolve tenant แล้ว |
| Workspace Registration | /api/v1/workspaces/... | สมัคร workspace, check slug, list my workspaces |

### API Tenant Path Rule

- Workspace API ไม่ใส่ tenant_id ใน URL
- Production resolve tenant จาก subdomain → slug → tenant_id → workspace_id
- Development resolve tenant จาก path /w/:slug → slug → tenant_id → workspace_id
- Platform API สามารถระบุ workspace_id ใน path ได้ เพราะเป็น API สำหรับ Platform Admin

```text
Production:
https://company-a-demo.yourdomain.com/api/v1/workspace/projects

Development:
http://localhost:3000/w/company-a-demo/api/v1/workspace/projects

Not allowed for Workspace API:
/api/v1/{tenant_id}/workspace/projects
```

## 9.2 Auth API

- Auth API ใช้จัดการตัวตนบัญชีผู้ใช้กลาง
- Auth API บอกว่า user คือใคร ส่วน Permission Guard บอกว่า user ทำอะไรได้ที่ไหน
- Workspace Role และ Project Role ไม่ควรผูกอยู่ใน Auth โดยตรง
- `user_accounts` เป็น account กลาง ส่วน `auth_identities` เป็นช่องทาง login เช่น email/password และ OAuth provider ในอนาคต
- ห้ามใช้ email อย่างเดียวเป็น source of truth ของ social login; OAuth ต้องแยกด้วย `provider + provider_user_id`

| Method | Endpoint | Purpose |
| --- | --- | --- |
| POST | /api/v1/auth/register | สร้าง account/profile เบื้องต้น |
| POST | /api/v1/auth/verify-email | ยืนยัน email |
| POST | /api/v1/auth/login | เข้าสู่ระบบและสร้าง session cookie |
| POST | /api/v1/auth/logout | revoke session ปัจจุบัน |
| POST | /api/v1/auth/logout-all | revoke session ทุกอุปกรณ์ |
| POST | /api/v1/auth/forgot-password | ขอ reset password |
| POST | /api/v1/auth/reset-password | ตั้งรหัสผ่านใหม่ด้วย token |
| POST | /api/v1/auth/change-password | เปลี่ยนรหัสผ่านเมื่อ login อยู่ |
| GET | /api/v1/auth/me | ข้อมูลผู้ใช้ปัจจุบัน + context ที่เกี่ยวข้อง |
| GET | /api/v1/auth/sessions | รายการ session ที่ใช้งานอยู่ |
| DELETE | /api/v1/auth/sessions/{session_id} | revoke session เฉพาะตัว |
| POST | /api/v1/auth/force-logout | Admin force logout ตามสิทธิ์ |

### Auth Security Rules

- Session หลักเก็บใน Redis และอ้างอิงผ่าน httpOnly Cookie
- Cookie ต้องเป็น Secure ใน production และ SameSite=Lax
- ไม่เก็บ token สำคัญไว้ใน localStorage
- auth_sessions ใช้เก็บ session metadata / revoke history / device history
- Login failed ต้องบันทึกใน auth_login_attempts
- Security event สำคัญต้องบันทึกใน security_events
- password_changed_at อยู่ที่ auth_identities และใช้ invalidate token/session เก่าหลังเปลี่ยนรหัสผ่าน
- Email verification จำเป็นก่อนใช้งานจริง
## 9.3 Tenant / Workspace API

| Method | Endpoint | Purpose |
| --- | --- | --- |
| POST | /api/v1/workspaces/register | สมัคร workspace + owner account + seed data + email verification |
| GET | /api/v1/workspaces/check-slug?slug=... | ตรวจ slug ซ้ำ รูปแบบ และคำสงวน |
| GET | /api/v1/workspaces/me | รายการ workspace ที่ user มีสิทธิ์ |
| GET | /api/v1/workspace | ข้อมูล workspace ปัจจุบันหลัง Tenant Resolution |
| PATCH | /api/v1/workspace | แก้ Workspace Profile |
| GET/POST/PATCH/DELETE | /api/v1/workspace/members | จัดการสมาชิก workspace |
| GET/PATCH | /api/v1/workspace/configs/{config_key} | จัดการ workspace config |
| GET/POST/PATCH/DELETE | /api/v1/workspace/master-data | จัดการ workspace master data |
| GET | /api/v1/workspace/usage | ดู usage/limit |
| POST | /api/v1/workspace/request-deletion | Owner ขอปิด workspace → pending deletion |
| POST | /api/v1/workspace/cancel-deletion | ยกเลิก pending deletion |

- Workspace API ต้องผ่าน Tenant Resolution
- tenant_id มาจาก backend context เท่านั้น
- ทุก query ต้อง filter tenant_id
- ต้องตรวจ membership status = active
- DELETE สมาชิก/master/workspace data ไม่ควรหมายถึง hard delete ทันที ต้องใช้ remove/inactive/soft delete ตามกรณี
## 9.4 Project API

| Method | Endpoint | Purpose |
| --- | --- | --- |
| GET | /api/v1/workspace/projects | List Projects พร้อม filter/pagination |
| POST | /api/v1/workspace/projects | Create Project |
| GET | /api/v1/workspace/projects/{project_id} | Get Project Detail |
| PATCH | /api/v1/workspace/projects/{project_id} | Update Project Profile |
| POST | /api/v1/workspace/projects/{project_id}/archive | Archive Project |
| POST | /api/v1/workspace/projects/{project_id}/restore | Restore Project |
| DELETE | /api/v1/workspace/projects/{project_id} | Delete/soft delete แบบมีเงื่อนไข |
| GET/POST/PATCH/DELETE | /api/v1/workspace/projects/{project_id}/members | Project Members |
| GET/PATCH | /api/v1/workspace/projects/{project_id}/settings | Project Settings |

### Create Project Rules

- set tenant_id และ workspace_id จาก backend context
- set created_by จาก user ปัจจุบัน
- default project_status = draft และ priority = medium
- generate หรือ validate project_code ตาม Workspace Policy
- เพิ่มคนสร้างเป็น Project Owner เริ่มต้น
- seed project task workflow จาก Workspace Template
## 9.5 Task API

| Method | Endpoint | Purpose |
| --- | --- | --- |
| GET | /api/v1/workspace/projects/{project_id}/tasks | List Tasks พร้อม filter/pagination |
| POST | /api/v1/workspace/projects/{project_id}/tasks | Create Task |
| GET | /api/v1/workspace/projects/{project_id}/tasks/{task_id} | Get Task Detail |
| PATCH | /api/v1/workspace/projects/{project_id}/tasks/{task_id} | Update Task |
| DELETE | /api/v1/workspace/projects/{project_id}/tasks/{task_id} | Soft Delete Task |
| POST | /api/v1/workspace/projects/{project_id}/tasks/{task_id}/status | Change Status |
| POST | /api/v1/workspace/projects/{project_id}/tasks/{task_id}/reassign | Reassign Task |
| GET/POST/DELETE | .../tasks/{task_id}/watchers | Watchers |
| GET/POST/PATCH/DELETE | .../tasks/{task_id}/checklists | Checklist |
| GET/POST/PATCH/DELETE | .../tasks/{task_id}/comments | Comments |
| GET/POST/DELETE | .../tasks/{task_id}/relations | Relations |
| POST/DELETE | .../tasks/{task_id}/attachments | Attachments ผ่าน File Center/Reference Tracking |

- ทุก query ต้องใช้ tenant_id + project_id + task_id; ห้าม query ด้วย task_id เดี่ยว
- เปลี่ยน assignee ต้องเก็บ task_reassign_logs และส่ง notification
- attachment คือ relation ไม่ใช่ copy file
## 9.6 Timeline / Deliverable API

| Method | Endpoint | Purpose |
| --- | --- | --- |
| GET | /api/v1/workspace/projects/{project_id}/timeline | List Timeline Items |
| POST | /api/v1/workspace/projects/{project_id}/timeline | Create Timeline Item |
| GET | /api/v1/workspace/projects/{project_id}/timeline/{timeline_item_id} | Get Timeline Detail |
| PATCH | /api/v1/workspace/projects/{project_id}/timeline/{timeline_item_id} | Update Timeline Item |
| DELETE | /api/v1/workspace/projects/{project_id}/timeline/{timeline_item_id} | Delete/Cancel Timeline Item |
| GET/POST/PATCH/DELETE | .../timeline/{timeline_item_id}/delivery-attempts | Delivery Attempts สำหรับ Type = Deliverable |
| GET/POST/PATCH/DELETE | .../timeline/{timeline_item_id}/attendees | Meeting Attendees สำหรับ Meeting/Appointment |
| GET/POST/DELETE | .../timeline/{timeline_item_id}/relations | Timeline Relations |

- Timeline เป็นศูนย์กลางวันสำคัญของ Project
- Deliverable เป็น Timeline Type หนึ่ง และ Delivery Attempt เป็น child ของ Timeline Item
- Timeline ไม่มี status ให้ user เลือกใน MVP; ใช้ indicator จากวันที่แทน
- Reminder ใช้ planned_date + remind_before_days
## 9.7 Finance API

| Method | Endpoint | Purpose |
| --- | --- | --- |
| GET | /api/v1/workspace/projects/{project_id}/finance/overview | Finance Overview |
| GET/PATCH | /api/v1/workspace/projects/{project_id}/finance/budget | Project Budget |
| GET/POST | /api/v1/workspace/projects/{project_id}/finance/entries | List/Create Finance Entries |
| GET/PATCH/DELETE | /api/v1/workspace/projects/{project_id}/finance/entries/{entry_id} | Detail/Update/Delete Finance Entry |
| GET/POST/DELETE | .../finance/entries/{entry_id}/attachments | Finance Entry Attachments |
| GET/PATCH | /api/v1/workspace/projects/{project_id}/finance/visibility | Finance Visibility |

- Finance API ต้องตรวจ Finance Visibility ทุกครั้ง
- attachment ต้องผ่าน File Center / Reference Tracking
- ยังไม่ทำ Invoice, Payment Tracking, Tax, Withholding Tax, Accounting และ P/L เต็มระบบใน MVP
## 9.8 Document / File API

| Method | Endpoint | Purpose |
| --- | --- | --- |
| GET | /api/v1/workspace/projects/{project_id}/files | List Folder/File |
| POST | /api/v1/workspace/projects/{project_id}/folders | Create Folder |
| PATCH/DELETE | /api/v1/workspace/projects/{project_id}/folders/{folder_id} | Update/Move/Delete Folder |
| POST | /api/v1/workspace/projects/{project_id}/files/upload-request | Request Upload |
| POST | /api/v1/workspace/projects/{project_id}/files/complete-upload | Complete Upload |
| GET | /api/v1/workspace/projects/{project_id}/files/{file_id} | Get File Detail |
| GET | /api/v1/workspace/projects/{project_id}/files/{file_id}/download-url | Signed/Temporary Download URL |
| PATCH/DELETE | /api/v1/workspace/projects/{project_id}/files/{file_id} | Update Metadata / Soft Delete |
| POST | /api/v1/workspace/projects/{project_id}/files/{file_id}/restore | Restore File |
| GET/POST/PATCH/DELETE | /api/v1/workspace/projects/{project_id}/file-links | External Links |
| POST/DELETE | /api/v1/workspace/projects/{project_id}/files/{file_id}/relations | File Relations/Attachments |

### File API Security Rules

- ห้ามเก็บหรือส่ง Full URL ถาวรเป็นหลัก
- การเปิดไฟล์ต้องผ่าน backend เพื่อตรวจสิทธิ์
- backend สร้าง signed URL / temporary URL ที่หมดอายุ
- delete จาก UI เป็น soft delete; object จริงถูกลบตอน cleanup / hard delete ตาม policy
- attachment คือ relation ไม่ใช่ copy file
## 9.9 Notification / Announcement API

| Group | Endpoint | Purpose |
| --- | --- | --- |
| Notification | GET /api/v1/workspace/notifications | List Notifications |
| Notification | GET /api/v1/workspace/notifications/unread-count | Unread Count |
| Notification | POST /api/v1/workspace/notifications/{id}/read | Mark as Read |
| Notification | POST /api/v1/workspace/notifications/mark-all-read | Mark All as Read |
| Notification | POST /api/v1/workspace/notifications/{id}/archive | Archive Notification |
| Announcement | GET/POST /api/v1/workspace/announcements | List/Create Announcement |
| Announcement | GET/PATCH/DELETE /api/v1/workspace/announcements/{id} | Detail/Update/Delete Announcement |
| Acknowledgement | POST /api/v1/workspace/announcements/{id}/acknowledge | Acknowledge Required Announcement |
| Acknowledgement | GET /api/v1/workspace/announcements/{id}/acknowledgements | Acknowledgement Status |
| Preferences | GET/PATCH /api/v1/workspace/notification-preferences | Notification Preferences เบา ๆ |

- Notification = In-app ราย user เป็นหลัก
- Email เป็น delivery เสริมสำหรับเรื่องสำคัญ
- Announcement ไม่ใช่ Chat
- Required Announcement ต้อง Acknowledge แยกจาก Mark as Read
- Mark all as read ห้ามนับเป็น Acknowledge
- In-app Notification และ Required Announcement ปิดทั้งหมดไม่ได้
## 9.10 Calendar / Report API

| Group | Endpoint | Purpose |
| --- | --- | --- |
| Calendar | GET /api/v1/workspace/calendar | Workspace Calendar |
| Calendar | GET /api/v1/workspace/projects/{project_id}/calendar | Project Calendar |
| Dashboard | GET /api/v1/workspace/dashboard | Workspace Dashboard |
| Dashboard | GET /api/v1/workspace/projects/{project_id}/overview | Project Overview |
| Report | GET /api/v1/workspace/reports/projects-summary | Project Summary |
| Report | GET /api/v1/workspace/reports/tasks-summary | Task Summary |
| Report | GET /api/v1/workspace/reports/timeline-summary | Timeline Summary |
| Report | GET /api/v1/workspace/reports/overdue | Overdue Report |
| Report | GET /api/v1/workspace/reports/finance-summary | Finance Summary |
| Report | GET /api/v1/workspace/reports/usage | Usage / Storage |

- Calendar ดึงข้อมูลจาก Task + Timeline เป็นหลัก
- Report ต้องกรองตามสิทธิ์ของ user
- Finance Report ต้องผ่าน Finance Visibility
- ยังไม่ทำ BI / Custom Report Builder ใน MVP
## 9.11 Platform Admin API

| Group | Endpoint | Purpose |
| --- | --- | --- |
| Workspace | GET /api/v1/platform/workspaces | List Workspaces |
| Workspace | GET/PATCH /api/v1/platform/workspaces/{workspace_id} | Get/Update Workspace |
| Lifecycle | POST /api/v1/platform/workspaces/{workspace_id}/suspend | Suspend Workspace |
| Lifecycle | POST /api/v1/platform/workspaces/{workspace_id}/restore | Restore Workspace |
| Lifecycle | POST /api/v1/platform/workspaces/{workspace_id}/hard-delete | Hard Delete Workspace |
| Config | GET/PATCH /api/v1/platform/configs | Platform Config |
| Master | GET/POST/PATCH/DELETE /api/v1/platform/master-data | App Master Data |
| Reference | GET/POST/PATCH/DELETE /api/v1/platform/reference/locations | Reference Locations |
| Cleanup | GET /api/v1/platform/cleanup-candidates | Cleanup Candidates |
| Cleanup | POST /api/v1/platform/files/cleanup | File Cleanup |
| Audit | GET /api/v1/platform/audit-logs | Audit Logs |
| Security | GET /api/v1/platform/security-events | Security Events |

- ใช้ได้เฉพาะ Platform Admin / System Admin / Support Admin ตามสิทธิ์
- ต้องไม่ใช้แทน Workspace API
- Hard Delete เป็น action ความเสี่ยงสูง ต้องมี log และตรวจสอบก่อนเสมอ
- API list ทั้งหมดต้องมี pagination และ filter
## 9.12 Response / Error / Pagination Standard

หมายเหตุ: Response Standard ในเอกสารนี้เป็นแนวทางเบื้องต้นสำหรับทีมพัฒนา รายละเอียดเชิงเทคนิคบางส่วนต้องยืนยันอีกครั้งตอนเริ่ม dev จริง

### Success Object

```text
{
"data": {
"id": "xxx",
"name": "Project A"
}
}
```

### Success List

```text
{
"data": {
"items": [
{
"id": "xxx",
"name": "Project A"
}
],
"pagination": {
"page": 1,
"limit": 20,
"total": 120,
"total_pages": 6
}
}
}
```

### Error

```text
{
"error": {
"code": "VALIDATION_ERROR",
"message": "ข้อมูลไม่ถูกต้อง",
"details": {
"project_name": "กรุณากรอกชื่อโปรเจค"
}
}
}
```

- success response ต้องอยู่ใต้ root key = data
- error response ต้องอยู่ใต้ root key = error
- list ใช้ data.items และ pagination อยู่ใน data.pagination
- error.details อาจเป็น object หรือ array รอยืนยันตอน dev จริง
- request_id อาจอยู่ header หรือ body รอยืนยันตอน dev จริง

| HTTP Status | ความหมาย |
| --- | --- |
| 200 | สำเร็จทั่วไป |
| 201 | สร้างข้อมูลสำเร็จ |
| 400 | request ผิดรูปแบบ |
| 401 | ยังไม่ได้ login / token ผิดหรือหมดอายุ |
| 403 | ไม่มีสิทธิ์ทำรายการ |
| 404 | ไม่พบข้อมูล หรือไม่ต้องการเปิดเผยว่าข้อมูลมีอยู่ |
| 409 | ข้อมูลชนกัน เช่น slug/code ซ้ำ |
| 422 | validation error |
| 429 | request ถี่เกินไป |
| 500 | server error |

## 9.13 API Security / Rate Limit / Logging

- ทุก API ที่ต้อง login ต้องตรวจ session จาก httpOnly Cookie / Redis session store
- Workspace API ต้องผ่าน Tenant Context + Permission Guard
- ต้องมี rate limit สำหรับ endpoint เสี่ยง
- ต้องมี request_id สำหรับ trace/debug
- ต้อง log request เพื่อ debug แต่ห้าม log sensitive data
- Activity Log และ Security Event ต้องแยกกัน
- File API ต้องใช้ signed URL และห้ามส่ง / เก็บ / log full URL ถาวร

| Endpoint เสี่ยง | เหตุผล |
| --- | --- |
| Login | ป้องกัน brute force |
| Forgot / Reset Password | ป้องกัน abuse และ email spam |
| Email Verification Resend | ป้องกัน email spam |
| Register Workspace | ป้องกัน spam workspace |
| File Upload Request | ป้องกันใช้ storage เกินผิดปกติ |

### ห้าม Log ข้อมูล Sensitive

- password
- session id / session cookie
- token / secret ที่ใช้ยืนยันตัวตนหรือ reset password
- reset token
- full signed file URL
- sensitive config value
## สรุป Endpoint Group สำคัญ

```text
Auth: /api/v1/auth/...
Workspace: /api/v1/workspaces/... และ /api/v1/workspace/...
Project: /api/v1/workspace/projects/...
Task: /api/v1/workspace/projects/{project_id}/tasks/...
Timeline: /api/v1/workspace/projects/{project_id}/timeline/...
Finance: /api/v1/workspace/projects/{project_id}/finance/...
File: /api/v1/workspace/projects/{project_id}/files/...
Notification: /api/v1/workspace/notifications และ /api/v1/workspace/announcements
Report: /api/v1/workspace/calendar, dashboard, reports/...
Platform: /api/v1/platform/...
```

## จุดที่ต้องระวังตอนทำจริง

Workspace API ห้ามใส่ tenant_id ใน URL

tenant_id ต้องมาจาก Tenant Context เท่านั้น

ทุก query ต้องใช้ tenant_id จาก backend

Auth บอกว่า user คือใคร แต่ Permission Guard บอกว่าทำอะไรได้ที่ไหน

File API ห้ามส่ง Full URL ถาวร

List API ทุกตัวต้องมี pagination

Response Standard เป็นแนวทาง ต้องยืนยันรายละเอียดตอน dev จริง

DELETE หลายจุดไม่ได้แปลว่า hard delete ต้องตีความตาม business rule

Action สำคัญต้องเขียน Activity Log / Security Event

ห้าม log sensitive data
