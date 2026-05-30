# 02 — Architecture (Backend)

> **Status: Canonical.** เอกสารนี้กำหนด architecture ฝั่ง backend ของ Prasankit v2 — tech stack, layering, tenant model/isolation, และ tenant resolution/auth.
> เป็น **design intent** ที่ derive มาจาก [01-vision.md](01-vision.md) — ถ้าขัด vision แปลว่าต้องแก้ vision ก่อน (gate [00-workflow.md](00-workflow.md) §8) ไม่ใช่ปล่อยให้ architecture เดินสวน vision.
> การเปลี่ยนเอกสารนี้ทำตาม gate เดียวกับ workflow §8 (User เริ่ม → Kael trade-off → option external reviewer → User decide → แก้ doc + commit รอบเดียว).

## 1. ขอบเขตเอกสารนี้

**อยู่ในเอกสารนี้:** tech stack (§2), hexagonal layers (§3), tenant model + isolation strategy (§4), tenant resolution + auth (§5), cross-cutting concerns ระดับ architecture (§6), data-modeling conventions (§7), API conventions (§8).

**ยกไปเอกสารถัดไป (ยังไม่ตัดสินในนี้):**
- **Data model เต็ม** (ตาราง, FK, index ต่อ module) — เอกสาร data-model ชุดถัดไป
- **Permission matrix รายละเอียด** (org role × project role × action) — เอกสารสิทธิ์ ([01-vision §6](01-vision.md))
- **Frontend stack** (framework, การคุยกับ API) — ยกไป build-plan (**D14**) เพราะรอบนี้ตั้ง backend ให้แน่นก่อน แบบเดียวกับ v1 ที่เป็น API-only
- **Scope demo แรก / ลำดับ build** — build-plan ([01-vision §10](01-vision.md), **D2**)

**Decisions ที่ fold เข้าจากรอบนี้:** D11 (stack), D12 (isolation), D13 (resolution/auth), D14 (FE deferred), D15 (no-enum/master table §7), D16 (authz per-request §5.5) — ดู [DECISIONS.md](DECISIONS.md).

## 2. Tech Stack — inherit v1 (D11)

v2 **rebuild เรื่อง process/loop ไม่ใช่ tech** (vision §2). stack ของ v1 ทันสมัยและทีมรู้จักดีอยู่แล้ว จึง inherit ทั้งชุดเป็น baseline; การ rebuild เกิดที่ "วิธีผลิต code" (spec → dispatch → review) ไม่ใช่การเปลี่ยนเครื่องมือ.

| Layer | Choice | Version (v1) | ทำไม |
| --- | --- | --- | --- |
| Language | Go | **1.26** (v1=1.25, bump D21) | static, เร็ว, deploy ง่าย, ทีมรู้จัก |
| HTTP framework | Fiber | v3 | เบา, middleware chain ชัด |
| DB | PostgreSQL | 18 | relational + CITEXT + RLS (ทางเลือก hardening §4) |
| DB driver | pgx | v5 | native Postgres driver |
| ORM/Query | GORM | v1.31 | ของ v1 — data-model doc อาจทบทวน (เช่น ย้าย hot path ไป pgx/sqlc) |
| Cache / Session | Redis | 8 | session store, rate limit |
| Object storage | MinIO | latest | File Center / attachment (S3-compatible) |
| Password | bcrypt (`golang.org/x/crypto`) | — | one-way hash |
| Primary key | **UUIDv7** (`uuid.NewV7`) | — | sortable, time-ordered, ไม่ leak ลำดับแบบ serial |
| Session token | opaque random (`pkg/securetoken`) | — | 32-byte random, เก็บ **hash (sha256)** ฝั่ง server (§5) |

> **หมายเหตุ GORM:** inherit ตาม v1 ในรอบนี้. ถ้า data-model doc พบ query ที่ ORM คุมยาก (เช่น tenant-scoped join หนักๆ, RLS) ให้เสนอย้ายเฉพาะจุดเป็น pgx/sqlc — เป็น decision ของ doc นั้น ไม่ใช่ในนี้.

## 3. Hexagonal Architecture (ports & adapters)

โครงเดิมของ v1 ใช้ต่อ — แยก domain ออกจาก I/O เพื่อให้ test ได้โดยไม่พึ่ง Postgres/Redis จริง.

