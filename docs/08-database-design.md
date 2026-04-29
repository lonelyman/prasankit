# ชุดที่ 8: Database Design

_Project Platform - Multi-tenant Project Management System_

สถานะ: Confirmed / ใช้ต่อจากเอกสารชุดที่ 1-7
ขอบเขต: เอกสารนี้ครอบคลุมเฉพาะ Database Design: ตารางหลัก, ตารางย่อย, ความสัมพันธ์, tenant_id, index/search, soft delete, audit และ security rule ของไฟล์
ไม่รวมในชุดนี้: API, UI รายหน้า, Docker/Deploy, โค้ดจริง

## 8.1 Database Design Principle

หลักการที่ล็อกสำหรับฐานข้อมูล:

1. ใช้ Shared Database + tenant_id
1. ทุก Workspace data table ต้องมี tenant_id
1. tenant_id ต้องมาจาก backend Tenant Context ไม่รับจาก frontend เป็นหลัก
1. ใช้ UUID v7 เป็น primary key โดยให้ Go application เป็นผู้สร้างค่า ID ก่อน insert
1. มี audit fields มาตรฐาน เช่น created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
1. มี soft delete ใน table สำคัญ
1. แยก Snapshot Value / Live Reference ให้ชัด
1. ใช้ Reference Tracking สำหรับข้อมูลที่เชื่อมข้าม module
1. ใช้ Main Table + Child Table เพื่อลด null จำนวนมากในตารางหลัก
1. ออกแบบให้ขยาย module / field / config เพิ่มในอนาคตได้

### Primary Key Strategy

- Database column type ใช้ `UUID`
- Application สร้าง ID ด้วย UUID v7 ก่อน insert เพื่อให้เรียงตามเวลาและเหมาะกับ index มากกว่า UUID v4 แบบสุ่มล้วน
- ห้ามพึ่ง `DEFAULT gen_random_uuid()` สำหรับ primary key ของ table ใหม่ เพราะเป็น UUID v4 และทำให้ไม่รู้ชัดว่า ID ถูกสร้างจากชั้นไหน
- PostgreSQL extension `pgcrypto` ยังใช้ได้สำหรับงาน crypto/utility อื่น แต่ไม่ใช่ primary key generator หลัก
### Main Table + Child Table

| ประเภท | ความหมาย | ตัวอย่าง |
| --- | --- | --- |
| Main Table | เก็บข้อมูลหลักที่จำเป็นและใช้บ่อย | projects, tasks, timeline_items, finance_entries |
| Child Table | เก็บข้อมูลหลายรายการ / optional / ประวัติ / เฉพาะ use case | task_watchers, task_comments, delivery_attempts, file_relations |

หลักสำคัญ: หลีกเลี่ยงการเพิ่ม column optional จำนวนมากในตารางหลักจนเกิด null พร่ำเพรื่อ

### Index / Search Principle

1. Column ที่ filter บ่อยและ join บ่อยต้องมี index
1. tenant_id ต้องอยู่ใน composite index ของ table ใหญ่
1. หลีกเลี่ยง LIKE '%keyword%' กับ table ใหญ่ เช่น projects, tasks, project_files, activity_logs, notifications
1. ทุก list page ต้องมี pagination
1. MVP ใช้ prefix search / filter + pagination / index column ที่ค้นบ่อย
1. อนาคตค่อยเพิ่ม PostgreSQL Full Text Search, trigram index, search vector หรือ external search engine
### Unique + Soft Delete Principle

ทุก unique field ต้องพิจารณาร่วมกับ soft delete ว่า record ที่ถูกลบแล้วควรนับซ้ำหรือไม่

CREATE UNIQUE INDEX uq_projects_code_active
ON projects (tenant_id, workspace_id, project_code)
WHERE deleted_at IS NULL;

1. Business name/code ที่ soft delete แล้วอาจนำกลับมาใช้ใหม่ได้ ให้ใช้ partial unique index
1. Audit / transaction / external reference สำคัญ ต้อง unique ถาวรแม้ record ถูก soft delete แล้ว
## 8.2 Core / Platform Tables

