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

> **Login gating + account-safety behavior (D33/D34, dispatch 5a):**
> - **Login requires `account_status='active'`.** `pending_verification` ถูกปฏิเสธด้วย distinct `email_not_verified` (403) **หลัง password verify** เท่านั้น (ไม่เป็น enum oracle); suspended/disabled/deleted → generic invalid-credentials. `IsLoginAllowed()` = `active` only.
> - **Signup** auto-ออก verification token + ส่ง email (best-effort, ไม่ fatal — signup ยัง 201 ถ้า email ล้ม). **Confirm** (`POST /auth/verify-email`): token state derive จาก timestamp (§2.6); guarded consume (`UPDATE ... WHERE used_at IS NULL AND revoked_at IS NULL`, `RowsAffected==0`→410) กัน double-confirm race; flip `pending_verification→active` แบบ conditional (ไม่ downgrade suspended). **Resend** (`POST /auth/verify-email/resend`) คืน 204 เสมอ (anti-enum best-effort, ไม่ทำ timing padding).
> - **Deferred (D34):** rate-limit/cooldown ของ auth endpoints = dedicated combined pass; verification token TTL = 24h, 1 active token ต่อ identity (ออกใหม่ = revoke เก่า).
>
> **Password reset (D35, dispatch 5b):**
> - **Request** (`POST /auth/password-reset/request`) คืน **204 เสมอ** (anti-enum; ออก token ได้แม้ account ยัง `pending_verification` — recovery). **Confirm** (`POST /auth/password-reset/confirm {token,new_password}`): validate password ≥8 → guarded consume token (`RowsAffected==0`→410) → set `password_hash`+`password_changed_at` → **clear lockout** (`failed_login_count=0`,`locked_until=NULL`). **ไม่ auto-login**; **ไม่แตะ `account_status_code`** (reset ≠ verify email, แยก concern). token TTL=1h, 1 active/identity.
> - `password_changed_at` อยู่ใน `auth_identities` (เขียนตอน reset) — มีใน GORM model+domain เพื่อ model สะท้อน table.
>
> **Lazy session revocation (D36, dispatch 5c — ปิด M1 BE):** `requireSession` → `ResolveSessionAccount`: ถ้า `session.created_at < identity.password_changed_at` → ลบ session ออกจาก Redis + 401 (password reset เตะอุปกรณ์เก่าทั้งหมดออกโดยไม่ต้องมี account→session index). `password_changed_at == nil` → ข้าม (no regression). consolidate `GetSession`+`GetAccountByID` → `ResolveSessionAccount`. **Open thread (future, จาก external review):** load account+identity เป็น single JOIN ถ้า `requireSession` กลายเป็น hot path; rethink revocation semantics เมื่อมี multi-identity/MFA (M2+).

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
- **Placeholder member + claim-by-email (M2 — User request 2026-05-28):** รองรับ "คน/โปรไฟล์" ใน workspace ที่ **ยังไม่มี account** (สร้างไว้ลอย ๆ, assign งานใน project ได้ แต่ **ล็อกอินไม่ได้** — ต่างจาก invite ที่ให้สิทธิ์เข้า) แล้ว **"claim/link"** ผูกเข้ากับ `user_account` จริงเมื่อเจ้าตัวสมัครภายหลัง. ต้อง **verify email-match** ตอน claim (กัน claim ผิดคน, เหมือน accept-invite §4.3). pattern: GitHub commit-claim-by-email / Jira assign-before-join. ต้องแยก **profile (workspace-scoped)** ออกจาก **user_account (global)** — v1 `memberships.profile_id` เป็นเค้าเดิม. **ติดที่ M1 ตอนนี้:** `workspace_memberships.user_account_id` = NOT NULL (สมาชิกต้องมี account); M2 ต้องเพิ่มชั้น profile ที่ account-optional. ออกแบบรวมกับ assignee-FK ด้านล่าง
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

---

## M2 — Functional Project

> **Status: Canonical (proposed).** เอกสารส่วนนี้ extend §M1 ตาม [03-build-plan §5 row M2](03-build-plan.md) — เพิ่ม `projects`, project lifecycle/type/role masters, `project_members`, positions (project + company) และ ALTER `audit_logs` เพิ่ม `project_id`. **อ้างอิงและสานต่อ conventions §2 และ tenant model §3 ของ M1 — ไม่เปิดใหม่.**
> Decisions เสนอที่นี่: **D38–D44** (รอ User เคาะ; ดู §M2.7).

## M2.1 ขอบเขตเอกสารนี้

**อยู่ในนี้ (M2):**
- `projects` (workspace-scoped business; CRUD + lifecycle status + type + owner + date range + slug)
- `project_statuses` (global system master, 8 rows — vision §4.2)
- `project_types` (global system master, 2 rows — vision §4.3) + `projects.requesting_unit` (free TEXT ตาม vision §4.3 wording; ดู notes §M2.3.3)
- `project_roles` (global system master, 5 rows — vision §6.2)
- `project_members` (workspace-scoped business; **composite FK `(workspace_id, workspace_membership_id)`** — "iron rule" §M2.2.1)
- `project_positions` + `company_positions` (workspace-scoped **customizable** masters — 2 ตารางแยก, ดู §M2.3.2 และ **D39**)
- `project_member_positions` (M:N link — 1 project_member สวมหลาย project position ได้, **D39**); **FK-by-code** ขนาน D28 (ดู §M2.2.4)
- ALTER `workspace_memberships` (เพิ่ม `company_position_code` + backing `UNIQUE (workspace_id, id)`)
- ALTER `audit_logs` (เพิ่ม nullable `project_id` + CHECK `project_id IS NULL OR workspace_id IS NOT NULL`) — ปิด open thread M1 §8

**ยังไม่อยู่ (defer to M3+):**
- **Placeholder member + claim-by-email** (M1 §8) — confirm defer ออกจาก M2; ต้อง profile layer ที่ account-optional, ออกแบบรวมกับ placeholder ใน M3+. M2 สมาชิก project ต้องมี `workspace_membership` active (ซึ่งมี `user_account` จริง). **D4 reaffirm:** external collaborator path ใน M2 = ws-invite (M1, D32) + add to project_members (M2) — external person ต้อง verify email ตาม D33 ก่อนถูก add. ไม่มี project-scoped invitation ใน M2
- **Per-workspace profile** (`memberships.profile_id`, M1 §8) — coupling กับ placeholder-member; defer พร้อมกัน. 3-axis vision §6 (Role / Project Position / Company Position) แสดงครบได้โดยไม่ต้องมี profile row
- **Project lifecycle state machine / transition guard** — M2 = free transitions (D44, build-plan locked). FK บังคับแค่ vocabulary; transition graph (เช่น Closed→Draft ห้าม) รอ M3+ เมื่อ deliverable/finance ผูก lifecycle จึงค่อยตัดสิน
- **Project owner soft-delete + ownership-transfer flow** — เทียบ workspace owner-orphan M1 §4.1; ห้าม remove ownership ที่ยัง own project active. enforce ที่ service-layer ตอน account-deletion / member-removal use case (ทั้งคู่ M3+)
- **Project-scoped invitation** (เชิญตรงเข้า project โดยไม่ผ่าน ws invite) — ปัจจุบันใช้ 2-step (ws invite → add to project) ครอบงานได้
- **Project member bulk add/remove + role history table** — service-level M3+; ในระหว่างนี้ role history reconstruct ผ่าน `audit_logs` (`project_member.role_change`) — single source of truth
- **Position seed default ตอน create workspace** — M2 workspace ใหม่เริ่มว่าง; ถ้า demo flow ต้องการ default = M3+ บน `workspace.create` service ไม่ใช่ migration seed
- Deliverable (M3) / Task (M4) / Finance (M5) — milestone ถัดไป (M5 finance read-gate hook ดู §M2.5)

**Decisions ที่ User ตัดสินไปแล้วก่อน M2 dispatch (fold เข้ารอบนี้):**
- Placeholder-member + claim-by-email **defer** — สมาชิกต้องมี `workspace_membership` active (มี `user_account_id`)
- Assignee/approver FK target = **`workspace_memberships(id)` composite `(workspace_id, membership_id)`** (iron rule จาก M1 external review, ปิด M1 §8 open thread)
- `project_statuses`/`project_types`/`project_roles` = **global system master** (`is_system=true`, ไม่มี `workspace_id`, vocab ล็อกโดย product)
- `positions` = **workspace-scoped customizable** master (มี `workspace_id`, อยู่ใต้ isolation invariant §4.3)
- Project lifecycle = **free transitions** ใน M2 (D44)

## M2.2 Conventions extension (M2)

นอกจาก conventions §2 (M1) ที่ apply ต่อทั้งหมด, M2 เพิ่ม 4 pattern:

### 2.8 Composite FK `(workspace_id, membership_id)` — "iron rule" (เสนอ **D38**; amend architecture §4.3)

**กฎใหม่:** ทุก workspace-scoped business row ที่อ้างถึง "คน/entity ใน ws" ใช้ **composite FK** ไปที่ parent table:

```sql
FOREIGN KEY (workspace_id, workspace_membership_id)
    REFERENCES workspace_memberships (workspace_id, id)
```

**ต้องมี backing UNIQUE บน parent (M2 ALTER):**
```sql
ALTER TABLE workspace_memberships
    ADD CONSTRAINT uq_workspace_memberships_workspace_id_id
    UNIQUE (workspace_id, id);
```
`id` ยังเป็น PK เดิม; constraint นี้เป็น no-op ด้าน data (PK guarantee uniqueness ของ `id` อยู่แล้ว) แต่ Postgres ต้องการ explicit composite UNIQUE บน referenced columns เป็น FK target.

**ทำไมไม่ใช้ single-column FK → `workspace_memberships(id)`:**
- single-column FK ยืนยันแค่ว่า "membership_id มีอยู่จริง" — **ไม่ยืนยันว่า membership อยู่ใน workspace เดียวกับ row ที่ถืออยู่**
- ถ้า `project_members.workspace_id = ws_A` แต่ `membership_id` ชี้ไป membership ของ `ws_B` → Postgres รับ; กลายเป็น app-level invariant ที่พลาดได้
- composite FK บังคับ `project_members.workspace_id == workspace_memberships.workspace_id` ที่ระดับ **DB** → cross-workspace assignment = constraint violation, ไม่ใช่ silent leak. นี่คือ **DB-level backstop** ของ isolation invariant §4.3 — service+repo ลืม `WHERE workspace_id` กลายเป็น insert/update fail (FK violation) ไม่ใช่ data leak

