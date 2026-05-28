# 04 — Data Model (Just-in-Time per Milestone)

> **Status: Canonical.** เอกสารนี้กำหนด schema จริง (ตาราง, คอลัมน์, FK, index, seed) ของ Prasankit v2 — **เขียน just-in-time ต่อ milestone** ([03-build-plan §4](03-build-plan.md), **D19**). ตอนนี้ครอบคลุม **M1 (Identity & Tenancy)**; milestone ถัดไป append section ใหม่ ไม่ model 8 module ล่วงหน้า (กัน over-build, vision §2).
> derive จาก [02-architecture.md](02-architecture.md) (tenant model, conventions §7) + [01-vision.md](01-vision.md) (role model §6, cross-cutting §8). ถ้าขัด → แก้ doc บนก่อน (gate [00-workflow.md](00-workflow.md) §8).
> การเปลี่ยน schema ที่ตัดสินไปแล้วทำตาม gate §8 + append [DECISIONS.md](DECISIONS.md).

## 1. ขอบเขตเอกสารนี้

**อยู่ในนี้ (M1):** identity/auth core, master tables (org role + status masters), workspace + membership + invitation, cross-cutting log tables (audit + security), conventions ที่ apply ลง schema จริง, seed data, isolation invariant mapping.

**ยังไม่อยู่ (milestone ถัดไป):** projects/team/positions (M2), deliverables/submissions (M3), tasks/board (M4), finance (M5), document/announcement (ตัดออก first cut, [03 §2](03-build-plan.md)).

**Decisions ที่ User ตัดสินแล้ว (fold เข้ารอบนี้):** D24 (collapse `workspace_id`), D25 (M1 auth scope = login + account-safety), D26 (split account↔identity), D27 (inline i18n label) — ดู [DECISIONS.md](DECISIONS.md).

**ผ่าน external review (ต่าง model) รอบ 1 — Kael เสนอ, รอ User เคาะก่อน fold เป็น DECISIONS:**
- **D28 (revised): master FK by `code`** (§2.4) — เดิมเสนอ by-`id`+startup-cache; review ชี้ pain (tooling ยาก + magic-UUID ใน partial index) → พลิกเป็น `code` (อ่านออก, ไม่ต้อง cache resolve, predicate สะอาด)
- **D29: log-table carve-out จาก D15** (§2.7) — log code = Go constant อย่างเดียว, ไม่มี FK/`CHECK` (review approve + ชี้ให้ตัด `CHECK` ออกเพื่อ consistent กับเหตุผล carve-out)
- **deferred ไป milestone หน้า** (review flagged): owner soft-delete invariant (§8), assignee/approver FK→`memberships` vs `user_accounts` = ตัดสินตอน M2 (§8)

## 2. Conventions ที่ apply ลง schema (จาก 02 §7, §8)

### 2.1 Primary key & timestamps
- **PK = `id UUID`** ทุกตาราง, **app-generated UUIDv7** (`uuid.NewV7` ใน Go) — **ไม่มี DB default** (v1 ถอด `gen_random_uuid` ออกแล้ว, ref migration `000003`). PK sortable + ไม่ leak ลำดับ serial (02 §2).
- ทุกตาราง: `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`, `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`.
- ตารางที่ลบได้แบบ soft: `deleted_at TIMESTAMPTZ` (NULL = ยังอยู่) + `deleted_by UUID REFERENCES user_accounts(id)`.
- ตาราง mutable ของ business: `created_by` / `updated_by` → `user_accounts(id)` (audit trail; log tables ไม่ต้อง เพราะ append-only).

### 2.2 Master table (D15 — no enum ทุกกรณี)
controlled vocabulary ทุกตัว (status, type, role) = **master table + FK** เสมอ. ห้าม Postgres `ENUM` / `CHECK (col IN (...))`. คอลัมน์มาตรฐานของ master:

| Column | ความหมาย |
| --- | --- |
| `id` | UUIDv7 PK |
| `code` | stable, unique, immutable — โค้ด/FK อ้าง vocabulary ผ่าน `code` |
| `label_th`, `label_en` | display label 2 ภาษา (**inline, D27**). `label_en` = canonical name ในตัว |
| `description` | คำอธิบาย (nullable) |
| `sort_order` | ลำดับแสดงผล (`>= 0`) |
| `is_system` | `true` = ระบบ branch ด้วย; user แก้/ลบ/เปลี่ยน `code` ไม่ได้ |
| `status` | `active` / `deprecated` (เลิกใช้โดยไม่ลบ คง referential history) |

> **`status` ของ master เองเป็น `active`/`deprecated`** — นี่คือ lifecycle ของ "ตัว row vocabulary" ไม่ใช่ business status, จึงคุมด้วย `CHECK` ได้ตามแบบ v1 `workspace_roles` (ปลายทางของ recursion — ไม่ทำ master-of-master). ส่วน business status (account/workspace/membership) = master table แยกตามด้านล่าง.

### 2.3 i18n (D27, 02 §6)
- master table เก็บ `label_th` + `label_en` inline (vision §8 ล็อก 2 ภาษา). ภาษาที่ 3 = migration เพิ่มคอลัมน์ (hypothetical, YAGNI).
- **row ระบบ (`is_system=true`) FE แปลจาก `code` ได้เลย** (02 §6) — label ใน DB เป็น reference/fallback/admin. label ใน DB จะ load-bearing จริงตอน M2 (custom task status ที่ user พิมพ์เอง).
- presenter ส่ง **`code`** (machine) ไม่ส่งข้อความตายตัว (02 §6).

### 2.4 FK อ้าง master: by `code` (D28 — revised หลัง external review)
- **physical FK = `code` (TEXT)** → `master(code)` (UNIQUE). master ยังมี `id UUID PK` ตาม convention §2.1 แต่ **FK อ้าง `code`** (Postgres FK อ้าง column UNIQUE ใดก็ได้).
- คอลัมน์อ้าง master ตั้งชื่อ `*_code` (เช่น `account_status_code`, `org_role_code`) เก็บค่า `code` ตรงๆ.
- **ทำไม code over id:** (1) `code` immutable (`is_system`, §2.2) → denormalize ปลอดภัย, ไม่มีปัญหา natural-key-เปลี่ยน; (2) predicate อ่านออก `WHERE status_code='active'` → partial index / raw query / psql tooling ใช้ได้ตรง ๆ ไม่ต้อง map UUID; (3) **ไม่ต้อง startup cache resolve `code→id`**; (4) domain constant = ค่าที่เก็บจริง (`const OrgRoleOwner="owner"`) ไม่มี indirection.
- **ราคา:** FK column เป็น TEXT (ใหญ่กว่า UUID นิดหน่อย) — ยอมรับได้เพราะ master เล็ก+นิ่ง, `code` สั้น.
- *(เดิม D28 เสนอ FK-by-`id` + cache; external review รอบ 1 ชี้ pain เรื่อง tooling + magic-UUID ใน partial index → revise มา `code` ซึ่งสะอาดกว่าและแก้ทั้งสอง pain. cache master rows ตอน startup ยังทำได้ — แต่เพื่อ **label/sort_order display** ไม่ใช่เพื่อ resolve FK)*

### 2.5 ORM
- M1 ใช้ **GORM** ตาม D11 — ไม่มี hot-path join หนักที่ ORM คุมยาก. ทบทวน pgx/sqlc เฉพาะจุดเมื่อ profiling พบ (02 §2).
- repo เก็บแค่ base `*gorm.DB`, ทุก method เริ่ม `db.WithContext(ctx)` (02 §3 adapter hygiene).

### 2.6 Token state = derive จาก timestamp ไม่ใช่ status enum
verification/reset token + invitation **ไม่มีคอลัมน์ status** — สถานะ derive จาก `used_at` / `revoked_at` / `expires_at` (single source of truth = timestamp, ไม่ต้อง master table ให้ค่าที่ timestamp บอกได้อยู่แล้ว). ต่างจาก v1 ที่พก `status` + timestamp ซ้ำกัน.