```
transport/http  (driving adapter — รับ request, auth, แปลง envelope)
      │  เรียก
      ▼
modules/<x>/<x>svc  (application service — orchestrate use case)
      │  พึ่ง port (interface)        ▲ domain entity + rule อยู่ใน modules/<x>
      ▼                               │
adapters/  (driven adapter — postgres, redis, minio, smtp ทำตาม port)
      │
      ▼
bootstrap/  (composition root — wiring จริงทั้งหมดอยู่ที่นี่ที่เดียว)
```

| Layer | ที่อยู่ (v1) | หน้าที่ | กฎ |
| --- | --- | --- | --- |
| Domain + Service | `internal/modules/{auth,workspace,project,task,...}` (+ `*svc`) | entity, business rule, use-case orchestration, **นิยาม port (interface)** | ห้าม import `adapters/` หรือ `transport/` — domain ไม่รู้จัก HTTP/SQL |
| Driving adapter | `internal/transport/http/*` (+ `middlewares`, `presenter`) | รับ HTTP, auth/tenant, validate input, เรียก service, render envelope | ไม่มี business rule — แค่ในออก-แปลง |
| Driven adapter | `internal/adapters/{database,cache,storage,email}/*` | implement port ของ service (Postgres repo, Redis session, MinIO) | ขึ้นกับ service interface ไม่ใช่ทางกลับ |
| Shared util | `pkg/{ids,securetoken,passwordhash,dbtypes}` | pure utility ไม่มี state ของ domain | import ได้จากทุก layer |
| Composition root | `internal/bootstrap/{app,db,http,redis,storage}.go` | สร้าง + wire dependency ทั้งหมด | layer เดียวที่รู้จัก concrete adapter ทุกตัว |

**Dependency rule:** การพึ่งพาชี้เข้าหา domain เสมอ. service นิยาม interface ที่มันต้องการ, adapter ทำตาม. ทำให้ทุก service unit-test ได้ด้วย fake adapter.

**Tenant flows ทุก layer:** `TenantContext` (§5) ถูกสร้างที่ transport แล้วส่งลงทุก service call และทุก repo query — ไม่มี query ไหน run โดยไม่มี tenant scope (§4).

**Adapter hygiene (GORM repo):** repo เก็บแค่ base `*gorm.DB` — **ห้ามเก็บ instance ที่ติดเงื่อนไข** (`db.Where(...)`) ไว้ใช้ซ้ำ. ทุก method เริ่มจาก `db.WithContext(ctx)` เสมอ ด้วย 2 เหตุผล: (1) **propagate context** — cancellation/timeout/tracing (เหตุผลหลัก); (2) ได้ Statement instance สะอาด (GORM "new session") กัน WHERE clause รั่วข้าม query — *backstop* ให้ isolation invariant (§4.3). หมายเหตุ: นี่คือ in-memory query-builder state ของ `*gorm.DB` **ไม่ใช่ connection pool** (`database/sql` จัดการ pool แยก, query clause ไม่ทำให้ pool เปื้อน).

## 4. Tenant Model & Isolation (D12)

### 4.1 โครงสร้าง
- **workspace = tenant = องค์กร, 1:1** (vision §4.1). v1 มี `workspaces.tenant_id` + `UNIQUE(tenant_id)` ยืนยัน 1:1 นี้.
- **User ↔ workspace = many-to-many** ผ่าน `workspace_memberships` (vision §4.1, **D3**) — 1 identity อยู่ได้หลาย workspace.
- ทุกตารางที่เป็นข้อมูลของ tenant ถือ **`tenant_id`** เป็น isolation key.

> **ค้างไว้ให้ data-model doc:** v1 พก `tenant_id` แยกจาก `workspace_id` แม้เป็น 1:1 (redundant, เสี่ยงสับสนว่า filter อันไหน). data-model doc ต้องตัดสินว่าจะ collapse (`tenant_id` ≡ workspace boundary ตัวเดียว) หรือคงสองคอลัมน์พร้อมเหตุผล future-proof. **ในนี้ขอใช้คำว่า "tenant_id = isolation key" เป็นหลักการ ไม่ผูก schema.**