> **Amends architecture §4.3 isolation invariant:** เดิม 3 ข้อ (repo รับ workspace_id, service ไม่เชื่อ client, test cross-ws). D38 เพิ่ม **ข้อ 4: DB-level composite FK backstop** บน workspace-scoped business row ที่อ้าง entity ใน ws. commit เดียวกับ M2 merge ต้อง update [02-architecture §4.3](02-architecture.md) ให้แสดง 4 ข้อ.

**Scope exclusion (ไม่ใช้ composite FK):**
- **Audit/identity columns** (`created_by`, `updated_by`, `deleted_by`, `invited_by`, `actor_user_account_id`, `owner_user_account_id` ที่ workspace-boundary level) → ใช้ single-column FK → `user_accounts(id)`. คน 1 คนมี membership หลาย ws (D3); actor อาจทำ action ก่อนได้ membership (เช่น owner สร้าง workspace) หรือหลัง membership ถูก remove (audit history). composite FK ที่นี่ผิด class
- **Self-referential workspace_id FK** เช่น `projects.workspace_id → workspaces(id)` ใช้ single-column FK (workspace boundary เอง — left==right จะ self-redundant)
- iron rule apply เฉพาะ **assignment columns** (assignee, approver, member-of-something, owner-of-something-ใน-ws)

**Worked example — `project_members` insert:**

ws A (`019d…aaa`) มี project P (`019d…ppp`) และ user U เป็นสมาชิก ws A ผ่าน membership M (`019d…mmm`, workspace_id=`019d…aaa`). สร้าง row assign U เข้า P เป็น Project Manager:

```sql
INSERT INTO project_members (
    id, workspace_id, project_id,
    workspace_membership_id, project_role_code, ...
) VALUES (
    '019d…xxx', '019d…aaa', '019d…ppp',
    '019d…mmm', 'project_manager', ...
);
```

DB ตรวจ:
1. `(workspace_id=…aaa, project_id=…ppp)` → `projects(workspace_id, id)` — ต้อง match
2. `(workspace_id=…aaa, workspace_membership_id=…mmm)` → `workspace_memberships(workspace_id, id)` — **บังคับ membership อยู่ ws เดียวกับ project_member**
3. `project_role_code='project_manager'` → `project_roles(code)` — vocabulary check

ถ้า attacker ส่ง payload assign membership ของ ws B (`…bbb`) เข้า project ของ ws A:
- service resolve `workspace_id` จาก `TenantContext` (= `…aaa`, ไม่เชื่อ client)
- insert จะมี `workspace_id=…aaa` แต่ `workspace_membership_id` ชี้ row ที่มี `workspace_id=…bbb`
- composite FK #2 ล้ม → ROLLBACK ก่อน data leak

> **Multi-ws user case (test requirement §M2.5):** ถ้า user U เป็นสมาชิก **ทั้ง** ws A และ ws B (membership_A, membership_B), attacker ที่รู้ `membership_B.id` แต่ context เป็น ws A → insert `workspace_id=…aaa, workspace_membership_id=membership_B.id` ก็ล้มที่ FK #2 ด้วยเหตุผลเดียวกัน. iron rule กัน "พลาด membership ผิด ws" ไม่ใช่กัน "user ผิด"

**ใช้ใน M2:** `projects.owner_project_member_id` (composite ไป `project_members`, D40), `project_members.workspace_membership_id` + `.project_id`, `project_member_positions.project_member_id` + `.project_position_code` (FK-by-code, §M2.2.4), `workspace_memberships.company_position_code`. **M3+** ทุก task_assignee / budget_approver / deliverable_reviewer ใช้ pattern เดียวกัน

### 2.9 Workspace-scoped customizable master ("user เพิ่ม row เองได้")

M1 มีแต่ global/system master. M2 เปิด class ใหม่: **workspace-scoped customizable master** (architecture §7 ระบุไว้). กฎที่ต่างจาก global master §2.2:

- มี `workspace_id NOT NULL` → อยู่ใต้ isolation invariant §4.3 (repo รับ `workspace_id` + ใส่ใน `WHERE` เสมอ)
- `is_system` = `false` สำหรับ row ที่ user สร้าง; ถ้า seed default ตอนสร้าง ws (M3+) ค่อย `is_system=true` กัน user แก้/ลบ/เปลี่ยน `code` (enforce **ที่ service layer** เหมือน M1 §2.2 — ไม่ใช่ DB CHECK เพราะกฎ "ต่าง field แก้ได้/ไม่ได้" ตามชนิด row ไม่ static)
- `code` UNIQUE **per workspace** ไม่ใช่ global: `UNIQUE (workspace_id, code)`
- `code ~ '^[a-z][a-z0-9_]{1,62}$'` ยังบังคับ (format เดียวกับ system master เพื่อ predictable URL/API)
- **`code` immutable ทุก row** (system **และ** non-system) — แก้ label/description/sort_order/status ได้ แต่ **ไม่เปลี่ยน code** เพราะ FK-by-code (D28, §M2.2.4) อ้างถึง. FE: ฟอร์ม edit ตัด field code ออกหลัง first save
- **ห้าม reuse code ที่ deprecated** — `INSERT` row ใหม่ที่ code ตรงกับ row ใด ๆ ใน ws นั้น (ไม่ว่า `active` หรือ `deprecated`) = service reject. enforce ที่ service layer + test; UNIQUE ที่ `(workspace_id, code)` non-partial บังคับ DB-level อยู่แล้ว ("eternal code per workspace" — คง referential history)
- **เลิกใช้ soft-delete pattern (`deleted_at`) — ใช้ `status='active'|'deprecated'` แทน** (matched M1 system master §2.2). เหตุผล: (1) consistent กับ M1; (2) Postgres FK target ต้องการ UNIQUE non-partial, ถ้าใช้ partial `WHERE deleted_at IS NULL` จะตั้ง composite FK ไม่ได้ → ใช้ `(workspace_id, code)` non-partial; (3) `deprecated` row คง referential history; FE filter ออกจาก picker
- เพิ่ม audit column `created_by`/`updated_by` (user เพิ่ม row เอง → ต้อง trace ใครทำ) — ต่างจาก system master ที่ใส่ทาง migration
- `description` = single-language เหมือน M1 §2.2 master (operator-facing TH ใน M2 product; ภาษาที่ 2 = migration เพิ่ม column ภายหลัง — YAGNI path เดียวกับ D27)

### 2.10 Project slug = public identifier per workspace (apply D32 pattern + cross-ws probe fix)

`projects.slug CITEXT NULL` — optional public identifier ภายใน workspace. URL: `/{workspace_slug}/projects/{project_slug}`.
- **nullable** — Draft/imported project ใช้ UUID ก็ทำงานได้; slug = FE convenience auto-gen จาก `project_name` ตอน create
- workspace-scoped unique: `UNIQUE (workspace_id, slug) WHERE deleted_at IS NULL AND slug IS NOT NULL`
- format `CHECK (slug IS NULL OR slug ~ '^[a-z0-9]([a-z0-9-]{1,61}[a-z0-9])?$')` — ตรงกับ workspace slug pattern
- **public-identifier treatment (refine D32 สำหรับ project):** D32 ต้นฉบับ (workspace) ใช้ 404 (ไม่พบ) vs 403 (พบแต่ไม่ใช่สมาชิก) เพราะ workspace existence รั่วทาง create 409 อยู่แล้ว. **Project ต่าง:** member ของ ws A สามารถ probe `/wsB/projects/{guess}` แล้วแยก 404 vs 403 ได้ → enumerate project name ของ ws B ที่ตัวเองรู้ slug. **M2 collapse 403↔404 สำหรับ cross-workspace probe:** ถ้า requester ไม่มี active membership ใน ws ของ URL → return **404 เสมอ** (ทั้งกรณี project ไม่มี และกรณี project มีแต่ไม่ใช่สมาชิก ws). ภายใน ws ที่ตัวเองเป็นสมาชิก → 404 (ไม่พบ project) vs 403 (พบแต่ project_role ไม่อนุญาต) แยกตามปกติ
- **D32 email-match clause ไม่ extend มา project ใน M2** — joining project ต้องมี `workspace_membership` active; ws-invite-accept (M1) บังคับ email-match แล้ว (D32). project-scoped invitation = M3+ → ทบทวนตอนนั้น

### 2.11 FK-by-code สำหรับ workspace-scoped customizable master (refine D28 scope)

D28 ระบุ "master FK by `code` (TEXT)" สำหรับ global master. M2 ขยายให้ครอบ **workspace-scoped customizable master** ด้วย ด้วยเหตุผลเดียวกัน + 1 ข้อใหม่:

- predicate อ่านออก, ไม่ต้อง resolve, denormalize ปลอดภัย — เหมือน D28 เดิม
- **ใหม่:** `status='deprecated'` row คง `code` เดิม → FK อ้าง `code` ยัง resolve ได้ (deprecate ≠ delete, §2.9). assignment เก่าทั้งหมดยังเชื่อม row เดิม. FK-by-`id` ไม่ได้ดีกว่าตรงนี้เพราะ `code` immutable ทุก row (§2.9)
- composite FK pattern (§M2.2.1) extend ให้ FK target = `(workspace_id, code)` non-partial UNIQUE
- **Apply consistently ทุก customizable master ใน M2:** `workspace_memberships.company_position_code → company_positions(workspace_id, code)` **และ** `project_member_positions.project_position_code → project_positions(workspace_id, code)` (ไม่มี exception)

## M2.3 M2 Schema

### M2.3.1 Global / system masters (no `workspace_id`, `is_system=true`)

**`project_statuses`** — project lifecycle (vision §4.2 — 8 rows, immutable)

มาตรฐาน master §2.2:

| Column | Type | Note |
| --- | --- | --- |
| `id` | UUID PK | UUIDv7 hardcoded ใน migration |
| `code` | TEXT NOT NULL UNIQUE | `CHECK (code ~ '^[a-z][a-z0-9_]{1,62}$')` |
| `label_th` | TEXT NOT NULL | |
| `label_en` | TEXT NOT NULL | canonical English |
| `description` | TEXT | nullable; single-language (TH ใน M2; ภาษาเพิ่ม = migration column, YAGNI per D27) |
| `sort_order` | INT NOT NULL | `CHECK (sort_order >= 0)` |
| `is_system` | BOOLEAN NOT NULL DEFAULT true | M2 = ทุก row system |
| `status` | TEXT NOT NULL DEFAULT 'active' | `CHECK (status IN ('active','deprecated'))` |
| `created_at/updated_at` | TIMESTAMPTZ NOT NULL DEFAULT now() | |