### 2.7 Log tables = machine code ไม่ใช่ master FK (**carve-out จาก D15 — ขอ User ยืนยัน**)
`audit_logs` + `security_events` ถือ vocabulary แบบ free/append-only (`action`, `resource_type`, `result`, `event_type`, `severity`) เป็น **machine code TEXT คุมด้วย Go domain constant อย่างเดียว** (ไม่มี `CHECK` ใน DB, ไม่มี FK, ไม่ใช่ master table). เหตุผล:
- log เป็น **append-only ปริมาณสูง** — FK lookup ทุก insert = write overhead เปล่า
- vocabulary **โตตามฟีเจอร์** (action ใหม่ทุก use-case) — master table จะกลายเป็น dumping ground
- ไม่ต้อง i18n ที่ระดับ row (log อ่านโดย ops/FE map จาก code, 02 §6); ไม่มี referential integrity ที่มีความหมาย (resource ถูกลบ log ต้องคงอยู่)
- controlled ที่ระดับ **domain constant ในโค้ด** (เช่น `AuditActionWorkspaceCreate`) แทน FK

> **นี่คือ deviation จาก D15 ("no enum ทุกกรณี").** D15 มุ่ง business vocabulary (status/type/role/category) ที่ user เห็น/ขยาย — log code คนละ class. external review (รอบ 1) approve carve-out + ชี้ว่า **ถ้าจะ carve ต้องตัด `CHECK` ใน DB ออกด้วย** ไม่งั้นยังผูก migration ตอนเพิ่ม code ใหม่ (ขัด logic ของ carve-out เอง) — **แก้แล้ว** (`result`/`severity` ไม่มี `CHECK`, คุมด้วย Go const). รอ User เคาะเป็น D29.

## 3. Tenant Model ใน schema (D24 — resolve open thread 02 §4.1)

**Collapse:** ตัด `tenant_id` แยกทิ้ง. **`workspaces.id` = tenant boundary**. ทุกตาราง workspace-scoped ถือ **`workspace_id UUID REFERENCES workspaces(id)`** เป็น **isolation key ตัวเดียว**.

- เหตุผล: vision §4.1 ล็อก workspace = tenant 1:1 (firm). v1 พก 2 คอลัมน์ = redundant + สับสนว่า filter อันไหน (02 §4.1 flag ไว้เอง). multi-workspace (D3) = 1 user → หลาย workspace ผ่าน membership ไม่ใช่ 1 tenant → หลาย workspace.
- **`TenantContext` (02 §5) ถือ `WorkspaceID`** = ค่าที่ inject ลงทุก repo query ของตาราง workspace-scoped.

### 3.1 Global vs workspace-scoped (จัดประเภทตาม 02 §7)

| ประเภท | ตาราง | isolation |
| --- | --- | --- |
| **Global identity** (cross-workspace — 1 identity อยู่ได้หลาย ws, D3) | `user_accounts`, `auth_identities`, `auth_email_verification_tokens`, `auth_password_reset_tokens`, `security_events` | **ไม่มี** `workspace_id` — identity ไม่ได้เป็นของ workspace ใด |
| **Global/system master** | `org_roles`, `account_statuses`, `workspace_statuses`, `membership_statuses`, `auth_identity_types` | **ไม่มี** `workspace_id`, `is_system=true` |
| **Workspace-scoped** (อยู่ใต้ isolation invariant §4.3) | `workspace_memberships`, `workspace_invitations`, `audit_logs` | **มี** `workspace_id` (ใส่ใน `WHERE` เสมอ) |
| **Boundary** | `workspaces` | query ด้วย `id`/`slug` + membership check (resolver §5) |

> **สำคัญ:** identity เป็น global — isolation key อยู่บน "ข้อมูลของ workspace" ไม่ใช่บน identity. การ "เห็น user ข้าม workspace ไม่ได้" บังคับที่ `workspace_memberships` (สมาชิกของ ws นี้คือใคร) ไม่ใช่ที่ `user_accounts`.

## 4. M1 Schema

### 4.1 Identity / Auth core (global)

**`user_accounts`** — identity กลาง (ไม่ผูก auth method, D26)