| Table | Purpose |
| --- | --- |
| platform_admins | สิทธิ์ผู้ดูแลระบบกลาง |
| platform_configs | Config ระดับ Platform |
| workspaces | แกนหลักของ tenant / workspace |
| reference_locations | ข้อมูลอ้างอิงที่อยู่สำหรับ autocomplete + free text |
| platform_deletion_logs | Log ก่อน/หลัง hard delete workspace |

### platform_admins

id
user_account_id
role -- platform_admin, system_admin, support_admin
status
created_at
created_by
updated_at
updated_by

หมายเหตุ: platform_admins ต้องอ้าง user_account_id เพราะ design จริงแยก user_profiles และ user_accounts

### platform_configs

id
config_key
config_value
value_type
description
is_sensitive
created_at
updated_at
updated_by

ตัวอย่าง config: workspace_approval_mode, default_usage_limit, demo_marker_policy, pending_deletion_period_default, default_notification_timing, file_cleanup_policy

### workspaces

id
tenant_id
workspace_name
slug
mode -- demo, production
status -- active, suspended, pending_deletion, deleted
contact_email
owner_user_account_id
email_verified_required
created_at
created_by
updated_at
updated_by
pending_deletion_at
deleted_at
hard_deleted_at

Index / Constraint สำคัญ:

1. unique(tenant_id)
1. slug ต้อง unique เฉพาะ active/non-deleted ตาม policy
1. status ใช้คุม access rule ของ workspace
### reference_locations

id
country_name
province_name
district_name
subdistrict_name
postal_code
status
aliases
created_at
updated_at

ใช้เป็นตัวช่วย autocomplete ไม่บังคับ FK แข็งในข้อมูลธุรกิจ ข้อมูลธุรกิจจริงเก็บ string snapshot + optional ref_id

### platform_deletion_logs

id
tenant_id
workspace_id
workspace_name
workspace_slug
deleted_by
deleted_at
reason
data_summary_json
created_at

## 8.3 Workspace / Tenant Tables

| Table | Purpose |
| --- | --- |
| workspace_roles | App Master สำหรับ role ของสมาชิก workspace |
| workspace_memberships | ความสัมพันธ์ user/profile กับ workspace และ role |
| workspace_configs | Config ระดับ Workspace |
| workspace_master_items | Master Data ระดับ Workspace แบบ generic |
| workspace_usage_snapshots | Usage snapshot เช่น project, member, file, storage |
| workspace_cleanup_candidates | Workspace ที่ควรตรวจสอบเพื่อ cleanup |

### workspace_roles

id
code -- owner, admin, executive, user
name
description
sort_order
is_system
status -- active, deprecated
created_at
updated_at

หลักการ:

1. เป็น App Master / seed กลางของระบบสำหรับ permission role
1. ตารางธุรกิจต้อง FK มาที่ `workspace_roles.id` ไม่เก็บ role เป็น free text
1. API ยังสามารถคืน `code` เช่น owner/admin ให้ frontend แสดงผลได้ แต่ backend ต้องใช้ FK เป็น source หลัก
1. Role ที่ถูกใช้งานแล้วไม่ควร hard delete ให้ใช้ `deprecated` แทน

### workspace_memberships

id
tenant_id
workspace_id
profile_id
user_account_id -- nullable ได้ หากเป็น profile ที่ยังไม่มี account
workspace_role_id -- FK -> workspace_roles.id
status -- active, removed, suspended
status_reason
joined_at
removed_at
suspended_at
created_at
created_by
updated_at
updated_by

หลักการ:

1. ต้องรองรับ user กลับมาเป็นสมาชิกใหม่ได้โดยไม่ทำลายประวัติเดิม
1. Removed -> Active ได้ แต่ต้องมี audit log
1. User Account Status แยกจาก Workspace Membership Status
1. Role เป็น Live Reference ผ่าน `workspace_role_id`; lifecycle status ยังเป็น system state ที่ควบคุมด้วย constraint ได้
### workspace_configs