### 4.2 Isolation strategy: shared DB, shared schema, tenant_id column
DB เดียว schema เดียว ทุกตารางมี `tenant_id`. เลือกแบบนี้เพราะ:
- **ถูก + เร็วต่อ demo และต่อ SME หลายราย** — ไม่ต้อง provision DB/schema ต่อ tenant
- **migration ครั้งเดียวจบ** — ไม่ fan-out ตาม tenant (ข้อเสียหลักของ schema-/DB-per-tenant)

**ราคาที่จ่าย:** isolation พึ่ง app-level discipline — ถ้าลืม filter `tenant_id` = ข้อมูลข้าม tenant รั่ว. จึงต้องมี **isolation invariant** บังคับ.

### 4.3 Isolation invariant (กฎเหล็ก กัน data leak)
1. ทุก repo method ที่แตะตาราง tenant-scoped **ต้องรับ `tenant_id`** (ผ่าน `TenantContext`) และใส่ใน `WHERE` เสมอ — ไม่มี method ที่ query ข้ามทั้งตารางโดยไม่ระบุ tenant (ยกเว้น admin/migration path ที่ระบุชัดและ test คุม)
2. service ไม่รับ `tenant_id` จาก input ของ client — รับจาก `TenantContext` ที่ middleware resolve แล้วเท่านั้น (§5)
3. test ของแต่ละ repo ต้องมี case "tenant อื่นมองไม่เห็น" อย่างน้อย 1
4. **(M2, D38) DB-level composite FK backstop** บน workspace-scoped business row ที่อ้าง assignment/owner/member-of-something ภายใน ws — composite FK `(workspace_id, *)` ไปที่ parent โดยมี backing `UNIQUE (workspace_id, id)` หรือ `(workspace_id, code)`. **ขจัด silent leak จาก service lapse:** insert/update ที่ `workspace_id` ไม่ match parent ถูก Postgres reject (FK violation = 500/test fail) ไม่ใช่ silent cross-tenant leak. **Scope exclusion:** audit/identity columns (`created_by`, `updated_by`, `deleted_by`, `invited_by`, `owner_user_account_id` ที่ workspace-boundary level) ใช้ single-column FK → `user_accounts(id)` เพราะ actor เป็น global identity (D3 multi-ws). รายละเอียดต่อ entity + worked example ดู [04-data-model §M2.2 (2.8)](04-data-model.md)

### 4.4 Defense in depth (future hardening)
รอบนี้ isolation อยู่ที่ app-level (invariant §4.3). **Postgres Row-Level Security (RLS)** เป็น hardening ชั้นสองที่ Postgres 18 รองรับ — เปิดทีหลังได้โดยตั้ง session var `app.tenant_id` ต่อ connection แล้ว policy บังคับ `tenant_id = current_setting(...)`. **ยังไม่ทำในรอบนี้** แต่ stack เลือกไว้ให้ทำได้ — เป็น candidate ของ build-plan/security doc.

## 5. Tenant Resolution & Auth (D13)

หลักการ vision §4.1: **resolve tenant ฝั่ง backend เท่านั้น ไม่เชื่อค่าจาก client.** v1 แก้ไว้สะอาดแล้ว — v2 inherit.

### 5.1 Authentication — opaque session (ไม่ใช่ stateless JWT)
- login สำเร็จ → server สร้าง **opaque token** สุ่ม 32 byte (`pkg/securetoken`), เก็บ **เฉพาะ hash (sha256)** ของ token ใน Redis (`sessionstore`) เป็น `SessionRecord` พร้อม TTL
- token ดิบส่งกลับใน **cookie** (httpOnly + secure + SameSite) — client เก็บ token จริงไม่ได้อ่านจาก JS
- ทุก request: middleware เอา token จาก cookie → hash → lookup Redis → ได้ account
- **เหตุผลเลือก session ไม่ใช่ JWT:** revoke ได้ทันที (ลบ key ใน Redis), ไม่มีปัญหา token ค้างอายุ, payload ไม่อยู่ฝั่ง client

### 5.2 Tenant resolution — X-Workspace-Slug header
- client ส่ง header **`X-Workspace-Slug`** บอกว่าจะทำงานใน workspace ไหน (1 identity อยู่ได้หลาย workspace — D3)
- backend `TenantResolver` (`ResolveTenantContext`): เอา account จาก session + slug จาก header → **เช็ค `workspace_memberships` ว่า account นี้เป็นสมาชิก active ของ workspace นั้นจริง** → ถ้าใช่ คืน `TenantContext`
- **เลือก header ไม่ใช่ path/subdomain เพราะ:** เหมาะกับ SPA, สลับ workspace ไม่ผูกกับ URL/DNS, ไม่ชน non-goal เรื่อง custom domain. ข้อแลก: URL ไม่ deep-link ราย workspace — ยอมรับได้รอบนี้ (FE ยกไป build-plan)