| Column | Type | Note |
| --- | --- | --- |
| `id` | UUID PK | |
| `primary_email` | CITEXT NOT NULL | email หลักของ identity (case-insensitive) |
| `display_name` | TEXT NOT NULL | ชื่อแสดงผล (เก็บตอน signup) |
| `account_status_code` | TEXT NOT NULL → `account_statuses(code)` | FK master by code (§2.4) |
| `failed_login_count` | INT NOT NULL DEFAULT 0 | lockout (D25), `>= 0` |
| `locked_until` | TIMESTAMPTZ | NULL = ไม่ล็อก |
| `last_login_at` | TIMESTAMPTZ | |
| `created_at/updated_at` | TIMESTAMPTZ | |
| `deleted_at` | TIMESTAMPTZ | soft delete |
| `deleted_by` | UUID → `user_accounts(id)` | |

- `CREATE UNIQUE INDEX uq_user_accounts_primary_email_active ON user_accounts (primary_email) WHERE deleted_at IS NULL` — 1 email = 1 active identity.
- `ix_user_accounts_account_status` WHERE `deleted_at IS NULL`.
- **Owner-orphan invariant (external review):** soft-delete account ที่ยังเป็น `owner_user_account_id` ของ workspace `active` อยู่ = ห้าม (จะเกิด orphan workspace). บังคับที่ **service layer** ตอน account-deletion use case (ต้อง transfer ownership ก่อน) — **ไม่ใช้ DB trigger** (business rule อยู่ service, 02 §3). account-deletion ไม่อยู่ M1 scope → ดู §8.

**`auth_identities`** — วิธี authenticate (M1 = email_password; OAuth-ready, D26)

| Column | Type | Note |
| --- | --- | --- |
| `id` | UUID PK | |
| `user_account_id` | UUID NOT NULL → `user_accounts(id)` | |
| `identity_type_code` | TEXT NOT NULL → `auth_identity_types(code)` | FK master by code (§2.4; แทน v1 CHECK) |
| `email` | CITEXT | สำหรับ email_password |
| `email_verified_at` | TIMESTAMPTZ | NULL = ยังไม่ยืนยัน (D25) |
| `password_hash` | TEXT | bcrypt |
| `password_changed_at` | TIMESTAMPTZ | |
| `last_used_at` | TIMESTAMPTZ | |
| `created_at/updated_at/deleted_at` | TIMESTAMPTZ | |

- `CREATE UNIQUE INDEX uq_auth_identities_email_password_active ON auth_identities (email) WHERE email IS NOT NULL AND deleted_at IS NULL` — กัน identity email ซ้ำ.
- `ix_auth_identities_user_account` WHERE `deleted_at IS NULL`.
- M1 มี identity_type เดียว (`email_password`). OAuth (เพิ่ม `provider_user_id` ฯลฯ) = future column เมื่อ un-non-goal vision §9.

**`auth_email_verification_tokens`** (D25) — state derive จาก timestamp (§2.6)

| Column | Type | Note |
| --- | --- | --- |
| `id` | UUID PK | |
| `auth_identity_id` | UUID NOT NULL → `auth_identities(id)` | |
| `token_hash` | TEXT NOT NULL | เก็บ sha256 hash (token ดิบส่งทาง email) |
| `expires_at` | TIMESTAMPTZ NOT NULL | |
| `used_at` | TIMESTAMPTZ | |
| `revoked_at` | TIMESTAMPTZ | |
| `created_at` | TIMESTAMPTZ | |

- `CREATE UNIQUE INDEX uq_email_verif_token_hash ON auth_email_verification_tokens (token_hash)`.
- `ix_email_verif_identity ON auth_email_verification_tokens (auth_identity_id)`.

**`auth_password_reset_tokens`** (D25) — โครงเดียวกับ verification token

| Column | Type | Note |
| --- | --- | --- |
| `id` | UUID PK | |
| `auth_identity_id` | UUID NOT NULL → `auth_identities(id)` | |
| `token_hash` | TEXT NOT NULL | sha256 hash |
| `expires_at` | TIMESTAMPTZ NOT NULL | |
| `used_at` / `revoked_at` / `created_at` | TIMESTAMPTZ | |

