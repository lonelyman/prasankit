# 99 — Handoff (อ่านก่อนเพื่อนเมื่อเปิด session ใหม่)

> บอก "ตอนนี้อยู่ตรงไหน + ทำอะไรต่อ". เป็น **living doc — เขียนทับทุกครั้งที่ปิด session**.
> รายละเอียดเต็มอยู่ใน docs ที่ลิงก์. log การตัดสินใจอยู่ใน [DECISIONS.md](DECISIONS.md).

## อัปเดตล่าสุด
**2026-06-05** — **M2 ปิดครบทั้ง milestone — committed + pushed `origin/dev`. ก้าวต่อ = M3.** ปิด M2 FE สำเร็จ (FE-A/B/C1/C2 + Teal theme); BE 6a/6b-1/6b-2 ครบก่อนหน้า. รอบนี้: **FE-C2 (`5062ea9`)** = section "ตำแหน่งของสมาชิก" บน project detail (**แยก component `PositionsSection`**) — assign/unassign project-position M:N chips + company-position Select inline (ws-level 1:1, ค่าจาก Decision-A) + empty-state→Settings + RBAC + optimistic value; **Decision-A (`4e1e388`)** = additive read-add `company_position_code` บน `projectmember.ListByProject` (non-breaking, omitempty) + integration test. **FE-C1 (`fdee34f`)** = masters CRUD `/settings/positions` (fetch สด ไม่ hardcode). **Theme (`f557402`)** = Teal token (ui-ux-pro-max) + fix font Arial→Jakarta/Noto-Thai. **3 post-smoke fixes:** nav-leak บน /login (`a12a373`, gate nav ด้วย `status==="authenticated"`), password show/hide (`3dea31b`, `PasswordInput`), **dev-seeder `make seed` (`36fade1`)**. **D53-D55.** **User manual smoke ผ่าน** (theme+FE-C1+FE-C2+fixes). ⚠️ implementer ทั้ง FE-C = **main-thread Opus** (User ปฏิเสธ sub-agent → author=reviewer, ชดเชย `/scrutinize` — ต่าง D37, User override). **ก้าวต่อ = M3 (Deliverable + ตรวจรับ — wedge ครึ่งแรก): design data-model §M3 → BE → FE.**

> **Smoke (dev) login — อย่าอ้าง `demo@` (ไม่มีในฐาน):** บัญชีจริงใน dev DB = `owner@prasankit.local` (owner) + `bob@prasankit.local` (user) บน ws `prasankit` · รหัส `demopass123`. **`make test` ล้าง DB ทุกครั้ง (TRUNCATE) → ถ้า login ไม่ได้ ให้ `cd app-api && make seed` (กู้ owner@/bob@/ws ผ่าน repo จริง, active + bcrypt) ก่อนบอก User.** login gate ที่ `user_accounts.account_status_code='active'` ไม่ใช่ email-verified. web :13000 / api :18080 / Mailpit :8025 / api CORS อนุญาตเฉพาะ :13000 (dev server :3000 จะโดน block).

> **บทเรียน FE-A (→ memory):** (1) **manual smoke ของ User จับ bug ที่ automated จับไม่ได้** — CORS `X-Workspace-Slug` (cross-origin), input "จม" (dark-mode), forced-invite/too-many-clicks UX. vitest/build/serve-200 เขียวหมดแต่ไม่ครอบ cross-origin CORS + visual + flow → **manual click-through ทุก FE slice**. (2) reviewer overturn ครั้งที่ 2 (PUT-omit-clear): GORM `Updates(map[string]any)` เขียนทุก key รวม nil = full-replace (ต่าง `Updates(struct)` ที่ skip zero) → **อ่าน source จริงก่อนรับ HIGH**. (3) IA: เปิดแอปต้องเจองานเลย (projects); workspace picker/home-landing = hop ส่วนเกิน. (4) project handler emit error `details` เป็น object `{fields}`/`{field}` ต่างจาก M1 ที่เป็น bare array.