### 5.3 Middleware chain (บังคับใช้ก่อนถึง handler)
```
requireSession            → มี session ใช้ได้ไหม (cookie → Redis)
    ↓
requireTenantContext      → slug + membership valid ไหม → สร้าง TenantContext
    ↓
requireWorkspacePermission(perm)  → role ใน workspace ทำ action นี้ได้ไหม
    ↓
handler                   → รับ TenantContext (ไม่รับ tenant_id จาก body)
```

### 5.4 สิ่งที่ **ไม่เชื่อ**จาก client (เด็ดขาด)
`tenant_id` / `workspace_id` ใน body · org role / project role · membership · ownership.
ทั้งหมด resolve/verify จาก session + DB ฝั่ง backend. client บอกได้แค่ "ฉันอยากทำงานใน workspace slug นี้" — backend เป็นคนตัดสินว่าได้ไหมและในฐานะอะไร.

### 5.5 Authz resolve per-request — ไม่ cache ใน session (D16)
membership / org role / project role **resolve ใหม่ทุก request** จาก DB ตาม `TenantContext` ที่ middleware สร้าง — **ไม่ denormalize เข้า `SessionRecord`**. `SessionRecord` ถือแค่ account identity (เหมือน v1). เหตุผล:
- **revoke ทันที** — ถอด role / เตะออกจาก workspace มีผลทันที ไม่รอ session หมดอายุ. caching authz เข้า session = เอา stale-payload ของ JWT กลับมาฝั่ง server เอง → undercut เหตุผลที่เลือก opaque session แทน JWT (§5.1)
- per-request `X-Workspace-Slug` (§5.2) ทำให้ "active workspace/role" ไม่ใช่ค่าระดับ session; project role เป็น per-(user, project) ไม่ fit session granularity
- lookup membership = index hit ราคาถูก. ถ้า profiling พบ hot path → เพิ่ม **cache แยก key `(account, workspace)` ที่ invalidate ได้ตอน role/membership เปลี่ยน** ไม่ใช่ยัดใน session token

## 6. Cross-cutting (map vision §8 ลง architecture)

| Concern (vision §8) | บังคับใช้ที่ layer ไหน |
| --- | --- |
| **i18n** | presenter ส่ง **error `code`** (machine) ไม่ใช่ข้อความตายตัว → FE map เป็นภาษา. label/status/menu แปลฝั่ง FE จาก code/enum |
| **Audit-ready log** | service layer เขียน log เชิงโครงสร้าง (actor, action, resource type/id, old/new, tenant_id, project_id, ts, IP, UA, result) — ไม่ใช่ข้อความลอย. IP/UA/request-id มาจาก transport |
| **Finance visibility** | บังคับที่ **service + query** ตาม project role (vision §6.2: Viewer default ไม่เห็นการเงิน) — ไม่ใช่ซ่อนแค่ฝั่ง FE |
| **Security-by-design** | isolation invariant (§4.3), opaque token เก็บ hash (§5.1), bcrypt password, rate limit (`adapters/cache/redis/ratelimit`), validate input ที่ transport boundary |

## 7. Data Modeling Conventions (D15)

**กฎหลัก: ไม่เก็บ enum ในทุกกรณี.** controlled vocabulary (status, type, role, category, ฯลฯ) ทุกตัวเป็น **master table + FK** เสมอ — ห้าม Postgres `ENUM` type และห้าม `CHECK (col IN (...))` แทน vocabulary.

**ทำไม:**
- **i18n (vision §8)** — master table ถือ label หลายภาษาได้ (โครงสร้าง column `label_th/label_en` vs translation table = data-model doc ตัดสิน)
- **ขยายได้โดยไม่ migration** — เพิ่ม row แทน `ALTER TYPE` — จำเป็นกับ per-project custom task status (**D5**) ที่เป็น enum ไม่ได้อยู่แล้ว
- **ถือ metadata ได้** — `sort_order`, color, description, deprecate ได้โดยไม่ลบของเก่า
- **referential integrity + consistency** — FK บังคับค่าถูกต้อง, ไม่มีคำถาม "อันนี้ enum หรือ table"