- `uq_pwreset_token_hash` UNIQUE (`token_hash`); `ix_pwreset_identity` (`auth_identity_id`).

> **Session ไม่อยู่ใน Postgres (D13).** opaque token สุ่ม 32B → เก็บ **sha256 hash ใน Redis** เป็น `SessionRecord` พร้อม TTL (ดู §6). v1 มี `auth_sessions` ใน Postgres — v2 **ไม่ทำ** (revoke = ลบ key Redis). device/session-list = future.

### 4.2 Master tables (global/system)

**`org_roles`** (vision §6.1; v1 ชื่อ `workspace_roles`) — role ระดับองค์กร/workspace

มาตรฐาน master §2.2. seed:

| code | label_th | label_en | sort | is_system |
| --- | --- | --- | --- | --- |
| `owner` | เจ้าของ | Owner | 10 | true |
| `admin` | ผู้ดูแลระบบ | Admin | 20 | true |
| `executive` | ผู้บริหาร | Executive | 30 | true |
| `user` | ผู้ใช้งาน | User | 40 | true |

- `CHECK (code ~ '^[a-z][a-z0-9_]{1,62}$')`, `uq_org_roles_code` UNIQUE (`code`).
- Domain constants: `OrgRoleOwner="owner"`, `OrgRoleAdmin="admin"`, `OrgRoleExecutive="executive"`, `OrgRoleUser="user"`.

**`account_statuses`** — business status ของ user_accounts

| code | label_th | label_en | sort |
| --- | --- | --- | --- |
| `pending_verification` | รอยืนยัน | Pending Verification | 10 |
| `active` | ใช้งาน | Active | 20 |
| `suspended` | ระงับชั่วคราว | Suspended | 30 |
| `disabled` | ปิดใช้งาน | Disabled | 40 |
| `deleted` | ลบแล้ว | Deleted | 50 |

**`workspace_statuses`**

| code | label_th | label_en | sort |
| --- | --- | --- | --- |
| `active` | ใช้งาน | Active | 10 |
| `suspended` | ระงับ | Suspended | 20 |
| `pending_deletion` | รอลบ | Pending Deletion | 30 |
| `deleted` | ลบแล้ว | Deleted | 40 |

**`membership_statuses`** (02 §7 ระบุเป็น code-branched system master)

| code | label_th | label_en | sort |
| --- | --- | --- | --- |
| `active` | ใช้งาน | Active | 10 |
| `suspended` | ระงับ | Suspended | 20 |
| `removed` | ถูกนำออก | Removed | 30 |

**`auth_identity_types`**

| code | label_th | label_en | sort |
| --- | --- | --- | --- |
| `email_password` | อีเมล/รหัสผ่าน | Email & Password | 10 |

> (future OAuth seed: `google`, `microsoft`, ... เพิ่ม row เมื่อทำจริง — ไม่ migration schema, แค่ INSERT)

ทุก master: `code` UNIQUE + `CHECK (code ~ '^[a-z][a-z0-9_]{1,62}$')` + `CHECK (status IN ('active','deprecated'))` + `CHECK (sort_order >= 0)`.

### 4.3 Workspace core

**`workspaces`** (boundary; collapse — ไม่มี tenant_id, D24)

| Column | Type | Note |
| --- | --- | --- |
| `id` | UUID PK | **= tenant boundary** |
| `workspace_name` | TEXT NOT NULL | |
| `slug` | CITEXT NOT NULL | URL-safe, unique ตอน active |
| `workspace_status_code` | TEXT NOT NULL → `workspace_statuses(code)` | FK master by code (§2.4) |
| `contact_email` | CITEXT NOT NULL | |
| `owner_user_account_id` | UUID NOT NULL → `user_accounts(id)` | |
| `created_at/created_by` | | `created_by` → `user_accounts(id)` |
| `updated_at/updated_by` | | |
| `pending_deletion_at` | TIMESTAMPTZ | |
| `deleted_at/deleted_by` | | soft delete |