- `uq_project_statuses_code` UNIQUE (`code`).
- Domain constants (Go): **per-master package** กัน collision กับ M1 (`WorkspaceStatusActive`, `MembershipStatusActive`, `AccountStatusActive` ก็มีค่า `"active"`). ใช้ `projectstatus.Draft`, `projectstatus.Planning`, `projectstatus.Proposal`, `projectstatus.Active`, `projectstatus.Closing`, `projectstatus.Maintenance`, `projectstatus.Closed`, `projectstatus.Archived` — ไม่ flat constants ที่ปนกัน package เดียว. test/lint บังคับว่า service ส่งค่า status ลง `projects.project_status_code` ต้องเป็น `projectstatus.*` เท่านั้น (DB FK ไม่จับ literal-value collision เพราะค่า `"active"` มีอยู่ทั้ง 4 master)
- **M2 = free transition** (D44; โค้ดไม่ branch validate "ไปไหนได้บ้าง"); FK บังคับแค่ "code อยู่ใน vocabulary". M3+ อาจเพิ่ม state-machine ที่ service layer (ไม่ใช่ DB CHECK) — ดู §M2.8
- seed → §M2.4

**`project_types`** — vision §4.3 (2 rows)

โครงเดียวกับ `project_statuses`. Domain constants: `projecttype.Internal="internal"`, `projecttype.Client="client"`. seed §M2.4.

- **Code branch ที่มี:** service validate "ถ้า `project_type_code='internal'` → `projects.requesting_unit` ควรไม่ว่าง" (warning, ไม่ใช่ DB CHECK — `requesting_unit` เป็น free text optional, ดู `projects` notes §M2.3.3)

**`project_roles`** — vision §6.2 (5 rows)

โครงเดียวกับ `project_statuses`. Domain constants: `projectrole.Owner="project_owner"`, `projectrole.Manager="project_manager"`, `projectrole.Member="member"`, `projectrole.Finance="finance"`, `projectrole.Viewer="viewer"`. seed §M2.4.

- **Code branch ที่มี (M5 finance read-gate, vision §6.3 / D4):** `viewer` default ไม่เห็นข้อมูลการเงิน; `finance` อ่าน/แก้ตามสิทธิ์; `project_owner`/`project_manager` เห็นการเงินตาม project role. M5 finance read-path จะ branch จาก `projectrole.*` constant — **rename code = breaking M5 contract**. M2 pin contract ที่นี่ (ดู §M2.5 invariant ข้อ 5)

> **คนละเรื่องกับ `org_roles` (M1).** `org_roles` = สิทธิ์ระดับ workspace (Owner/Admin/Executive/User, vision §6.1); `project_roles` = สิทธิ์ระดับโปรเจค (vision §6.2). 3-axis vision §6 แยกกันชัดเจน.

### M2.3.2 Workspace-scoped customizable masters (มี `workspace_id`)

**Design call — two tables, not one with category column** (เสนอ **D39**)

vision §6 แยก **Project Position** (เช่น Technical Lead, BA — บทบาทในโปรเจคหนึ่ง) ออกจาก **Company Position** (เช่น Infrastructure Manager — ตำแหน่งงานในบริษัท) เป็น 2 แกนใน 3-axis model. M2 implement เป็น **2 ตารางแยก**:

- **คนละ attachment point + cardinality:** Project Position attach ที่ `project_members` แบบ **M:N** (1 คนสวมหลาย hat ได้: TL + Architect — vision §6 ไม่ constraint 1-to-1); Company Position attach ที่ `workspace_memberships` แบบ **1:1 nullable** ใน M2 (ดู notes §M2.3.5 เรื่อง cardinality + upgrade path)
- **คนละ lifecycle + admin:** Company Position = HR/org chart stable นานๆ; Project Position = หมุนเวียนตามโปรเจค, PMO/Project Owner คุม
- **ป้องกัน category leak:** ถ้ารวมตารางเดียว + `category_code` → FK จาก `project_member_positions` กับ `workspace_memberships.company_position_code` จะอ้าง pool เดียวกัน = company title โผล่ใน project hat picker, vice versa — ต้อง guard ที่ service หรือ partial-index ด้วย `WHERE category_code='...'` (re-import enum-like predicate ที่ D28 พยายามหนี)
- **ราคา:** boilerplate schema ซ้ำ (~12 column mirrored) — ยอมรับเพราะ master pattern templated อยู่แล้ว และ duplication เป็น mechanical ไม่ใช่ conceptual

**`project_positions`** — หน้าที่ในโปรเจค (workspace-scoped customizable)

| Column | Type | Note |
| --- | --- | --- |
| `id` | UUID PK | UUIDv7 (app) |
| `workspace_id` | UUID NOT NULL → `workspaces(id)` | **isolation key** |
| `code` | TEXT NOT NULL | unique **per workspace**, immutable (§2.9), no reuse after deprecate |
| `label_th` | TEXT NOT NULL | |
| `label_en` | TEXT NOT NULL | |
| `description` | TEXT | nullable; single-language (§2.9) |
| `sort_order` | INT NOT NULL DEFAULT 0 | `CHECK (sort_order >= 0)` |
| `is_system` | BOOLEAN NOT NULL DEFAULT false | M2 ไม่ seed; ws ใหม่เริ่มว่าง |
| `status` | TEXT NOT NULL DEFAULT 'active' | `CHECK (status IN ('active','deprecated'))` — ใช้แทน soft-delete (§2.9) |
| `created_at/updated_at` | TIMESTAMPTZ NOT NULL DEFAULT now() | |
| `created_by` | UUID NOT NULL → `user_accounts(id)` | |
| `updated_by` | UUID → `user_accounts(id)` | |

Constraints/indexes:
- `CHECK (code ~ '^[a-z][a-z0-9_]{1,62}$')`
- `CREATE UNIQUE INDEX uq_project_positions_workspace_code ON project_positions (workspace_id, code)` — **non-partial** (covers ทั้ง active+deprecated); FK target ของ `project_member_positions` ตาม §M2.2.4
- `CREATE INDEX ix_project_positions_workspace_status ON project_positions (workspace_id, status)` — list picker filter `status='active'`
- **Isolation invariant §4.3:** repo method ทุกตัวรับ `workspace_id` + ใส่ใน `WHERE` เสมอ
- **Test requirement:** "workspace B สร้าง `code='tech_lead'` ได้แม้ workspace A มีอยู่แล้ว"; "workspace A query ไม่เห็น project_positions ของ workspace B"; "เพิ่ม row ที่ code ตรงกับ row deprecated ใน ws เดียวกัน → service reject (no code reuse §2.9)"
- M2 = **ไม่ seed** (ws ใหม่เริ่มว่าง); user/admin เพิ่มผ่าน CRUD ใน UI. vision §6 ตัวอย่าง (Technical Lead, ...) specific ต่ออุตสาหกรรม → ห้าม preset (vision §3 ไม่ล็อก industry)
- `is_system=true` immutability (กัน user แก้/ลบ/เปลี่ยน code) enforce ที่ **service layer** ตาม M1 §2.2 pattern — ไม่ใช่ DB CHECK

**`company_positions`** — ตำแหน่งงานในบริษัท (workspace-scoped customizable)

โครงเดียวกับ `project_positions` ทุกประการ (column + constraint mirror; เปลี่ยนชื่อ index/constraint จาก `project_` → `company_`). attach ที่ `workspace_memberships.company_position_code` (ALTER §M2.3.5).

- `uq_company_positions_workspace_code` UNIQUE (`workspace_id`, `code`) — **non-partial**; FK target ของ `workspace_memberships`
- `ix_company_positions_workspace_status` ON (`workspace_id`, `status`)
- Isolation + test ตามแบบ `project_positions`

### M2.3.3 Workspace-scoped business — `projects`

**`projects`** — entity หลักของ M2 (workspace-scoped, soft-delete)

| Column | Type | Note |
| --- | --- | --- |
| `id` | UUID PK | UUIDv7 (app) |
| `workspace_id` | UUID NOT NULL → `workspaces(id)` | **isolation key** |
| `project_name` | TEXT NOT NULL | `CHECK (length(project_name) BETWEEN 1 AND 200)` |
| `slug` | CITEXT | nullable — public identifier per ws (§2.10, D42) |
| `project_type_code` | TEXT NOT NULL → `project_types(code)` | FK master by code (§2.4) |
| `project_status_code` | TEXT NOT NULL → `project_statuses(code)` | FK master by code; **free transition** ใน M2 (D44) |
| `owner_project_member_id` | UUID | single canonical Project Owner; composite FK ด้านล่าง (D40) — **NULL ได้ชั่วครู่ใน create-tx เท่านั้น** |
| `requesting_unit` | TEXT | nullable; free text สำหรับ Internal Project (vision §4.3); `CHECK (requesting_unit IS NULL OR length(requesting_unit) BETWEEN 1 AND 200)` — DoS guard |
| `description` | TEXT | nullable; `CHECK (description IS NULL OR length(description) <= 10000)` — DoS guard (consistent กับ project_name CHECK; service-layer validation ทับเพิ่มได้) |
| `start_date` | DATE | nullable; วันเริ่มจริง |
| `end_date` | DATE | nullable; วันสิ้นสุดตามแผน |
| `created_at` | TIMESTAMPTZ NOT NULL DEFAULT now() | |
| `created_by` | UUID NOT NULL → `user_accounts(id)` | audit trail |
| `updated_at` | TIMESTAMPTZ NOT NULL DEFAULT now() | |
| `updated_by` | UUID → `user_accounts(id)` | |
| `deleted_at` | TIMESTAMPTZ | soft delete |
| `deleted_by` | UUID → `user_accounts(id)` | |