**ต่างจาก v1:** v1 ทำผสม — `workspace_roles` เป็น master table (มี `code`+`is_system`+seed) แต่ `workspaces.status`/`mode` ใช้ `CHECK (... IN ...)`. v2 = master table **ทุกที่**, ยก pattern ของ `workspace_roles` ขึ้นเป็นมาตรฐานเดียว.

**ทุก master table มีอย่างน้อย:**

| Column | ความหมาย |
| --- | --- |
| `id` | UUIDv7 PK |
| `code` | stable, unique, immutable — **โค้ดอ้าง vocabulary ผ่าน `code` เท่านั้น** ไม่ใช่ id หรือ label |
| `name` + i18n label | ชื่อแสดงผลหลายภาษา |
| `sort_order` | ลำดับแสดงผล |
| `is_system` | `true` = ระบบใช้ในการตัดสินใจ, **user แก้/ลบ/เปลี่ยน `code` ไม่ได้** |
| `status` | `active` / `deprecated` (เลิกใช้โดยไม่ลบ เพื่อคง referential history) |

**Gotcha ที่กฎนี้กันไว้:** ค่าที่ **โค้ดต้อง branch** (เช่น 3 universal category `To Do/In Progress/Done` — D5; `membership.status`; project lifecycle status — vision §4.2) ถึงย้ายเป็น master table แล้ว โค้ดก็ยังต้อง "รู้จัก key" → row พวกนี้ **ต้อง `is_system=true` + `code` immutable** ให้โค้ดอ้างได้อย่างปลอดภัยและ user ทำพังไม่ได้.

**Domain constants สำหรับ system row:** domain layer นิยาม constant ของ `code` ที่โค้ด branch (เช่น `const TaskCategoryDone = "done"`) — branch จาก constant ไม่ใช่ hardcode label/id. ถ้า FK อ้าง master row ด้วย `id` (UUID) ให้ driven adapter resolve `code → id` และ **cache ตอน startup** ได้ เพราะ system row immutable → ไม่ต้อง I/O ซ้ำราย request. (FK จะอ้าง `code` หรือ `id` = data-model doc)

**Master table 2 ชนิด (data-model doc จัดประเภทรายตาราง):**
- **Global/system** — ใช้ร่วมทุก tenant (เช่น org role, universal category). `is_system=true`, **ไม่มี `tenant_id`**
- **Tenant/project-scoped customizable** — user เพิ่ม row เองได้ (เช่น task status รายโปรเจค — D5). **มี `tenant_id`** และอยู่ใต้ isolation invariant (§4.3); default row ที่ seed ตอนสร้าง = `is_system=true`

## 8. API Conventions (จาก v1 presenter)

- **Success (item):** `{"data": <obj>}`
- **Success (list):** `{"data": {"items": [...], "pagination": {...}}}`
- **Error:** `{"error": {"code": "<machine_code>", "message": "<human>", "details?: ...}}`
- **Pagination:** offset-based — `?page=&limit=` (default 10, **max 100**) → `OffsetPagination{page,limit,total,total_pages,has_more}`
- **Central error handler** + **request-id middleware** ต่อทุก request (`transport/http/middlewares`)
- **PK = UUIDv7** ทุกตาราง; password = bcrypt; session token = opaque hashed

## 9. ความสัมพันธ์กับเอกสารอื่น

- [01-vision.md](01-vision.md) — เอกสารนี้ derive มาจาก vision; module เป้าหมายอยู่ vision §5
- [00-workflow.md](00-workflow.md) — gate การเปลี่ยน architecture; pattern review ตอน dispatch Sonnet (§5–§6)
- **data-model doc (ถัดไป)** — schema เต็ม, resolve `tenant_id` vs `workspace_id` (§4.1), index, FK, **apply master-table convention + จัดประเภท global vs tenant-scoped (§7)**
- **build-plan (ถัดไป)** — scope demo แรก, FE stack (D14), ลำดับ build, candidate RLS (§4.4)
- `archive/v1-legacy` — reference implementation ของ pattern ในนี้ (`git show archive/v1-legacy:app-api/...`) — ไม่ใช่ข้อผูกมัด