- `CHECK (slug ~ '^[a-z0-9]([a-z0-9-]{1,61}[a-z0-9])?$')`.
- `CREATE UNIQUE INDEX uq_workspaces_slug_active ON workspaces (slug) WHERE deleted_at IS NULL`.
- `ix_workspaces_owner_status ON workspaces (owner_user_account_id, workspace_status_code) WHERE deleted_at IS NULL`.
- ตัดจาก v1: `mode` (demo/production), `email_verified_required`, `hard_deleted_at` — ไม่ใช่ M1 scope (future).

**`workspace_memberships`** (workspace-scoped; collapse → `workspace_id` เท่านั้น, D24)

| Column | Type | Note |
| --- | --- | --- |
| `id` | UUID PK | |
| `workspace_id` | UUID NOT NULL → `workspaces(id)` | **isolation key** |
| `user_account_id` | UUID NOT NULL → `user_accounts(id)` | |
| `org_role_code` | TEXT NOT NULL → `org_roles(code)` | FK by code (§2.4) |
| `membership_status_code` | TEXT NOT NULL → `membership_statuses(code)` | FK by code; **source of truth เดียวของ status** |
| `invited_by_user_account_id` | UUID → `user_accounts(id)` | NULL ถ้า owner สร้างเอง |
| `joined_at` | TIMESTAMPTZ | เมื่อ membership active (audit timestamp, ไม่ใช่ source of truth ของ status) |
| `created_at/created_by/updated_at/updated_by` | | |

- `CREATE UNIQUE INDEX uq_membership_active_user ON workspace_memberships (workspace_id, user_account_id) WHERE membership_status_code = 'active'` — 1 user = 1 active membership ต่อ workspace. predicate เป็น text literal คงที่ (FK-by-code §2.4) — **ไม่ผูก magic UUID ของ seed, ไม่ fragile** (external review fix).
- `ix_membership_workspace_status ON workspace_memberships (workspace_id, membership_status_code)`.
- `ix_membership_user ON workspace_memberships (user_account_id)`.
- **ตัด `removed_at`/`suspended_at` ออก** (เดิม redundant กับ status_code = 2 source of truth เสี่ยง drift, แบบที่ §2.6 วิจารณ์ v1). transition history (ใครเปลี่ยน status เมื่อไหร่) → `audit_logs`.
- ตัดจาก v1: `tenant_id` (collapse), `profile_id` (per-ws profile = future).

**`workspace_invitations`** (workspace-scoped; D20 accept-invite, D4 external client) — state derive จาก timestamp (§2.6)

| Column | Type | Note |
| --- | --- | --- |
| `id` | UUID PK | |
| `workspace_id` | UUID NOT NULL → `workspaces(id)` | **isolation key** |
| `email` | CITEXT NOT NULL | เชิญด้วย email (ผู้รับอาจยังไม่มี account) |
| `org_role_code` | TEXT NOT NULL → `org_roles(code)` | role ที่จะได้เมื่อรับเชิญ (FK by code §2.4) |
| `token_hash` | TEXT NOT NULL | sha256 (token ดิบส่งทาง email/link) |
| `invited_by_user_account_id` | UUID NOT NULL → `user_accounts(id)` | |
| `expires_at` | TIMESTAMPTZ NOT NULL | |
| `accepted_at` / `revoked_at` | TIMESTAMPTZ | |
| `accepted_user_account_id` | UUID → `user_accounts(id)` | ใครรับ (ผูกตอน accept) |
| `created_at` | TIMESTAMPTZ | |

- `uq_invitation_token_hash` UNIQUE (`token_hash`).
- `CREATE UNIQUE INDEX uq_invitation_pending ON workspace_invitations (workspace_id, email) WHERE accepted_at IS NULL AND revoked_at IS NULL` — กันเชิญ email ซ้ำตอนยัง pending.
- `ix_invitation_workspace ON workspace_invitations (workspace_id)`.

### 4.4 Cross-cutting (ทอตั้งแต่ M1 — 03 §6)

**`audit_logs`** (workspace business activity, vision §8) — append-only, structured