id
tenant_id
workspace_id
config_key
config_value
value_type
description
is_sensitive
created_at
created_by
updated_at
updated_by

### workspace_master_items

id
tenant_id
workspace_id
master_type
code
name
description
sort_order
status
is_system_default
created_at
created_by
updated_at
updated_by
deleted_at
deleted_by

ตัวอย่าง master_type: project_position, task_type, finance_category, finance_type, penalty_adjustment_type, priority, timeline_type, document_folder_template

### workspace_usage_snapshots

id
tenant_id
workspace_id
project_count
member_count
file_count
storage_used_bytes
active_task_count
snapshot_date
created_at

### workspace_cleanup_candidates

id
tenant_id
workspace_id
reason -- pending_verification_expired, never_logged_in, no_activity, deleted_ready_for_cleanup
status
detected_at
reviewed_by
reviewed_at
note

## 8.4 User / Profile / Auth Tables

หลักการสำคัญ: User Account ≠ Workspace Profile

| Concept | Meaning |
| --- | --- |
| User Account | บัญชี login กลางของคนหนึ่งคน ไม่ผูกกับ Workspace เดียว |
| User Profile | ข้อมูลบุคคล/โปรไฟล์ภายใน Workspace ใช้แสดงและอ้างอิงงาน อาจยัง login ไม่ได้ |
| Workspace Membership | ความสัมพันธ์ระหว่าง User Account กับ Workspace และ Role ใน Workspace นั้น |

### user_profiles

id
tenant_id
workspace_id
display_name
email
phone
profile_type -- internal_user, external_contact, customer_contact, placeholder
status
created_at
created_by
updated_at
updated_by
deleted_at
deleted_by

### user_accounts

id
primary_email
status -- active, suspended, disabled
status_reason -- resigned, account_restricted, other
last_login_at
created_at
updated_at
disabled_at

หลักการ:

1. 1 profile อาจไม่มี account ได้
1. 1 account สามารถอยู่หลาย Workspace ผ่าน workspace_memberships
1. 1 account อาจมีหลาย Workspace Profile ตาม Workspace ที่เข้าร่วม
1. user_accounts ไม่ควรมี tenant_id / workspace_id และไม่ควรผูก profile_id เดียวแบบตายตัว
1. user_accounts ไม่ใช่ login identity โดยตรง; วิธี login ต้องอยู่ใน auth_identities

### auth_identities

id
user_account_id
identity_type -- email_password, oauth
provider -- email, google, facebook, microsoft, line, github
provider_user_id -- ใช้กับ oauth เท่านั้น
email
email_verified_at
password_hash -- ใช้กับ email_password เท่านั้น
password_changed_at
last_used_at
created_at
updated_at
deleted_at

หลักการ:

1. email ไม่ใช่ source of truth ของ social identity
1. OAuth identity ต้องหาโดย provider + provider_user_id
1. email/password login เป็น identity ประเภทหนึ่ง ไม่ใช่ logic พิเศษใน user_accounts
1. Google/Facebook ที่ใช้ email เดียวกันต้องไม่ auto-merge แบบเงียบ ๆ; ต้องผ่าน flow link account ที่ชัดเจน
1. password_changed_at อยู่ที่ auth_identities เพื่อ invalidate session/token เฉพาะ identity ที่เกี่ยวข้อง
### auth_sessions

id
user_account_id
session_id_hash
device_info
ip_address
user_agent
status -- active, revoked, expired
created_at
last_used_at
expires_at
revoked_at
revoked_by
revoked_reason

หลัก Session:

1. MVP ใช้ Session-based Auth โดย session หลักเก็บใน Redis
1. Browser ใช้ httpOnly Cookie และไม่เก็บ token สำคัญใน localStorage
1. auth_sessions ใช้เก็บ metadata ของ session, device history, revoke history และช่วย logout/logout all devices
1. ไม่ใช้ token-based auth เป็น auth หลักใน MVP
1. รองรับ logout, logout all devices, admin force logout, revoke session เมื่อ suspended/disabled
### auth_login_attempts