Constraints/indexes (ครบใน 6a migration `000010`, §M2.6):
- `CHECK (slug IS NULL OR slug ~ '^[a-z0-9]([a-z0-9-]{1,61}[a-z0-9])?$')` — pattern เดียวกับ `workspaces.slug`
- `CHECK (start_date IS NULL OR end_date IS NULL OR start_date <= end_date)` — date sanity
- `CREATE UNIQUE INDEX uq_projects_workspace_slug_active ON projects (workspace_id, slug) WHERE deleted_at IS NULL AND slug IS NOT NULL` — slug unique per ws ตอน active
- `CREATE INDEX ix_projects_workspace_status_created ON projects (workspace_id, project_status_code, created_at DESC) WHERE deleted_at IS NULL` — list view + status filter (executive dashboard, vision §6.1)
- `CREATE INDEX ix_projects_workspace_type ON projects (workspace_id, project_type_code) WHERE deleted_at IS NULL`
- `CREATE INDEX ix_projects_workspace_owner ON projects (workspace_id, owner_project_member_id) WHERE deleted_at IS NULL` — "โปรเจคของฉัน" view
- `ALTER TABLE projects ADD CONSTRAINT uq_projects_workspace_id_id UNIQUE (workspace_id, id)` — backing สำหรับ composite FK ของ `project_members.(workspace_id, project_id)` (ตั้ง inline ใน create-table step)
- **Owner composite FK (ใส่หลัง `project_members` exists; 6b-1 migration `000013`):**
  ```sql
  ALTER TABLE projects
      ADD CONSTRAINT fk_projects_owner_project_member
      FOREIGN KEY (workspace_id, owner_project_member_id)
      REFERENCES project_members (workspace_id, id);
      -- IMMEDIATE FK (refined D40 หลัง external review #1, 2026-05-30).
      -- column nullable → standard 3-step sequence ทำงานได้กับ IMMEDIATE
      -- โดยไม่ต้อง DEFERRABLE: INSERT projects (owner=NULL) ผ่าน (NULL no check),
      -- INSERT project_members ผ่าน, UPDATE projects SET owner_id ผ่าน (target exists แล้ว).
      -- เลิกใช้ DEFERRABLE INITIALLY DEFERRED ที่เคยอยู่ใน workflow candidate.
  ```