| Column | Type | Note |
| --- | --- | --- |
| `id` | UUID PK | |
| `workspace_id` | UUID → `workspaces(id)` | isolation key (**nullable**: event ก่อนมี ws context = signup ไม่ลงที่นี่ → security_events) |
| `actor_user_account_id` | UUID → `user_accounts(id)` | ใครทำ (NULL ได้ถ้า system) |
| `action` | TEXT NOT NULL | machine code เช่น `workspace.create`, `membership.role_change` |
| `resource_type` | TEXT NOT NULL | เช่น `workspace`, `membership` |
| `resource_id` | UUID | id ของ resource |
| `old_value` / `new_value` | JSONB | diff ก่อน/หลัง |
| `result` | TEXT NOT NULL | `success` / `failure` |
| `ip_address` | INET | จาก transport |
| `user_agent` | TEXT | |
| `request_id` | TEXT | จาก request-id middleware (02 §8) |
| `created_at` | TIMESTAMPTZ | |

- `ix_audit_workspace_created ON audit_logs (workspace_id, created_at DESC)`.
- `ix_audit_actor_created ON audit_logs (actor_user_account_id, created_at DESC)`.
- `action`/`resource_type`/`result` = machine code (i18n ฝั่ง FE, 02 §6). **ไม่ทำเป็น master table** เพราะเป็น log แบบ free-vocabulary ที่โตตามฟีเจอร์ — controlled ที่ระดับ domain constant ในโค้ด ไม่ใช่ FK (ต่างจาก business vocabulary §2.2). M1 ยังไม่มี UI (03 §2).
- **Write policy = must-succeed in-tx (D31):** business mutation เขียน `audit_logs` ใน transaction เดียวกับ insert หลัก (all-or-nothing) ผ่าน `auditRepo.LogTx(tx, entry)`. ต่างจาก `security_events` (auth) ที่ best-effort.

**`security_events`** (auth/account security trail; D25 account-safety) — append-only

| Column | Type | Note |
| --- | --- | --- |
| `id` | UUID PK | |
| `user_account_id` | UUID → `user_accounts(id)` | nullable (login fail ของ email ที่ไม่มี account) |
| `event_type` | TEXT NOT NULL | `login.success`, `login.failure`, `account.locked`, `password.reset`, `email.verified`, ... |
| `severity` | TEXT NOT NULL DEFAULT 'info' | Go-const: `info`/`warning`/`critical` (ไม่มี `CHECK` — §2.7) |
| `ip_address` | INET | |
| `user_agent` | TEXT | |
| `metadata` | JSONB NOT NULL DEFAULT '{}' | เช่น failure_reason |
| `created_at` | TIMESTAMPTZ | |

- `ix_security_user_created ON security_events (user_account_id, created_at DESC)`.
- `ix_security_type_created ON security_events (event_type, created_at DESC)`.
- แยกจาก `audit_logs` เพราะ: (1) auth event ส่วนใหญ่ **ไม่มี workspace context** (signup/login เกิดก่อนเลือก ws); (2) คนละ audience (security ops vs business activity). v1 ก็แยกสองตารางนี้.

## 5. Isolation invariant mapping (02 §4.3)

| ข้อ (02 §4.3) | M1 บังคับที่ไหน |
| --- | --- |
| 1. repo method ตาราง workspace-scoped รับ `workspace_id` + ใส่ `WHERE` เสมอ | `workspace_memberships`, `workspace_invitations`, `audit_logs` (เมื่อ workspace-scoped) |
| 2. service ไม่รับ `workspace_id` จาก client input — รับจาก `TenantContext` | resolver §5 (02) สร้าง `TenantContext.WorkspaceID` จาก `X-Workspace-Slug` + membership check |
| 3. test "workspace อื่นมองไม่เห็น" อย่างน้อย 1 ต่อ repo | **`workspace_memberships` repo** (M1 key) + `workspace_invitations` |

- **identity tables เป็น global** (§3.1) → ไม่อยู่ใต้ invariant นี้ (admin/global path ที่ระบุชัด + test คุม, 02 §4.3 ข้อ 1 ยกเว้น).
- resolver flow (02 §5.2): session → account → `X-Workspace-Slug` → lookup `workspace_memberships` ว่า account นี้ active ใน ws นั้น → ถ้าใช่คืน `TenantContext`. authz (org role) resolve per-request ไม่ cache ใน session (D16).