id
email
ip_address
user_agent
success
failed_reason
attempted_at

### auth_password_reset_tokens

id
user_account_id
token_hash
expires_at
used_at
created_at
ip_address
user_agent

### auth_email_verification_tokens

id
user_account_id
token_hash
expires_at
used_at
created_at

### Password Changed Tracking

auth_identities ต้องมี password_changed_at สำหรับ identity ที่มี password

ใช้เพื่อ:

1. ตรวจว่ารหัสผ่านถูกเปลี่ยนล่าสุดเมื่อไหร่
1. invalidate session/token เก่าหลังเปลี่ยนรหัสผ่าน
1. บังคับ login ใหม่ทุกอุปกรณ์
1. ใช้ audit/security report
Token validation:

ถ้า token_issued_at < password_changed_at
= token นี้ใช้ไม่ได้แล้ว

## 8.5 Project Tables

| Table | Purpose |
| --- | --- |
| project_roles | App Master สำหรับ role ของสมาชิก project |
| project_priorities | App Master สำหรับ priority ของ project |
| projects | ข้อมูลหัวโปรเจคหลัก |
| project_members | คนใน Project |
| project_member_positions | ตำแหน่งจริงของสมาชิกใน Project แบบหลายค่า |
| project_code_counters | รันเลข Project Code |
| project_archives | ประวัติ archive / restore / delete project |

### project_roles

id
code -- project_owner, project_manager, member, finance, viewer
name
description
sort_order
is_system
status -- active, deprecated
created_at
updated_at

### project_priorities

id
code -- low, medium, high
name
sort_order
is_system
status -- active, deprecated
created_at
updated_at

### projects

id
tenant_id
workspace_id
project_code
project_name
project_type -- internal, client
project_status -- draft, planning, proposal, active, closing, maintenance, closed, archived
priority_id -- FK -> project_priorities.id
description
client_or_requesting_unit
scope_or_objective
created_by
created_at
updated_by
updated_at
deleted_by
deleted_at
archived_at

หลักการ:

1. Project Owner อยู่ใน project_members ไม่ใช่ field ตายตัวใน projects
1. Created By เป็น audit ไม่ใช่ owner
1. Project Code unique ภายใน workspace เฉพาะ record ที่ยังไม่ deleted
### project_members

id
tenant_id
workspace_id
project_id
profile_id
user_account_id -- nullable ได้
project_role_id -- FK -> project_roles.id
status -- active, removed
joined_at
removed_at
created_by
created_at
updated_by
updated_at

Project ต้องมี Project Owner อย่างน้อย 1 คน และห้าม remove Project Owner คนสุดท้าย

หลักการ:

1. Project Role เป็น Live Reference ผ่าน `project_role_id`
1. Project Priority เป็น Live Reference ผ่าน `priority_id`
1. Lifecycle เช่น project_status และ member status ยังเป็น system state ที่ควบคุมด้วย constraint ได้

### project_member_positions

id
tenant_id
workspace_id
project_id
project_member_id
position_id
created_at
created_by

Project Role = สิทธิ์ / Project Position = หน้าที่จริง ใช้แสดงผล; 1 project member มี role หลัก 1 role แต่มี position ได้หลายรายการ

### project_code_counters

id
tenant_id
workspace_id
prefix
year
running_no
number_length
last_generated_at
created_at
updated_at

### project_archives

id
tenant_id
workspace_id
project_id
action -- archive, restore, delete
reason
acted_by
acted_at

## 8.6 Task Tables

| Table | Purpose |
| --- | --- |
| tasks | ตารางหลักของงาน |
| task_watchers | ผู้ติดตาม/ผู้เกี่ยวข้อง |
| task_checklists | Checklist ภายใน task |
| task_comments | Comment/update ความคืบหน้า |
| task_reassign_logs | ประวัติการส่งต่องาน/โอนงาน |
| task_relations | ความสัมพันธ์ระหว่าง task กับข้อมูลอื่น |
| task_counters | รันเลข task ภายใน project |