> **บทเรียน external review (→ memory):** brief ต้องสั่ง reviewer ยืนยัน/ถามหาไฟล์ก่อนรีวิว + ใส่ fingerprint ให้ self-detect blind review (round 1 ของ 6b-1 รีวิวโดยไม่เห็นไฟล์ → identifier ผิดหมด).

## สถานะตอนนี้
- ✅ docs ฐานครบ: [00-workflow](00-workflow.md) · [01-vision](01-vision.md) · [02-architecture](02-architecture.md) · [03-build-plan](03-build-plan.md) · [04-data-model](04-data-model.md) (รวม **§M2 ครบ — projects/team/positions/audit-by-project**) · [DECISIONS](DECISIONS.md) (**D1–D55**)
- ✅ **M1 = Identity & Tenancy ปิดครบทั้ง milestone** — BE auth/account-safety/workspace/invitation + FE 5a auth (signup/login/verify/reset) + FE 5b workspace (list/create/accept-invite/home). proof-of-loop รอบแรกพิสูจน์ vertical slice ใช้ได้จริง
- ✅ **Workflow §2 อัปเป็น Opus 4.8 hybrid (D37, cadeb6a)** — feature/module ใหญ่ = dispatch Opus แยก instance (role implementer) คง คนเขียน≠คนรีวิว · งานเล็ก/iterate/debug = Opus main-thread เขียนตรง · Sonnet = option เฉพาะ trivial
- ✅ **M2 data-model design folded (06f284f)** — §M2 ใน [04-data-model.md](04-data-model.md): conventions §2.8-§2.11 (composite FK iron rule, ws-scoped customizable master, project slug 403→404, FK-by-code customizable) · schema projects/project_members/positions/junction + ALTER ws_memberships/audit_logs · isolation invariant ขยายเป็น 6 ข้อ · migration plan Pin A (6a 000008-000011 + 6b 000012-000016). D38-D44 ครบ
- ✅ **M2 Phase 1 = data-model done** · ✅ **M2 Phase 2 step 6a-BE done (feb3f91)** — projects table + 3 masters + audit project_id + project module (domain/service/repo/handler) + middleware + routes; 43 project tests pass, gates ครบ, smoke e2e 13/13 (รวม D43 #6 SQL proof)
- ✅ infra-bug fix (696c997) — `.env.example` inline comment → own-line (กัน make-export pollution)

## วิธีรัน / เทสต์ (สำคัญ — กันงงรอบหน้า)
- `make dev-up` (infra→migrate→BE→FE→health). `make down` ปิด. `make api-up` rebuild api เฉพาะ
- **host ports:** API `18080`, Web `13000`, pg `15433`*, redis `16380`*, minio `19000/19001`, **Mailpit SMTP `11025` / UI-API `18025`** (`*`=local `.env` ต่างจาก example เพราะชนเครื่อง dev)
- **เทสต์ ต้องใช้ `cd app-api && make test`** (= `go test -p 1 ./...`) — **ห้าม `go test ./...` เปล่า ๆ** (integration ใช้ DB+Redis+Mailpit ร่วม + `TRUNCATE` → ขนานตีกัน flake). ⚠️ การ์ดเตือน: implementer สรุปว่า "เสร็จ" บางทีไม่จบจริง — **รัน `make test` + อ่าน diff เองทุกครั้ง**; เทสต์ integration ที่ "cached/skip" ให้ force-run `-count=1 -v` ดูว่า PASS จริงไม่ใช่ skip
- **⚠️ `make test` ล้าง dev DB:** integration test ใช้ DB:15433 เดียวกับแอป + `TRUNCATE` teardown → **owner@/bob@/ws/projects หายทุกครั้ง**. รัน `cd app-api && make seed` กู้คืน (owner@prasankit.local + bob@prasankit.local / `demopass123` บน ws `prasankit`) — **อย่าลืมหลัง `make test`** (Makefile `test`/`seed` comment เตือนไว้แล้ว)
- **Goose CLI:** `make goose-up` (apply all), `make goose-down` (ถอย 1 step), `make goose-status`. หมายเหตุ Makefile default `GOOSE_DSN` ใช้ port 15432 แต่ docker bind 15433 — set env `GOOSE_DSN=postgres://prasankit:change_me@localhost:15433/prasankit?sslmode=disable` (สอดคล้องกับ integration test DSN) — minor infra inconsistency ไม่ใช่ blocker
- **Mailpit UI:** `http://localhost:18025`
- **FE dev (`pnpm dev`):** ต้องมี `app-web/.env.local` ชี้ `NEXT_PUBLIC_API_URL=http://localhost:18080` (root `.env` ครอบ docker build-arg อยู่แล้ว)

## Workflow patterns ที่ proven แล้ว session นี้ (Ultracode + D37)
- **Design phase = multi-agent workflow แบบ N architect parallel → synthesize → M critic parallel → refine** — ใช้ใน M2 data-model (9 agents) + 6a spec (4 agents). ทั้ง 2 รอบจับ issue ระดับ design ก่อน implementation
- **Implementer dispatch = D37 hybrid** (workflow 1 phase = 1 Opus implementer; main-thread Opus = architect/reviewer separate instance) — ใช้ใน 6a (1 implementer + 2 adversarial verifier). คง author≠reviewer ที่ระดับ instance
- **StructuredOutput failure mode (v1 ของ 6a spec workflow)**: prompt prescriptive ยาวมาก (CONTEXT_BLOCK เกิน 50 บรรทัด) → agent ไม่เรียก StructuredOutput tool, response เป็น prose. **Fix** = ทำ prompt สั้น (point to docs แทน prescribe inline) + เตือน "MUST call StructuredOutput" ชัดเจน + ลด architect count (1 ดีกว่า 3 ที่งงตาย). v2 ผ่านสบาย
- **trust-but-verify ตอน implementer return:** อ่าน diff เอง + `make test -count=1` (กัน cached) + smoke test e2e (curl ผ่าน live API หลัง `make api-up`) + SQL check audit invariant. ทุกข้อจับได้จริงรอบ session นี้

## ทำอะไรต่อ (เลือก — ถาม User ก่อน, ระบุข้อแนะนำ; ถามแบบ numbered list ในข้อความ ไม่ใช้ option-picker)
1. **M3 — Deliverable + ตรวจรับ — แนะนำ (milestone ถัดไป)** — wedge ครึ่งแรก: ส่งงาน (submission) → ตรวจรับหลายรอบ (pass/reject/conditional). เริ่มที่ **design data-model §M3** (`deliverables` / `submissions` / `deliverable_status` master + acceptance flow) → BE (module + migrations 000017+) → FE. pattern เดิม: design workflow → spec → dispatch → trust-but-verify + **manual smoke** (+ `make seed` หลัง test). ⚠️ §M3 ยังไม่ได้ design — ต้องทำ data-model ก่อน (เหมือน M2 Phase 1)
2. **PasswordInput → signup/reset (polish ค้าง)** — เอา `PasswordInput` (eye toggle, มีใน [ui.tsx](../app-web/src/components/ui.tsx) แล้ว) ไปใช้ที่ signup + reset-password ให้ครบ (login ใช้แล้ว)
3. **flow-doc / permission matrix / Security-rate-limit pass** — sideline ที่ defer ไว้ (ดู open threads)

## Open threads (ยังไม่ตัดสิน — อย่าลืม)
**จาก M2 §M2.8 (open threads ของ data-model — ดูเต็มใน [04 §M2.8](04-data-model.md)):**
- **Placeholder member + claim-by-email** — confirm defer ออก M2 ตามเดิม; เปิดทาง M3+ split profile+user_account
- **Project lifecycle state machine** (D44 free transition ใน M2) — M3+ พิจารณาถ้า deliverable/finance ผูก lifecycle
- **Project owner soft-delete + ownership-transfer flow** — service-level M3+
- **Project restore + slug uniqueness** (external review pass #2 จับ) — M3+ ตัดสิน policy: (a) prompt rename, (b) slug-mutate-on-delete, (c) interactive conflict-resolve
- **Multi-company-position per membership** (D41 trade-off = 1:1 nullable) — M3+ upgrade เป็น join table ถ้าจำเป็น
- **`requesting_unit` upgrade to master** — M3+ ถ้า executive rollup pain ของจริง
- **Position seed default per workspace** — M3+ บน workspace.create service
- **Project-scoped invitation / bulk add-remove / role-history table** — M3-M5+
- **RLS hardening + audit policy for nullable ws_id** — security pass

**จาก M1 §8 ที่ยังค้าง:**
- **Rate-limit/cooldown auth endpoints** (D34) — dedicated combined pass; ยังไม่มี ratelimit infra
- **requireSession +1 query** (5c) — single JOIN ถ้าเป็น hot path; defer
- **Session-revocation เมื่อ multi-identity/MFA** — M2+ rethink
- **Dead code:** `extractVerifyTokenFromDB` ใน `auth_handler_test.go` — ลบได้ทีหลัง
- **Test parallel-safety** (`-p 1` dependency) — tech-debt, ไม่ด่วน
- **`isUniqueViolation` string-match "23505"** — harden เป็น typed `pgconn.PgError` ทีหลัง

**Repo hygiene เล็ก ๆ:**
- `app-web/.env.example` ถูก gitignore (`app-web/.gitignore:34 .env*`) — template ใน app-web ไม่มีผลต่อ fresh clone. ถ้าอยากให้ helpful ต้อง un-ignore แยก
- Makefile root `GOOSE_DSN` default 15432 vs docker bind 15433 — ใช้ env override ผ่านได้ ไม่ใช่ blocker

## M2 = Functional Project (build-plan §5) — ✅ ปิดครบทั้ง milestone
**Phase 1 data-model = done** (D38-D49) · **BE 6a/6b-1/6b-2 = done** (projects/masters/members/owner-FK/positions/junction/company_position; migrations 000008-000016; D45/D46/D47) · **FE-A/B/C1/C2 = done** (FE-A D48/D49 · FE-B D51 · FE-C1 masters `/settings/positions` D53 · FE-C2 assign บน project detail + Decision-A read-add D55) · **Teal theme + post-smoke fixes (nav-leak/password/dev-seeder) = done** (D54/D55) · **User manual smoke ผ่าน**. → **M2 ปิดครบ. ก้าวต่อ = M3 (Deliverable + ตรวจรับ — wedge ครึ่งแรก); ⚠️ §M3 ยังไม่ได้ design → ทำ data-model ก่อน (เหมือน M2 Phase 1).**

## เริ่ม session หน้ายังไง
อ่านตามลำดับ: **ไฟล์นี้ → 00-workflow (D37 hybrid implementer policy) → 01-vision → 02-architecture (§4.3 4 invariants) → 03-build-plan → 04-data-model (§M2 ทั้งหมด) → DECISIONS (D1-D55)** แล้วถาม User ว่าจะไปข้อไหนใน "ทำอะไรต่อ" (แนะนำ **M3 — Deliverable + ตรวจรับ**; M2 ปิดครบทั้ง milestone แล้ว — BE 6a/6b-1/6b-2 + FE-A/B/C1/C2 + theme + fixes + smoke ผ่าน). ⚠️ **§M3 ยังไม่ได้ design** → เริ่มที่ data-model §M3 ก่อน (deliverables/submissions/acceptance) เหมือน M2 Phase 1. ⚠️ หลัง `make test` อย่าลืม `make seed` (DB โดน TRUNCATE).

อย่าลืม:
- **Workflow ทุก substantive task** (Ultracode active) — design phase = multi-agent fan-out; implementation = D37 dispatch + adversarial verify
- **trust-but-verify** ตอน implementer return — make test (`-count=1` กัน cached) + diff อ่านเอง + smoke e2e + SQL check invariants
- **Decision ทุกอันที่ตกลงต้อง fold เข้า docs/ + DECISIONS.md ใน round เดียว** (§9 — ไม่ fold = ไม่ตัดสิน)
- **dispatch implementer = D37 hybrid** (Opus คนละ instance, ไม่ใช่ main-thread เขียนเอง+รีวิวเอง — confirmation bias)