## 6. Session shape (Redis, ไม่ใช่ table — D13)

`SessionRecord` ใน Redis (key = sha256(token), TTL):

```
account_id     UUID        // identity เท่านั้น (ไม่ denormalize role/membership — D16)
created_at     timestamp
last_seen_at   timestamp
ip_address     string      // optional, สำหรับ security_events
user_agent     string
```

- login → สร้าง opaque token 32B (`pkg/securetoken`) → เก็บ hash ใน Redis → token ดิบลง cookie (httpOnly+secure+SameSite, 02 §5.1).
- logout/revoke = ลบ key. authz (membership/org role) resolve ใหม่ทุก request จาก Postgres ตาม `TenantContext` (D16) — **ไม่อยู่ใน SessionRecord**.

## 7. Migration plan (M1)

goose, ต่อจาก `000001_enable_extensions` (มี CITEXT แล้ว). ลำดับ (master ก่อน เพราะ FK):

1. `000002_create_auth_identity_masters` — `auth_identity_types`, `account_statuses` (+ seed)
2. `000003_create_auth_core` — `user_accounts`, `auth_identities`, `auth_email_verification_tokens`, `auth_password_reset_tokens`
3. `000004_create_org_role_master` — `org_roles` (+ seed)
4. `000005_create_workspace_masters` — `workspace_statuses`, `membership_statuses` (+ seed)
5. `000006_create_workspace_core` — `workspaces`, `workspace_memberships`, `workspace_invitations`
6. `000007_create_crosscutting_logs` — `audit_logs`, `security_events`

> seed master row: hardcode `id` UUIDv7 + `code` ใน migration (immutable system row). FK ใช้ `code` (§2.4) → **ไม่ต้อง resolve code→id**; partial index `WHERE status_code='active'` ใช้ text literal ตรง ๆ. cache master rows ตอน startup เพื่อ label display ได้ (optional).

## 8. ค้างไว้ให้ milestone ถัดไป

- per-workspace **profile** (v1 `memberships.profile_id`) — ถ้า M2 ต้องการชื่อ/ตำแหน่งราย workspace
- workspace `mode` (demo/production), `email_verified_required` policy — เมื่อมี onboarding policy จริง
- OAuth columns บน `auth_identities` (`provider_user_id` ฯลฯ) — เมื่อ vision §9 un-non-goal SSO
- audit_logs **UI** (03 §2 ตัดออก first cut) + `project_id` column (เพิ่มตอน M2 มี projects)
- RLS hardening (02 §4.4) — candidate security pass
- **Owner soft-delete invariant** (external review) — ห้าม soft-delete account ที่ owns active workspace; enforce service-level ตอนสร้าง account-deletion use case + **ownership transfer** flow (ทั้งคู่ยังไม่อยู่ M1)
- **Assignee/approver FK target = decide ตอน M2** (external review เสนอ "iron rule") — workspace-scoped entity (task assignee, budget approver) จะ FK → `workspace_memberships(id)` (บังคับว่า assignee เป็นสมาชิก ws จริง + isolation audit ง่าย; แลกกับ join เพิ่ม + ต้องคง membership row + composite FK `(workspace_id, membership_id)`) หรือ → `user_accounts(id)` (ตรงกว่า แต่ assign คนนอก ws ได้ถ้าไม่ guard). **ไม่ล็อกตอน M1** (D19 just-in-time); ประเมินตอนมี entity จริง

## 9. ความสัมพันธ์กับเอกสารอื่น

- [02-architecture.md](02-architecture.md) — tenant model §4 (D24 resolve §4.1), conventions §7, resolution/auth §5
- [01-vision.md](01-vision.md) — role §6, cross-cutting §8
- [03-build-plan.md](03-build-plan.md) — M1 scope §5, cross-cutting ทอ M1 §6
- `archive/v1-legacy` — reference schema (`git show archive/v1-legacy:app-api/database/migrations/...`) — ไม่ใช่ข้อผูกมัด (v2 ต่างที่ master-everywhere, no-tenant_id, session-in-Redis)