### tasks

id
tenant_id
workspace_id
project_id
task_no
task_title
task_type_id
status_id
priority_id
assignee_member_id
description
start_date
due_date
completed_date
created_by
created_at
updated_by
updated_at
deleted_by
deleted_at

หลักการ:

1. Task อยู่ใต้ Project
1. Assignee หลักมีได้ 1 คน
1. Watcher / Participant แยกตารางย่อย
1. Task Type เป็น Workspace Master
1. Status เป็น Project Task Workflow
### task_watchers / task_checklists / task_comments

task_watchers:
id, tenant_id, workspace_id, project_id, task_id, project_member_id, created_at, created_by

task_checklists:
id, tenant_id, workspace_id, project_id, task_id, item_text, is_checked, sort_order, created_at, created_by, updated_at, updated_by

task_comments:
id, tenant_id, workspace_id, project_id, task_id, comment_text, created_by, created_at, updated_at, deleted_at

### task_reassign_logs / task_relations / task_counters

task_reassign_logs:
id, tenant_id, workspace_id, project_id, task_id, from_assignee_member_id, to_assignee_member_id, reason, reassigned_by, reassigned_at

task_relations:
id, tenant_id, workspace_id, project_id, task_id, related_type, related_id, relation_type, created_at, created_by

task_counters:
id, tenant_id, workspace_id, project_id, prefix, running_no, last_generated_at, created_at, updated_at

## 8.7 Timeline / Deliverable Tables

| Table | Purpose |
| --- | --- |
| timeline_items | ตารางหลักของวันสำคัญใน Project |
| timeline_item_field_values | Field เฉพาะตาม Timeline Type |
| delivery_attempts | รอบการส่งมอบของ Timeline Type = Deliverable |
| meeting_attendees | ผู้เข้าร่วมของ Timeline Type = Meeting |
| timeline_relations | Relation optional ของ Timeline |
| timeline_type_fields | Future: นิยาม field เฉพาะตาม type |

### timeline_items

id
tenant_id
workspace_id
project_id
timeline_type_id
title
planned_date
actual_date
responsible_member_id
note
remind_before_days
created_by
created_at
updated_by
updated_at
deleted_by
deleted_at
cancelled_at
cancelled_by

หลักการ:

1. ไม่มี status ให้ user เลือกใน MVP
1. ระบบแสดง indicator จากวันที่ เช่น ใกล้ถึงกำหนด / เลยกำหนด
1. remind_before_days เลือกได้ 1 ค่าใน MVP
### timeline_item_field_values

id
tenant_id
workspace_id
project_id
timeline_item_id
field_key
field_value_text
field_value_number
field_value_date
field_value_json
created_at
updated_at

### delivery_attempts

id
tenant_id
workspace_id
project_id
timeline_item_id
attempt_no
submitted_date
submitted_by_member_id
result -- pending_review, accepted, rejected, accepted_with_condition
reviewed_date
reviewed_by_member_id
remark
created_at
created_by
updated_at
updated_by
deleted_at
deleted_by

### meeting_attendees / timeline_relations

meeting_attendees:
id, tenant_id, workspace_id, project_id, timeline_item_id, profile_id, project_member_id, attendance_status, created_at, created_by

timeline_relations:
id, tenant_id, workspace_id, project_id, timeline_item_id, related_type, related_id, relation_type, created_at, created_by

## 8.8 Finance Tables

| Table | Purpose |
| --- | --- |
| project_budgets | งบประมาณระดับ Project |
| finance_entries | รายการการเงินหลักแบบยืดหยุ่น |
| finance_entry_attachments | เอกสารประกอบรายการการเงิน |
| finance_visibility_rules | คุมสิทธิ์การมองเห็น Finance |

### project_budgets