**Notes:**
- **`owner_project_member_id` = single canonical Project Owner column (D40, refined):** ขนาน `workspaces.owner_user_account_id` (M1 §4.3). **IMMEDIATE composite FK + standard 3-step sequence** (refined post-external-review #1): 1 tx → INSERT projects (owner_id=NULL) → INSERT project_members (creator role=`project_owner`) → UPDATE projects SET owner_project_member_id = new project_member.id → COMMIT. FK ผ่านทุก statement (NULL ตอน INSERT, member-exists ตอน UPDATE) — ไม่ต้อง DEFERRABLE.
- ทำไมไม่ derive owner จาก `project_members WHERE project_role_code='project_owner'`: (1) O(1) ownership lookup ทุก list query (ไม่ JOIN+filter); (2) ขนาน `workspaces.owner_user_account_id`; (3) single-owner ที่ระดับ column (1 FK = 1 owner); ถ้าเก็บใน project_members ต้องบังคับด้วย partial unique ที่ขัดกับ role-reassign flow
- **Service-layer guard 3 ข้อ** ตอน set/update `owner_project_member_id` (FK เป็น backstop, b/c คุม semantic ที่ FK ไม่จับ):
  1. **mutation pattern ที่ legal มีเดียว**: create flow ด้านบน หรือ ownership-transfer flow (M3+) เท่านั้น — ห้าม UPDATE owner_id raw จาก path อื่น
  2. service-layer guard ตอน set/update:
     - (a) target `project_member.workspace_id` = project.workspace_id (composite FK เป็น backstop — DB จับให้ที่ FK violation **ทันที** ตอน statement, ไม่ต้องรอ COMMIT)
     - (b) target `project_member.project_role_code = 'project_owner'` (cross-table CHECK ทำใน Postgres ไม่ได้)
     - (c) target `project_member.removed_at IS NULL` (กัน owner ที่ออกจาก project ไปแล้ว — composite FK target เป็น non-partial UNIQUE จึงไม่จับเอง §M2.3.4)
  3. test cases (§M2.5): set owner ไป non-owner role → service reject 422; set owner ไป removed member → service reject 422; UPDATE owner ไปคน ws อื่น → IMMEDIATE composite FK reject ที่ statement (proof DB backstop)
- **Role-drift guard:** ห้าม UPDATE `project_members.project_role_code` ของ row ที่ `projects.owner_project_member_id` ชี้อยู่ โดยไม่ผ่าน ownership-transfer service. M2 enforce ที่ service + test (โอกาส abuse ใน M2 ต่ำเพราะ ownership-transfer ยังไม่อยู่ scope); M3+ พิจารณา trigger ถ้า service-layer พิสูจน์ leaky
- **Owner-orphan invariant (parallel กับ M1 §4.1 workspace owner):** ห้าม transition owner's `workspace_membership` → `removed` หรือ remove จาก project_members ถ้า project ยัง status ≠ `archived`/`closed` — bind ที่ **service layer** ตอน member-removal/ownership-transfer use case (เหมือน workspace owner-orphan M1). ไม่ทำ DB trigger. ทั้ง use case ยังไม่อยู่ M2 scope → §M2.8
- **`requesting_unit` = free text (ไม่ใช่ master FK):** vision §4.3 ระบุชัด "ระบุ Requesting Unit **แทนการสร้าง type ใหม่**" → ตั้งใจให้ per-project free text. ราคา: cross-project rollup by department (vision §6.1 executive view) ไม่มี vocabulary consistency → คำถาม "งบรวมแผนก IT" ตอบไม่ได้ตรง ๆ. **ยอมรับใน M2** เพราะ vision wording explicit; ถ้า product พิสูจน์ pain → upgrade เป็น `requesting_units` workspace-scoped customizable master (M3+) — ดู §M2.8
- **`slug` = nullable, auto-gen at service:** project_id ใช้งานได้เสมอ; slug = FE convenience. service slugify จาก `project_name` ตอน create (พร้อม uniqueness check ใน ws); user แก้ได้ภายหลัง
- **Project lifecycle = free transition (D44):** FK บังคับแค่ "code อยู่ใน vocabulary"; ทุก status change เขียน `audit_logs` (action=`project.status_change`, old/new) — ตามได้ทาง log แม้ user ทำผิด. state machine = M3+ ที่ service layer
- **Soft-delete:** index ทุกตัวมี `WHERE deleted_at IS NULL`; restore = service-level (`UPDATE deleted_at=NULL`)
- **Isolation invariant §4.3:** repo รับ `workspace_id` + `WHERE` เสมอ. test "workspace อื่นมองไม่เห็น project ของ ws นี้" + "สร้าง project ที่ owner เป็น project_member ของ ws อื่น → FK reject"

### M2.3.4 Workspace-scoped business — `project_members` + `project_member_positions`

**`project_members`** — ผูก `workspace_membership` เข้ากับ `project` พร้อม project role

| Column | Type | Note |
| --- | --- | --- |
| `id` | UUID PK | UUIDv7 (app) |
| `workspace_id` | UUID NOT NULL → `workspaces(id)` | **isolation key** (denormalized; bind composite FK) |
| `project_id` | UUID NOT NULL | composite FK ↓ |
| `workspace_membership_id` | UUID NOT NULL | composite FK ↓ ("iron rule" §M2.2.1) |
| `project_role_code` | TEXT NOT NULL → `project_roles(code)` | FK by code; 1 role/member/project |
| `joined_at` | TIMESTAMPTZ NOT NULL DEFAULT now() | timestamp เข้าโปรเจค |
| `removed_at` | TIMESTAMPTZ | NULL = ยังอยู่; non-NULL = ออกจาก project (state derive จาก timestamp, §2.6) |
| `created_at/updated_at` | TIMESTAMPTZ NOT NULL DEFAULT now() | |
| `created_by` | UUID NOT NULL → `user_accounts(id)` | ใคร add member |
| `updated_by` | UUID → `user_accounts(id)` | |

Composite FKs (iron rule §M2.2.1):
```sql
FOREIGN KEY (workspace_id, project_id)
    REFERENCES projects (workspace_id, id),

FOREIGN KEY (workspace_id, workspace_membership_id)
    REFERENCES workspace_memberships (workspace_id, id)
```

Constraints/indexes:
- `CREATE UNIQUE INDEX uq_project_members_active ON project_members (workspace_id, project_id, workspace_membership_id) WHERE removed_at IS NULL` — 1 membership = 1 active row ต่อ project (กัน double-add; re-add หลัง remove = row ใหม่)
- `CREATE INDEX ix_project_members_project_role ON project_members (workspace_id, project_id, project_role_code) WHERE removed_at IS NULL` — list "ทีมโปรเจคนี้" + filter ตาม role
- `CREATE INDEX ix_project_members_membership ON project_members (workspace_id, workspace_membership_id) WHERE removed_at IS NULL` — "user นี้อยู่ project ไหนบ้างใน ws นี้"
- `ALTER TABLE project_members ADD CONSTRAINT uq_project_members_workspace_id_id UNIQUE (workspace_id, id)` — **non-partial**; FK target สำหรับ `projects.owner_project_member_id` และ `project_member_positions`

**Notes:**
- **ไม่ใช้ soft-delete แบบ `deleted_at`** — semantic ของ "remove member from project" = ออกจากทีม ไม่ใช่ "ลบ row data". ใช้ `removed_at` (timestamp-derived state ตาม §2.6) เหมือน invitation/token. ผลลัพธ์: history คง row; M3+ deliverable/task ที่อ้าง member นี้ยังเชื่อม row เดิมได้
- **ไม่มี `member_status_code` master** — vision §6.2 ไม่ระบุ vocabulary; `removed_at` เป็น single source of truth (เลี่ยง dual-source drift §2.6)
- **Suspended เฉพาะ project = ไม่ทำ M2** — over-design; suspend ที่ระดับ ws via `workspace_memberships.membership_status_code='suspended'` ครอบ (ws-suspended member filter ออกจาก project view ที่ service layer)
- **Role change = in-place UPDATE** `project_role_code` + audit_log (`project_member.role_change`, old/new); ไม่ remove+add. **Role history reconstructable จาก `audit_logs` เท่านั้น** (no temporal column ใน M2) — เพียงพอจน M5 finance/compliance ต้องการ "role as of timestamp" query ตรง ๆ จึงพิจารณา `project_member_role_history` table (M5+, ไม่กระทบ M2 schema)
- **`removed_at` กับ composite FK target:** UNIQUE `(workspace_id, id)` non-partial → removed row ยังเป็น FK target ที่ valid. service ต้องตรวจ `removed_at IS NULL` เมื่อ set `projects.owner_project_member_id` หรือเมื่อ insert `project_member_positions.project_member_id` (cross-table CHECK ทำไม่ได้). test ครอบ §M2.5
- **Isolation invariant §4.3:** repo รับ `workspace_id` + `WHERE` เสมอ
- **Test requirements (≥6 cases, ดู §M2.5):** cross-ws membership reject, cross-ws project_id reject, ws-isolation read, re-add หลัง remove, multi-ws user (worked example §M2.2.1), display_name JOIN ผ่าน workspace_memberships ของ ws เดียวกัน

**`project_member_positions`** — M:N project_member ↔ project_position (D39, FK-by-code §M2.2.4)

vision §6 ไม่ constraint 1-1 ระหว่าง member กับ project position; ทีมจริงสวมหลาย hat (TL + Solution Architect). single-FK บน project_members จะบังคับ fake "primary position" rule ที่ vision ไม่ได้พูด → M:N safe default; collapse ทีหลังถ้า product พิสูจน์.

| Column | Type | Note |
| --- | --- | --- |
| `id` | UUID PK | UUIDv7 (app) |
| `workspace_id` | UUID NOT NULL → `workspaces(id)` | **isolation key** (denormalized) |
| `project_member_id` | UUID NOT NULL | composite FK ↓ |
| `project_position_code` | TEXT NOT NULL | composite FK ↓ (FK-by-code §M2.2.4) |
| `created_at` | TIMESTAMPTZ NOT NULL DEFAULT now() | |
| `created_by` | UUID NOT NULL → `user_accounts(id)` | |

Composite FKs:
```sql
FOREIGN KEY (workspace_id, project_member_id)
    REFERENCES project_members (workspace_id, id),

FOREIGN KEY (workspace_id, project_position_code)
    REFERENCES project_positions (workspace_id, code)
    -- บังคับ member กับ position อยู่ ws เดียวกัน + FK-by-code ขนาน D28
```

Constraints/indexes:
- `CREATE UNIQUE INDEX uq_pmp_member_position ON project_member_positions (workspace_id, project_member_id, project_position_code)` — กัน assign position ซ้ำกับ member เดียวกัน
- `CREATE INDEX ix_pmp_position ON project_member_positions (workspace_id, project_position_code)` — "ใครเป็น `tech_lead` ใน project นี้/ws นี้" reverse query

**Notes:**
- **ไม่มี soft-delete; ไม่มี `updated_at`/`updated_by`** — junction row ตามชื่อบ่งบอก = append/delete only (ไม่มี UPDATE path: ถอด position = DELETE, เพิ่ม = INSERT row ใหม่). **deviation จาก §2.1 "ทุกตาราง มี updated_at"** ขนาน M1 token tables (`auth_email_verification_tokens` / `auth_password_reset_tokens`) ที่ omit ด้วยเหตุผลเดียวกัน — flag ที่ note นี้
- history (ใครเคย hold position อะไรเมื่อไหร่) อยู่ใน `audit_logs` (action=`position.assign` / `position.unassign`, project_id ฝัง — ดู §M2.3.5)
- **FK-by-code = stable across deprecate:** `deprecated` position row ยังคง code → assignment เก่ายังเชื่อม row เดิม (§2.9 + §M2.2.4). FE filter `status='deprecated'` ออกจาก picker ของใหม่
- service ต้องตรวจ `project_member.removed_at IS NULL` ก่อน INSERT (กัน assign position ให้ member ที่ออกจาก project แล้ว — composite FK ไม่จับเพราะ target UNIQUE non-partial)
- **Test (§M2.5):** "workspace อื่นเห็น assignment ของ ws นี้ไม่ได้"; "reject assignment ที่ member.ws ≠ position.ws (composite FK ปฏิเสธที่ DB)"; "assign position ของ ws อื่นให้ member → FK reject"

### M2.3.5 ALTER M1 tables

**ALTER `workspace_memberships`** — backing UNIQUE (step 1, §M2.6) + company_position (step 9 / migration 000016, §M2.6)

> **Why 2 ALTERs:** step 1 (backing `UNIQUE (workspace_id, id)`) ต้องลง **ก่อน** step 5 (`project_members` composite FK ใช้ workspace_memberships เป็น parent target). company_position (step 9, migration 000016) ต้องลง **หลัง** step 7 (`company_positions` create-table, migration 000014). FK dependency graph บังคับ split — ขนาน M1 §7 migration narrative. *(หมายเหตุ post-D45 renumber: step 9 = 000016; เลข "step 8 (000015)" ใน comment SQL ด้านล่างเดิม stale, แก้แล้ว.)*

```sql
-- step 1 (000008): backing composite UNIQUE (iron rule §M2.2.1 — FK target)
ALTER TABLE workspace_memberships
    ADD CONSTRAINT uq_workspace_memberships_workspace_id_id
    UNIQUE (workspace_id, id);

-- step 9 (000016): company_position_code (nullable; D41)
ALTER TABLE workspace_memberships
    ADD COLUMN company_position_code TEXT;

ALTER TABLE workspace_memberships
    ADD CONSTRAINT fk_workspace_memberships_company_position
    FOREIGN KEY (workspace_id, company_position_code)
    REFERENCES company_positions (workspace_id, code);
```

Index เพิ่ม: `CREATE INDEX ix_workspace_memberships_company_position ON workspace_memberships (workspace_id, company_position_code) WHERE company_position_code IS NOT NULL`.

- `company_position_code` **nullable** — M1 onboarding ไม่บังคับให้ตอบ; user เพิ่มทีหลัง. existing M1 rows = NULL (no backfill)
- Composite FK บังคับว่า company_position อยู่ ws เดียวกับ membership ("iron rule" §M2.2.1 + FK-by-code §M2.2.4 — consistent กับ `project_member_positions`)
- **Cardinality = 1:1 nullable (M2 choice, explicit trade-off):** vision §6 ตัวอย่าง singular ("Infrastructure Manager"). ราคา: ถ้าจริงๆ คนถือ "Engineering Manager + Acting Head of Platform" พร้อมกัน → M2 บังคับเลือก 1. **Upgrade path คงเปิด:** ถ้า requirement multi-position ของจริง → migration เพิ่ม `workspace_membership_company_positions` join table (โครงขนาน `project_member_positions`) + drop column + API response คืน array — ไม่ break contract (FE handle 0/1/N เสมอ). M3+ พิจารณาตามหลักฐาน
- **Ops note (production):** step 1 build UNIQUE index บน existing table. M1 ws-sized DB เล็ก lock สั้น; production-scale data ให้ใช้ `CREATE UNIQUE INDEX CONCURRENTLY` + `ADD CONSTRAINT ... USING INDEX` กัน table lock (M2 migration ใช้ inline เพราะ pre-prod)
- **Open thread (M3+ placeholder member):** ถ้า profile layer มา (`memberships.profile_id`), company_position อาจย้ายลง profile table เพราะเป็น attribute ของ "ตัวบุคคลใน ws" มากกว่า "membership row" → revisit ตอนนั้น

**ALTER `audit_logs`** — เพิ่ม `project_id` (เสนอ **D43**; ปิด M1 §8 open thread)

```sql
ALTER TABLE audit_logs ADD COLUMN project_id UUID;  -- nullable, NO FK

ALTER TABLE audit_logs
    ADD CONSTRAINT chk_audit_logs_project_implies_workspace
    CHECK (project_id IS NULL OR workspace_id IS NOT NULL);
    -- structural integrity: project-scoped event ต้องมี workspace context
    -- (อยู่ใน carve-out D29 ที่ห้าม CHECK เฉพาะ vocabulary columns ไม่ใช่ structural)

CREATE INDEX ix_audit_workspace_project_created
    ON audit_logs (workspace_id, project_id, created_at DESC)
    WHERE project_id IS NOT NULL;
```

- **nullable** — workspace-level mutation (workspace.create, member.invite) ไม่มี project context → NULL. existing M1 rows = NULL (no backfill needed; semantic ถูกต้อง)
- **NO FK to `projects(id)`** ตามเหตุผลเดียวกับ `resource_id` (M1 §4.4): (1) log append-only ต้องคงอยู่แม้ project soft-deleted หรือ hard-deleted (M3+); (2) วงจร dep ใน migration. pattern เดียวกับ `resource_id` (bare UUID)
- **CHECK `project_id IS NULL OR workspace_id IS NOT NULL`** — structural invariant: project มี ws เสมอ → project-scoped log ต้องมี ws ด้วย. D29 carve-out ห้าม CHECK เฉพาะ **vocabulary columns** (`action`/`resource_type`/`result`/`event_type`/`severity`); CHECK structural แบบนี้อยู่นอก carve-out
- **Partial index** `WHERE project_id IS NOT NULL` — workspace-level event ไม่ bloat index นี้; ใช้ `ix_audit_workspace_created` เดิมสำหรับ ws-level
- **`project_id` ไม่ใช่ isolation key — `workspace_id` ยังเป็นตัวเดียว.** read invariant (must-enforce, ดู §M2.5 invariant ข้อ 1 + 6):
  1. **Write path:** `auditRepo.LogTx(tx, entry)` รับ `workspaceID` + `projectID` แยก param. `projectID != nil` ต้องคู่กับ `workspaceID != nil` (CHECK บังคับ DB-level อยู่แล้ว แต่ repo assertion fail fast); `projectID` ต้องเป็นค่าเดียวกับ `project_id` ที่ validated against `TenantContext.WorkspaceID` ใน tx ปัจจุบัน — ห้ามรับจาก client raw
  2. **Read path:** ทุก query บน `audit_logs` ที่ filter ด้วย `project_id` ต้อง `WHERE workspace_id = ? AND project_id = ?` พร้อมกัน. ห้าม `WHERE project_id = ?` เดี่ยว. PR review = **P0 block** ถ้าเจอ
  3. ถ้า service bug เขียน `(workspace_id=A, project_id=B's-project)` → CHECK ไม่จับ (workspace_id ไม่ NULL), read ตาม invariant 2 จะไม่ surface row นั้นใน ws A's "audit by project" view เพราะ workspace_id WHERE bind = A. test ครอบกรณีนี้ (§M2.5)
- M2 mutation ใหม่ที่เขียน audit (project_id ฝัง): `project.create` / `project.update` / `project.delete` / `project.status_change` / `project_member.add` / `project_member.remove` / `project_member.role_change` / `project_position.create` / `project_position.update` / `project_position.deprecate` / `company_position.create` / `company_position.update` / `company_position.deprecate` / `position.assign` / `position.unassign`

## M2.4 Seed data (M2 system masters)

> seed master row: hardcode `id` UUIDv7 + `code` ใน migration (immutable system row, M1 pattern). FK ใช้ `code` (§2.4) → ไม่ต้อง resolve id; partial index ใช้ text literal. cache รอตอน startup เพื่อ label display ได้.
> `description` = TH only (single-language convention §M2.2.2 §2.9 / M1 §2.2). บรรทัด vision §4.2 ที่มี " / MA" suffix ถูกตัดออกจาก label เพื่อ UI สะอาด; ข้อมูล MA อยู่ใน `description` แทน — flag ที่นี่กัน drift ตรวจสอบกับ vision verbatim.

### `project_statuses` (vision §4.2 — 8 rows, `is_system=true`, `status='active'`)

| code | label_th | label_en | sort_order | description (TH) |
| --- | --- | --- | --- | --- |
| `draft` | ร่าง | Draft | 10 | เก็บข้อมูลไว้ก่อน ยังไม่ครบหรือเป็นแนวคิดเบื้องต้น |
| `planning` | วางแผน | Planning | 20 | ยังไม่เริ่มจริง แต่เตรียมทีม เอกสาร แผน งบ |
| `proposal` | เสนองาน | Proposal | 30 | เสนอราคา / ยื่นประมูล / รอลูกค้าตัดสินใจ |
| `active` | ดำเนินงาน | Active | 40 | เริ่มทำจริง มี task, team, deliverable, ค่าใช้จ่าย |
| `closing` | ปิดงาน | Closing | 50 | ส่งมอบหลักแล้ว/ใกล้จบ รอตรวจรับ เคลียร์เอกสาร |
| `maintenance` | ดูแลหลังส่งมอบ | Maintenance | 60 | ส่งมอบแล้ว แต่ยังมี MA / warranty / support (vision §4.2 verbatim) |
| `closed` | จบงาน | Closed | 70 | จบภาระผูกพันทั้งหมด |
| `archived` | จัดเก็บ | Archived | 80 | ซ่อนจากงานหลัก แต่ค้นย้อนหลังได้ |

### `project_types` (vision §4.3 — 2 rows)

| code | label_th | label_en | sort_order | description (TH) |
| --- | --- | --- | --- | --- |
| `internal` | งานภายในองค์กร | Internal Project | 10 | งานภายใน อาจมาจากแผนกตัวเองหรือแผนกอื่น (ระบุ Requesting Unit) |
| `client` | งานลูกค้า / งานรับจ้าง | Client Project | 20 | งานให้ลูกค้า / ผู้ว่าจ้างภายนอก |

### `project_roles` (vision §6.2 — 5 rows)

| code | label_th | label_en | sort_order | description (TH) |
| --- | --- | --- | --- | --- |
| `project_owner` | เจ้าของโปรเจค | Project Owner | 10 | รับผิดชอบภาพรวมของโปรเจค |
| `project_manager` | ผู้จัดการโปรเจค | Project Manager | 20 | จัดการงาน, ทีม, งวดงาน, ความคืบหน้า |
| `member` | สมาชิกโปรเจค | Member | 30 | ทำงานตามที่ได้รับมอบหมาย |
| `finance` | การเงินโปรเจค | Finance | 40 | ดู/จัดการข้อมูลการเงินตามสิทธิ์ |
| `viewer` | ผู้ดูอย่างเดียว | Viewer | 50 | ดูข้อมูลที่อนุญาต โดย default ไม่เห็นข้อมูลการเงิน (vision §6.3 / D4) |

### `project_positions` / `company_positions` — **ไม่ seed ใน M2**

workspace-scoped customizable; ws ใหม่เริ่มว่าง; user/admin เพิ่มผ่าน CRUD ใน UI (M2 FE scope). vision §6 list ตัวอย่าง (Technical Lead, Infrastructure Manager) แต่ specific ต่ออุตสาหกรรม — vision §3 ไม่ล็อก industry → ห้าม preset ใน migration. ถ้า product พบ pain ของ empty list → M3+ พิจารณา seed default ตอน `workspace.create` service (ไม่ใช่ migration seed) เพื่อให้ tenant-scoped customizable คง pattern; ตั้ง `is_system=true` กัน user แก้ code.

## M2.5 Isolation invariant mapping (M2 — extend M1 §5 + amend architecture §4.3)

Inherit ทุกข้อจาก M1 §5. M2 ขยายดังนี้:

| ข้อ (02 §4.3 หลัง amend D38) | M2 บังคับที่ไหน |
| --- | --- |
| 1. repo method ตาราง workspace-scoped รับ `workspace_id` + ใส่ `WHERE` เสมอ | `projects`, `project_members`, `project_member_positions`, `project_positions`, `company_positions`, `audit_logs` (`project_id` เป็น filter เสริม; `workspace_id` WHERE ยังเป็นตัวเดียว). repo method ที่ filter project_id ต้อง bind ทั้ง `workspace_id` + `project_id` (PR review P0) |
| 2. service ไม่รับ `workspace_id` จาก client input | M2 endpoints (`POST /projects`, `GET /projects`, `POST /projects/{id}/members`, …) resolve `workspace_id` จาก `TenantContext` (M1 §5); `project_id` จาก path/body ตรวจที่ service ว่าอยู่ใน `TenantContext.WorkspaceID` (composite FK เป็น backstop) |
| 3. test "workspace อื่นมองไม่เห็น" อย่างน้อย 1 ต่อ repo | `projects` repo, `project_members` repo, `project_member_positions` repo, `project_positions` repo, `company_positions` repo, `audit_logs` (per-project filter test) |
| **4. (NEW M2, D38 amends architecture §4.3) DB-level composite FK backstop** บน workspace-scoped business row ที่อ้าง entity ใน ws | composite FK iron rule §M2.2.1 — `project_members.(workspace_id, workspace_membership_id)` → `workspace_memberships(workspace_id, id)`, `project_members.(workspace_id, project_id)` → `projects(workspace_id, id)`, `project_member_positions.(workspace_id, project_member_id)` + `.(workspace_id, project_position_code)`, `workspace_memberships.(workspace_id, company_position_code)` → `company_positions(workspace_id, code)`, `projects.(workspace_id, owner_project_member_id)` → `project_members(workspace_id, id)` (IMMEDIATE, refined D40 post-external-review #1). test ≥ 1 ต่อ FK ที่ proof "insert ข้าม ws → DB reject" |
| **5. (NEW M2, M5 contract pin) Finance read-gate hook** | `project_roles.code` = source of truth สำหรับ finance gating ที่ M5. `viewer` (และ role อื่นที่ไม่ใช่ `project_owner`/`project_manager`/`finance`) MUST NOT เห็น finance field by default (vision §6.3 / D4). Domain constants `projectrole.*` = M5 contract — **rename code = breaking M5**. enforce ตอน M5 build; M2 pin pattern เพื่อกัน drift |
| **6. (NEW M2) Audit-by-project read discipline** | `audit_logs.project_id` ไม่มี FK + ไม่มี composite-FK backstop (เป็น append-only bare UUID เหมือน `resource_id`). compensating control = **read invariant** (§M2.3.5 ALTER notes): ทุก query filter `project_id` ต้อง `WHERE workspace_id = ? AND project_id = ?` พร้อมกัน. write invariant: `LogTx` รับ workspaceID+projectID แยก param, projectID ต้อง validated ก่อน. PR review P0 block ถ้าเจอ `WHERE project_id` เดี่ยว |

**ทำไม composite FK ปิดช่องที่ M1 invariant อาศัย service-discipline อย่างเดียว**

isolation invariant §4.3 (เดิม 3 ข้อ) พึ่ง "repo+service ใส่ `WHERE workspace_id`" ครบทุกที่ — พลาด = silent leak. M2 ยก **DB เป็น defense-in-depth (ข้อ 4 ใหม่)**: แม้ service บั๊กลืม guard, insert/update ที่ workspace_id ไม่ match parents จะถูก Postgres reject → bug กลายเป็น 500 ที่ test/log จับได้, ไม่ใช่ silent leak. นี่คือ pattern ที่ M1 external review (iron rule) ขอ adopt — ปิด M1 §8 open thread. **`audit_logs.project_id` คือ exception (ข้อ 6)** ที่ DB backstop ไม่ครอบ → compensating discipline strict กว่าและ test ครอบ proof case

### Test requirements per repo (M2 — รวมขั้นต่ำ)

- **`projectRepo`**:
  (a) list ของ ws A ไม่เห็น project ของ ws B; (b) update/delete ผ่าน ws-A context → not-found; (c) UPDATE `owner_project_member_id` เป็น project_member ของ ws อื่น → IMMEDIATE composite FK reject ที่ statement (proof DB backstop, refined D40); (d) set owner เป็น project_member ที่ `project_role_code != 'project_owner'` → service reject 422; (e) set owner เป็น project_member ที่ `removed_at IS NOT NULL` → service reject 422
- **`projectMemberRepo`**:
  (a) cross-ws membership assignment → FK reject (worked example §M2.2.1); (b) cross-ws project_id → FK reject; (c) workspace B query ไม่เห็น member ของ workspace A; (d) remove แล้ว re-add ได้ (partial unique `WHERE removed_at IS NULL`); (e) **multi-ws user**: user U เป็นสมาชิก ws A + ws B; attempt insert project_member ที่ ws-A's project P ด้วย U's ws-B membership_id → DB reject (composite FK proof); (f) JOIN `project_members → workspace_memberships → user_accounts`: returned `display_name` ต้องมาจาก membership ใน project's workspace (sanity check กัน JOIN bug ที่ดึง display_name จาก sibling membership ของ user เดียวกันใน ws อื่น)
- **`projectMemberPositionRepo`**: (a) assign position ของ ws อื่นให้ member → FK reject; (b) ws อื่นมองไม่เห็น; (c) assign position ให้ project_member ที่ `removed_at IS NOT NULL` → service reject 422
- **`projectPositionRepo` / `companyPositionRepo`**: (a) workspace A และ B สร้าง `code='tech_lead'` ได้ทั้งคู่; (b) ws อื่นมองไม่เห็น; (c) reuse code ที่ deprecated ใน ws เดียวกัน → service reject (§2.9 no-reuse rule); (d) UPDATE `code` ของ row → service reject (§2.9 code immutable)
- **`auditLogRepo` (NEW M2)**:
  (a) write `(workspace_id=A, project_id=B's-project)` ผ่าน raw INSERT (bypass service) → CHECK ผ่าน (workspace_id ไม่ NULL) — proof ว่า read-side ต้อง bind workspace_id; (b) query "audit ของ project X" ผ่าน `auditLogRepo.ListByProject(workspaceID=A, projectID=X)` ไม่ surface row จาก (a) เพราะ WHERE workspace_id=A bind; (c) attempt `LogTx` ที่ projectID != nil แต่ workspaceID == nil → repo assertion panic/error fast; (d) attempt write `project_id != NULL` + `workspace_id IS NULL` ผ่าน raw → CHECK reject
- **Standard sequence create test:** 1 tx → INSERT projects (owner=NULL) → INSERT project_members → UPDATE projects SET owner → COMMIT; assert ทุก statement ผ่าน, project มี owner_id เซตถูก (refined D40: ไม่ต้อง DEFERRABLE)

## M2.6 Migration plan (M2)

goose, ต่อจาก `000007_create_crosscutting_logs`. ลำดับแบ่งตาม **slice (Pin A — 6a contiguous block 000008-000011 + 6b contiguous block 000012-000016)**, refined post-6a dispatch — กัน gap-reservation ที่จะทำให้ directory listing มี hole + 6b รู้จริงตอนทำว่าต้องการอะไร. ลำดับ overall = **ALTER M1 backing UNIQUE ก่อน → masters → business → ALTER ที่ต้องรอ FK target**.

### Slice 6a (committed)

1. **`000008_alter_workspace_memberships_backing_unique`** — `ALTER TABLE workspace_memberships ADD CONSTRAINT uq_workspace_memberships_workspace_id_id UNIQUE (workspace_id, id);`. **Rationale:** prerequisite ของ 6b-1 (`project_members` composite FK target → workspace_memberships; migration `000012`). **Ops note:** บน M1 ws-sized DB เล็ก lock สั้น; production-scale ใช้ `CREATE UNIQUE INDEX CONCURRENTLY` + `ADD CONSTRAINT ... USING INDEX` ภายหลัง
2. **`000009_create_project_masters`** — `project_statuses`, `project_types`, `project_roles` (+ seed 8/2/5 rows) — global system master. indexes: `uq_*_code` UNIQUE
3. **`000010_create_projects`** — `projects` table; FK `workspace_id → workspaces(id)`, `project_status_code → project_statuses(code)`, `project_type_code → project_types(code)`; **indexes ใน file นี้**: `uq_projects_workspace_slug_active` (partial), `ix_projects_workspace_status_created`, `ix_projects_workspace_type`, `ix_projects_workspace_owner`, `uq_projects_workspace_id_id` UNIQUE (FK target backing). **ยังไม่ตั้ง** FK `owner_project_member_id` (project_members ยังไม่มี — ตั้งใน 6b-1 `000013`). CHECKs: `project_name`/`slug`/`requesting_unit`/`description`/`start_date<=end_date`
4. **`000011_alter_audit_logs_add_project_id`** — `ALTER audit_logs ADD COLUMN project_id UUID` (nullable, no FK per D43) + `CHECK (project_id IS NULL OR workspace_id IS NOT NULL)` + partial index `ix_audit_workspace_project_created`. ปิด M1 §8 open thread + เปิด audit-by-project read path สำหรับ 6a project mutations

### Slice 6b-1 (committed 2026-05-31)

5. **`000012_create_project_members`** — `project_members` table; composite FKs ครบ (projects, workspace_memberships — iron rule §M2.2.1); `uq_project_members_workspace_id_id` UNIQUE non-partial (FK target backing สำหรับ owner FK + 6b-2 project_member_positions). indexes: `uq_project_members_active` (partial WHERE removed_at IS NULL), `ix_project_members_project_role`, `ix_project_members_membership`
6. **`000013_alter_projects_add_owner_fk`** — `ALTER TABLE projects ADD CONSTRAINT fk_projects_owner_project_member FOREIGN KEY (workspace_id, owner_project_member_id) REFERENCES project_members (workspace_id, id);` **IMMEDIATE** (D40 refined — เลิกใช้ `DEFERRABLE INITIALLY DEFERRED`; ดู §M2.3.3 notes). depends on 000012

### Slice 6b-2 (committed 2026-05-31)

7. **`000014_create_position_masters`** — `project_positions`, `company_positions` (workspace-scoped customizable; ไม่ seed). indexes ต่อ table: `uq_*_workspace_code` UNIQUE non-partial (FK target backing), `ix_*_workspace_status`
8. **`000015_create_project_member_positions`** — `project_member_positions` junction; composite FKs ไป `project_members` (000012) + `project_positions` (000014, FK-by-code §M2.2.4). indexes: `uq_pmp_member_position` UNIQUE, `ix_pmp_position`
9. **`000016_alter_workspace_memberships_company_position`** — `ALTER workspace_memberships ADD COLUMN company_position_code TEXT` + composite FK ไป `company_positions(workspace_id, code)` (000014) + `ix_workspace_memberships_company_position` (partial)

> **Renumber note (D45, Pin A):** เดิม §M2.6 (ก่อน split) วาง 6b เป็น 000012 position_masters → 000016. เมื่อ split: **6b-1** เอา 000012 (project_members) + 000013 (owner FK); **6b-2** เลื่อนเป็น 000014 (position_masters) / 000015 (junction) / 000016 (company_position). FK ordering ยังถูก — `project_members` (000012) ไม่ขึ้นกับ position_masters; junction (000015) + company_position (000016) ขึ้นกับ position_masters (000014). **ค้างที่ migration file `000010` (committed):** inline comment ของมัน naming owner FK เป็น `000012_alter_projects_add_owner_fk` — ไม่เคยถูกต้อง (owner FK = `000013`); ไม่แก้ไฟล์ committed, flag ไว้ตรงนี้

> **ทำไมแยก 9 migrations:** (1) traceability ต่อ entity/operation (ตาม M1 pattern 000002–000007); (2) rollback granular (down ทีละขั้นถ้า issue); (3) FK ordering ชัดเจน — แต่ละ migration assume tables ก่อนหน้า exists; (4) seed master ใน file เดียวกับ CREATE TABLE master = atomic ต่อ vocabulary.
> **No backfill needed:** existing M1 audit_logs rows มี `project_id = NULL` (workspace-level events ก่อน M2 — semantic ถูกต้องโดย default + CHECK ผ่านเพราะ workspace_id อาจ NULL หรือ NOT NULL ก็ได้เมื่อ project_id NULL); membership rows มี `company_position_code = NULL`.
> **Down migration:** reverse order (000016→000008). DROP TABLE / DROP CONSTRAINT / DROP COLUMN; CI ต้องทดสอบ up/down idempotent.
> **Seed pattern:** hardcode `id` UUIDv7 + `code` ใน INSERT (immutable system row, M1 §7). FK ใช้ `code` (§2.4) → ไม่ต้อง resolve. label cache ตอน startup ได้ (optional).

## M2.7 Decisions to fold (เสนอ D38–D44 — รอ User เคาะ)

ขอ User เคาะ append เข้า `DECISIONS.md` (กฎ append-only ไม่แก้ row เก่า). D38 ต้อง **commit คู่กับ update [02-architecture §4.3](02-architecture.md)** จาก 3 invariants → 4 invariants ใน commit เดียวกับ M2 doc merge:

| # | Decision | Rationale | Section ลิงก์ |
| --- | --- | --- | --- |
| **D38** | **Composite FK iron rule — amend architecture §4.3 จาก 3→4 invariants** — ทุก workspace-scoped business row ที่อ้าง "assignment/owner/member-of-something" ใช้ composite FK `(workspace_id, *)` ไปที่ parent โดยมี backing `UNIQUE (workspace_id, id)` หรือ `(workspace_id, code)`. DB = **defense-in-depth** ข้อ 4 ของ isolation invariant (เดิม 3 ข้อพึ่ง service-discipline). **Scope exclusion:** audit/identity columns (created_by, updated_by, deleted_by, invited_by, owner_user_account_id ที่ workspace-boundary) ใช้ single-column FK → `user_accounts(id)` เพราะ actor เป็น global identity (D3 multi-ws). M2 apply: projects.owner, project_members.(project+membership), project_member_positions, workspace_memberships.company_position. M3+: task_assignee, budget_approver, deliverable_reviewer | [04 §M2.2 (2.8)](04-data-model.md) + [02 §4.3](02-architecture.md) |
| **D39** | **Positions = 2 ตารางแยก + FK-by-code consistent + junction append/delete-only** — `project_positions` (M:N ผ่าน `project_member_positions`, 1 member สวมหลาย hat) + `company_positions` (1:1 nullable column บน `workspace_memberships`, M2 trade-off). 2 ตารางป้องกัน category leak ที่ table เดียว+category_code จะเปิด. **FK strategy consistent** ทั้ง 2 customizable masters = FK-by-code ขนาน D28 (§M2.2.4); code immutable + no-reuse-after-deprecate ทำให้ stable เท่า id (§2.9). **`project_member_positions` junction = append/delete-only** (ไม่มี `updated_at`/`updated_by`; history อยู่ใน `audit_logs`) — ขนาน M1 token tables. **External review #1 disagreement (recorded):** reviewer เสนอเพิ่ม `status='active'|'revoked'` + `updated_at`/`updated_by` บน junction (เพื่อ row-level history). Kael ค้านว่าขัด §2.6 (single-source-of-truth: timestamp/status enum ไม่เก็บซ้อนกัน) — status column + audit_logs = dual source ที่ drift inevitable; audit_logs เป็น proper history mechanism อยู่แล้ว ไม่สิ้นเปลือง; ถ้า future compliance ต้องการ "as of timestamp" query → `project_member_role_history` table แยก (§M2.8). **User เลือกตาม Kael — คงเดิม.** | [04 §M2.3.2, §M2.3.4, §M2.3.5](04-data-model.md) |
| **D40** | **Project owner = column `projects.owner_project_member_id` composite IMMEDIATE FK** ไป `project_members(workspace_id, id)`. ขนาน `workspaces.owner_user_account_id` (M1 §4.3). O(1) ownership lookup; single-owner ที่ column-level. **Standard 3-step create sequence** (INSERT projects NULL owner → INSERT project_members → UPDATE projects SET owner) ทำงานได้กับ IMMEDIATE เพราะ column nullable. Service-layer guard 3 ข้อ (ws-match จาก composite FK / role-match `project_owner` / `removed_at IS NULL`) + role-drift guard ห้าม UPDATE project_role_code ของ owner row นอก ownership-transfer service. **External review #1 refinement (history):** workflow candidate ใช้ `DEFERRABLE INITIALLY DEFERRED` รองรับ chicken-and-egg ที่ COMMIT-time. Reviewer ทักว่า DEFERRABLE เพิ่ม complexity + claim deadlock risk (เหตุผล deadlock อ้าง "Postgres internal execution plan swap" ของ DML — ไม่ตรง Postgres semantics: Postgres ไม่ reorder DML statements ใน tx; INSERT row ใหม่ไม่ block cross-tx). Reviewer เสนอ swap INSERT order (member ก่อน project) — ใช้ไม่ได้กับ composite FK iron rule (`project_members.(workspace_id, project_id) → projects` IMMEDIATE จะ fail ทันที). **Underlying concern (DEFERRABLE เพิ่ม complexity) ถูก** → Kael พลิกเป็น IMMEDIATE + nullable + standard sequence: ทำงานได้โดยไม่ต้อง DEFERRABLE feature, ตรงเจตนา reviewer ที่จะเลี่ยง DEFERRABLE | [04 §M2.3.3](04-data-model.md) |
| **D41** | **Company position = ALTER `workspace_memberships`** (nullable `company_position_code` + composite FK ไป `company_positions(workspace_id, code)`, FK-by-code §M2.2.4). **Cardinality 1:1 nullable** ตาม vision §6 ตัวอย่าง singular ("Infrastructure Manager") — trade-off explicit: ถ้าจริงๆ multi-position (Engineering Manager + Acting Head) → migration upgrade เป็น join table + API คืน array (FE handle 0/1/N เสมอ, no break). M3+ พิจารณาตามหลักฐาน | [04 §M2.3.5](04-data-model.md) |
| **D42** | **Project slug = optional public identifier per workspace + cross-ws 404 collapse** (refine D32 สำหรับ project). CITEXT nullable, UNIQUE per ws among non-deleted, format pattern เดียวกับ workspace slug. **Cross-workspace probe collapse 403→404:** member ของ ws A probe `/wsB/projects/{guess}` → return 404 เสมอ (ไม่แยก "ไม่มี" vs "มีแต่ไม่ใช่สมาชิก ws B") กัน enumerate project name ของ ws B. ภายใน ws ที่ตัวเป็นสมาชิก → 404 vs 403 แยกตามปกติ. D32 email-match clause ไม่ extend มา project ใน M2 (ws-invite-accept บังคับ email-match แล้ว) | [04 §M2.2 (2.10), §M2.3.3](04-data-model.md) |
| **D43** | **`audit_logs.project_id` ALTER policy** — ปิด M1 §8 open thread. nullable + partial index `WHERE project_id IS NOT NULL` + **`CHECK (project_id IS NULL OR workspace_id IS NOT NULL)`** (structural, อยู่นอก D29 carve-out ที่ห้าม CHECK เฉพาะ vocabulary columns). **NO FK to `projects(id)`** เพราะ audit append-only ต้อง outlive resource deletion + กัน circular dep migration (pattern M1 `resource_id`). **Compensating discipline (NEW invariant #6 §M2.5):** write path `LogTx` รับ workspaceID+projectID แยก param; read path WHERE workspace_id + project_id พร้อมกัน; PR review P0 block `WHERE project_id` เดี่ยว. **External review #1 disagreement (recorded):** reviewer เสนอ runtime existence-check ใน `LogTx` (SELECT projects ก่อนบันทึก audit) เพื่อกัน "orphaned log injection". Kael ค้านว่า threat model misaligned (`LogTx` = internal repo, ไม่ exposed ต่อ client; ถ้า attacker เรียกได้ตรง = code execution อยู่แล้ว, FK ไม่ช่วย) + fix เพิ่ม +1 SELECT ทุก audit-write (hot-path overhead) + ซ้ำซ้อน service-layer validation; "FE crash" ไม่ realistic (JOIN-miss → "Unknown project" graceful fallback คือ pattern audit-outlive-resource ปกติ). read invariant + FE graceful-handle ครอบ scenario จริงพอ. **User เลือกตาม Kael — คงเดิม.** | [04 §M2.3.5, §M2.5, §M2.6 step 9](04-data-model.md) |
| **D44** | **Project lifecycle = free transitions ใน M2** (no state machine, no DB CHECK, no transition table) — FK บังคับแค่ vocabulary; ทุก status change เขียน `audit_logs`. confirm-only ตาม build-plan §5 row M2. M3+ อาจเพิ่ม transition guard ที่ **service layer** (ไม่ใช่ DB CHECK) ถ้า product พบ misuse บ่อย — review ตอน deliverable/finance entity มาผูก lifecycle | [04 §M2.1, §M2.3.1, §M2.3.3](04-data-model.md) |

## M2.8 ค้างไว้ให้ milestone ถัดไป

- **Placeholder member + claim-by-email** (M1 §8 — confirm defer) — M2 ยังคง `workspace_memberships.user_account_id NOT NULL`. M3+ ต้อง: (1) split profile (workspace-scoped, account-optional) จาก user_account (global); (2) FK target ของ `project_members.workspace_membership_id` อาจ revisit ว่าอ้าง profile หรือ membership; (3) `company_position_code` อาจย้ายลง profile. pattern อ้างอิง: GitHub commit-claim / Jira assign-before-join. verify email-match ตอน claim (เหมือน accept-invite D32)
- **Per-workspace profile** (`memberships.profile_id`, M1 §8) — defer พร้อม placeholder; ถ้าไม่ทำ placeholder ใน M3+ ก็ไม่มีค่าเพิ่ม
- **Project lifecycle state machine / transition guard** (D44) — M3+ พิจารณาถ้า product พบ invalid transition บ่อย (เช่น Draft→Archived ข้าม Active) หรือ deliverable/finance ผูก lifecycle (เช่น "ห้าม `closed` ถ้ายังมี deliverable ค้าง"). ทำที่ service layer (ไม่ใช่ DB CHECK) หรือ master `project_status_transitions(from_code, to_code)`
- **Project owner soft-delete + ownership-transfer flow** — เหมือน workspace owner-orphan §4.1; ห้าม remove project owner จาก project ถ้ายัง own project active. enforce service layer ตอน account-deletion / member-removal use case (ทั้งคู่ M3+). Trigger-based assertion ที่ DB consider ถ้า service-layer พิสูจน์ leaky
- **Project archive policy** — `archived` status vs `deleted_at` ทำงานยังไง? M2: archived = ยัง query ได้แต่ filter ออกจาก default view (status-based); deleted = soft-removed (UI ซ่อน). hard-delete archived projects เพื่อ data retention → M3+ policy
- **Project restore flow + slug uniqueness** (raised by external review #1, 2026-05-30) — M2 มีแค่ soft-delete; restore = service-level `UPDATE deleted_at=NULL` ที่ยังไม่อยู่ scope. **Edge case ตอน restore feature มา (M3+):** user soft-delete project slug=`alpha` → สร้างใหม่ slug=`alpha` (partial index `WHERE deleted_at IS NULL` รับ — ถูกต้อง) → ต้องการ restore เก่า = unique violation ที่ `uq_projects_workspace_slug_active`. M3+ ต้องตัดสิน policy 1 ใน 3 ตอน restore feature ถูกใส่: (a) prompt user rename slug ตอน restore (UX friction, ข้อมูลคงเดิม); (b) auto-mutate slug ของ deleted row เป็น `{slug}-deleted-{ts}` ตอน soft-delete (reviewer's suggestion — slug หลักว่างทันที, แต่ mutate "preserved" data → audit-trail history สับสน); (c) restore = service prompts conflict resolution interactive. **ไม่ blocker M2** (restore ไม่อยู่ scope); flag pattern พร้อมเลือกตอน M3+ deliverable/restore work
- **Project member suspended state** — M2 = ไม่มี; suspend ที่ระดับ ws ครอบ. ถ้า "อยู่ใน project แต่ห้ามทำอะไร" use case จริง → เพิ่ม `suspended_at` ใน project_members (single source of truth §2.6)
- **`requesting_unit` upgrade to master** — ถ้า product พิสูจน์ value (group/report by แผนกใน executive dashboard vision §6.1) → upgrade เป็น `requesting_units` workspace-scoped customizable. M2 ไม่ทำ (vision §4.3 wording explicit "ระบุ ... แทนการสร้าง type ใหม่" → free text intent); YAGNI จนกว่า data จะพิสูจน์
- **Position seed default per workspace** — M2 ไม่ seed (vision §3 ไม่ล็อก industry). M3+ พิจารณา seed ตอน `workspace.create` service (ไม่ใช่ migration) หรือผูกกับ industry template; `is_system=true` กัน user แก้ code
- **Multi-company-position ต่อคน** (D41 trade-off) — M2 inline column 1:1. ถ้า requirement multi-position จริง → migration เพิ่ม `workspace_membership_company_positions` join table + drop column. ไม่ break API contract (response array)
- **Project-scoped invitation** — flow ปัจจุบัน = ws invite + add to project (2-step). ถ้า "เชิญตรงเข้า project พร้อมสร้าง ws membership" use case → unified invitation pattern M3+
- **Project member bulk operation** — add/remove หลายคนพร้อมกัน → service-level API M3+ ถ้าจำเป็น
- **`project_member_role_history` table (M5+ compliance)** — M2 role history reconstruct จาก `audit_logs` (`project_member.role_change`). ถ้า M5 finance/compliance ต้องการ "role as of timestamp" query ตรงๆ → เพิ่ม table; ไม่กระทบ M2 schema
- **Date-range business rule** — M2 = CHECK soft (`end_date >= start_date`); product อาจต้องการ "Closing → Closed ต้องมี actual_end_date" → service layer M3+
- **Task assignee / budget approver FK** — apply iron rule D38 (composite FK ไป `workspace_memberships`) ตอน M4/M5; ห้ามเปิดทาง `user_accounts(id)` shortcut. test_assignee จะมี `removed_at` check pattern เดียวกับ owner_project_member_id
- **`projects.slug` anti-enumeration timing pad** — M2 = D42 (404 collapse cross-ws, distinct within-ws); review ตอน security pass
- **RLS hardening** (M1 §8 ต่อ) — composite FK ของ D38 ช่วย Policy `workspace_id = current_setting('app.workspace_id')` natural fit; ทำตอน M3+ คุ้มกว่า (table มากขึ้น + pattern ครบ). RLS สำหรับ `audit_logs` ที่มี nullable `workspace_id` ต้องคิด policy แยก (workspace-level events ไม่มี ws context → admin-only)
- **Hot-path JOIN ของ list project_members + workspace_memberships + user_accounts** — display_name ต้อง join 3 ตาราง. profiling พบ slow → cache หรือ denormalize `display_name_cached` (M1 GORM ใช้ได้ตาม D11)
- **Audit log UI** (M1 §8 ค้าง) — M2 มี `project_id` แล้ว → query "log per project" พร้อม. UI = append เมื่อ User เปิด
- **`requireSession` JOIN optimization + multi-identity/MFA revocation semantics** — carried from M1 §4.1 (D36 open thread)
- **Backing UNIQUE pattern สำหรับ M3+ entities** — ถ้า M3+ entity (deliverable, task) มี composite FK ไป `user_accounts(workspace_id, id)` pattern (ซึ่งจะผิดเพราะ user เป็น global) — confirm scope exclusion D38 ไว้กันความเข้าใจผิด