id
tenant_id
workspace_id
project_id
budget_amount
currency
note
created_by
created_at
updated_by
updated_at
deleted_by
deleted_at

### finance_entries

id
tenant_id
workspace_id
project_id
entry_type_id
category_id
entry_date
title
description
amount
currency
recorded_by_member_id
note
created_by
created_at
updated_by
updated_at
deleted_by
deleted_at

### finance_entry_attachments / finance_visibility_rules

finance_entry_attachments:
id, tenant_id, workspace_id, project_id, finance_entry_id, file_id, created_by, created_at

finance_visibility_rules:
id, tenant_id, workspace_id, project_id, visibility_mode, created_by, created_at, updated_by, updated_at

visibility_mode:
owner_manager_finance, finance_only, custom

ยังไม่ทำใน MVP-0: Finance เต็มรูปแบบ

ยังไม่ทำใน MVP-1: Invoice, Payment Tracking, ภาษี, หัก ณ ที่จ่าย, บัญชีลูกหนี้/เจ้าหนี้, P/L เต็มระบบ, เชื่อม Deliverable กับ Finance โดยตรง

## 8.9 Document / File Tables

| Table | Purpose |
| --- | --- |
| project_folders | Folder/Subfolder ภายใน Project |
| project_files | Metadata ของไฟล์ |
| file_links | External Link ใน File Center |
| file_lifecycle_logs | ประวัติ lifecycle ของไฟล์ |
| file_cleanup_candidates | ไฟล์ที่ควร cleanup |
| file_relations | เชื่อมไฟล์กับข้อมูลอื่น |

### project_folders

id
tenant_id
workspace_id
project_id
parent_folder_id
folder_name
sort_order
created_by
created_at
updated_by
updated_at
deleted_by
deleted_at
archived_at

รองรับ folder ซ้อน folder, Max depth เริ่มต้น = 5, ย้ายได้ภายใน Project เดียวกัน, ยังไม่รองรับย้ายข้าม Project ใน MVP

### project_files

id
tenant_id
workspace_id
project_id
folder_id
file_name
file_ext
mime_type
file_size_bytes
storage_provider
bucket_key
object_key
file_category -- business_document, replaceable_media
lifecycle_status -- active, soft_deleted, archived, replaced, orphan, hard_deleted
uploaded_by
uploaded_at
deleted_by
deleted_at
archived_at

### file_links / file_lifecycle_logs / file_cleanup_candidates / file_relations

file_links:
id, tenant_id, workspace_id, project_id, folder_id, title, url, link_type, description, created_by, created_at, updated_by, updated_at, deleted_by, deleted_at

file_lifecycle_logs:
id, tenant_id, workspace_id, project_id, file_id, action, from_status, to_status, reason, acted_by, acted_at

file_cleanup_candidates:
id, tenant_id, workspace_id, project_id, file_id, reason, status, detected_at, reviewed_by, reviewed_at, note

file_relations:
id, tenant_id, workspace_id, project_id, file_id, related_type, related_id, relation_type, created_by, created_at, deleted_by, deleted_at

### File URL Security Rule

1. Database ห้ามเก็บ Full URL / Public URL จริงของไฟล์เป็นค่าหลัก
1. ให้เก็บ storage_provider, bucket_key, object_key, file_name, mime_type, file_size_bytes
1. ผู้ใช้เปิดไฟล์ผ่าน backend: ตรวจสิทธิ์ก่อน แล้วสร้าง signed URL / temporary URL
1. URL ต้องมีอายุจำกัด
1. endpoint/domain ต้องเปลี่ยนได้ผ่าน config
1. object_key ไม่ควรเดาง่าย
1. ถ้า DB หลุด ต้องไม่สามารถเปิดไฟล์จริงได้ทันที
## 8.10 Notification / Announcement Tables

| Table | Purpose |
| --- | --- |
| notifications | In-app notification ราย user เป็นหลัก |
| notification_delivery_logs | Log การส่งออกช่องทางอื่น เช่น Email |
| announcements | ประกาศระดับ Workspace / Project |
| announcement_acknowledgements | การกดรับทราบประกาศสำคัญ |
| announcement_recipients | กลุ่มผู้รับประกาศ |

### notifications

id
tenant_id
workspace_id
user_account_id
notification_type
title
message
priority
related_type
related_id
status -- unread, read, archived
created_at
read_at
archived_at

### notification_delivery_logs

id
tenant_id
workspace_id
notification_id
channel -- email; future: line, slack, teams, push, sms
recipient
status
sent_at
failed_at
error_message
retry_count
created_at

### announcements

id
tenant_id
workspace_id
project_id -- nullable: null = workspace announcement
title
message
announcement_type
priority
is_pinned
require_acknowledgement
expires_at
created_by
created_at
updated_by
updated_at
deleted_by
deleted_at

### announcement_acknowledgements / announcement_recipients

announcement_acknowledgements:
id, tenant_id, workspace_id, announcement_id, user_account_id, acknowledged_at, ip_address, user_agent

announcement_recipients:
id, tenant_id, workspace_id, announcement_id, recipient_type, recipient_id, created_at, created_by

recipient_type:
workspace_all, project_members, role, user

ข้อห้ามสำคัญ: Mark all as read ต้องไม่ถือว่าเป็นการ Acknowledge ประกาศสำคัญ

## 8.11 Master / Config Tables

| Table | Purpose |
| --- | --- |
| app_master_items | App Master / seed กลางระบบ |
| workspace_master_items | Workspace Master |
| project_master_items | Project Master เฉพาะที่จำเป็น เช่น Task Workflow |
| platform_configs | Config ระดับ Platform |
| workspace_configs | Config ระดับ Workspace |
| project_configs | Config ระดับ Project |
| master_seed_logs | ประวัติการ seed/copy master |

### app_master_items / workspace_master_items / project_master_items

common fields:
id
tenant_id / workspace_id / project_id ตามระดับของ table
master_type
code
name
description
sort_order
status
is_system_default
created_at
created_by
updated_at
updated_by
deleted_at
deleted_by

### platform_configs / workspace_configs / project_configs

common fields:
id
tenant_id / workspace_id / project_id ตามระดับของ table
config_key
config_value
value_type
description
is_sensitive
created_at
created_by
updated_at
updated_by

### master_seed_logs

id
source_level
source_id
target_level
target_id
seed_type
seeded_by
seeded_at
summary_json

หลักการ:

1. App Master เปลี่ยนภายหลังไม่เปลี่ยน Workspace Master เก่าอัตโนมัติ
1. Master/Config ทุกตัวต้องระบุ Snapshot Value หรือ Live Reference
1. Master code/name ที่ soft delete แล้วต้องพิจารณา partial unique index
## 8.12 Audit / Reference Tracking Tables

| Table | Purpose |
| --- | --- |
| activity_logs | เก็บว่าใครทำอะไร เมื่อไหร่ กับข้อมูลไหน |
| reference_links | เก็บว่าข้อมูลใดถูกเชื่อมกับข้อมูลใด |
| security_events | เหตุการณ์ด้านความปลอดภัย |
| audit_log_exports | Future: export audit/report |

### activity_logs

id
tenant_id
workspace_id
project_id
actor_user_account_id
actor_profile_id
action
resource_type
resource_id
old_value_json
new_value_json
result
ip_address
user_agent
created_at

### reference_links

id
tenant_id
workspace_id
project_id
source_type
source_id
target_type
target_id
relation_type
status
linked_by
linked_at
unlinked_by
unlinked_at
note

ใช้ตรวจว่าไฟล์/task/timeline/finance ถูกอ้างอิงที่ไหน และใช้กันการลบข้อมูลที่ยังถูกใช้งานอยู่

### security_events

id
tenant_id
workspace_id
user_account_id
event_type
severity
ip_address
user_agent
metadata_json
created_at

ตัวอย่าง event_type: login_failed, login_success, password_changed, password_reset_requested, password_reset_success, session_revoked, force_logout, email_verified, account_suspended, account_disabled

### audit_log_exports (future)

id
tenant_id
workspace_id
requested_by
export_type
filter_json
file_id
status
requested_at
completed_at

## 8.13 Index / Constraint / Soft Delete Rules

### Index Strategy

ทุก table ที่เป็นข้อมูลระดับ Workspace ต้องมี index ที่ขึ้นต้นด้วย tenant_id

ตัวอย่าง:

1. projects: tenant_id + workspace_id, tenant_id + project_status, tenant_id + project_code
1. tasks: tenant_id + project_id, tenant_id + project_id + status_id, tenant_id + assignee_member_id, tenant_id + due_date
1. timeline_items: tenant_id + project_id, tenant_id + planned_date, tenant_id + timeline_type_id
1. project_files: tenant_id + project_id, tenant_id + project_id + folder_id, tenant_id + lifecycle_status
1. activity_logs: tenant_id + workspace_id + created_at, tenant_id + project_id + created_at, tenant_id + resource_type + resource_id
### Search Strategy

ไม่ควรใช้ LIKE '%keyword%' กับ table ใหญ่โดยตรง

MVP ใช้:

1. prefix search
1. filter + pagination
1. index column ที่ค้นบ่อย
อนาคต:

1. PostgreSQL Full Text Search
1. trigram index
1. search vector
1. external search engine
### Pagination Rule

ทุก list page ต้องมี pagination ห้าม query list ใหญ่แบบไม่จำกัดจำนวน

ควรมี:

1. limit
1. offset หรือ cursor
1. sort_by
1. sort_order
### Soft Delete / Hard Delete Rule

Soft Delete ใช้กับข้อมูลที่ต้องเก็บประวัติ เช่น:

1. projects
1. project_members
1. tasks
1. timeline_items
1. project_files
1. finance_entries
1. announcements
1. workspace_memberships
1. master data
Hard Delete:

1. ทำโดยผู้มีสิทธิ์สูง เช่น Platform Admin
1. ต้องมี deletion log
1. ต้องตรวจ reference ก่อนเสมอ
1. ต้องลบ DB metadata และ file object ที่เกี่ยวข้องให้ครบ
### Foreign Key / Reference Rule

ใช้ FK กับความสัมพันธ์ที่แน่นอน เช่น:

1. task -> project
1. project_member -> project
1. finance_entry -> project
relation ข้าม module ที่ยืดหยุ่นให้ใช้ reference_links เช่น:

1. file -> task
1. file -> finance_entry
1. task -> timeline_item
1. announcement -> file
### Snapshot vs Live Reference Rule

| Type | ใช้เมื่อ | ตัวอย่าง |
| --- | --- | --- |
| Snapshot Value | ข้อมูลเก่าต้องไม่เปลี่ยนตาม master/config ใหม่ | ชื่อ category ตอนบันทึก finance, ข้อมูลที่อยู่, ชื่อไฟล์ตอนแนบ |
| Live Reference | แก้ master แล้วอยากให้แสดงชื่อใหม่ | project position, task workflow status, priority name |

## Final Notes / จุดที่ต้องระวังตอนทำจริง

1. Core table เดิมมี users แต่ design จริงให้ใช้ user_profiles + user_accounts เพื่อลดความซ้ำและรองรับ profile ที่ยังไม่มี account
1. platform_admins ควรอ้าง user_account_id
1. workspace_memberships และ project_members ต้องรองรับ profile ที่ยังไม่มี account
1. ไฟล์ห้ามเก็บ full URL จริง ต้องเก็บ object_key แล้วให้ backend สร้าง signed URL หลังตรวจสิทธิ์
1. ทุก table ใหญ่ต้องมี index และ pagination
1. ทุก unique field ต้องพิจารณา soft delete
1. activity_logs, notifications, project_files, tasks เป็น table ที่โตเร็ว ต้องออกแบบ performance ตั้งแต่ต้น
