# 99 — Handoff (อ่านก่อนเพื่อนเมื่อเปิด session ใหม่)

> บอก "ตอนนี้อยู่ตรงไหน + ทำอะไรต่อ". เป็น **living doc — เขียนทับทุกครั้งที่ปิด session**.
> รายละเอียดเต็มอยู่ใน docs ที่ลิงก์. log การตัดสินใจอยู่ใน [DECISIONS.md](DECISIONS.md).

## อัปเดตล่าสุด
**2026-05-28** — ปิด session: **M0 เสร็จสมบูรณ์ (dual-stack) + boot จริงผ่าน end-to-end**. ลง 6 commits, อุด `/spec`, จัดระเบียบ `.env`. จุดต่อไป = M1 (data-model doc ก่อน)

## สถานะตอนนี้
- ✅ docs ฐานครบ: [00-workflow](00-workflow.md) · [01-vision](01-vision.md) · [02-architecture](02-architecture.md) · [03-build-plan](03-build-plan.md) · [DECISIONS](DECISIONS.md) (D1–D23)
- ✅ **M0 walking skeleton (dual-stack) — committed + boot verified**
  - **BE** (`app-api/`): Go 1.26 hexagonal + docker-compose (pg18/redis8/minio) + goose + `GET /health|/health/live|/health/ready` ผ่าน middleware chain (request-id→CORS→routes) + presenter envelope + central error-handler + 7 tests
  - **FE** (`app-web/`): Next.js (App Router) + TS + Tailwind + pnpm; หน้า health เรียก BE cross-origin (`credentials:'include'`) โชว์ status+checks + Vitest 3 tests
  - **boot จริง:** `make dev-up` → 5 container healthy; ยืนยัน API ready ok, web หน้า render, **CORS pipe** (Allow-Origin+Credentials) ทำงาน
- ✅ `/spec` skill tuned (acceptance: `gofmt -l` + บังคับ `docker build` จริงสำหรับ dispatch ที่แตะ Docker)

## วิธีรัน M0 (สำคัญ — กันงงรอบหน้า)
- `make dev-up` (ขึ้น infra→migrate→BE→FE→health). `make down` ปิด
- **host ports = high ports (D23):** API `http://localhost:18080`, Web `http://localhost:13000` (container ยัง 8080/3000)
- **local `.env` ต่างจาก `.env.example`:** pg/redis host port remap `15433`/`16380` (ของจริง example=15432/16379) เพราะชนเครื่อง dev (eap-dev-postgres จอง 15432). `.env` จัดเป็น Active(M0)/Future(คอมเมนต์ M1,M4)/ลบ(path-tenant) + scrub SMTP creds แล้ว
- **อย่า re-introduce 3 gotcha ที่แก้ไปแล้ว:** ต้องมี `app-web/.dockerignore`, `ENV HOSTNAME=0.0.0.0` ใน FE Dockerfile, healthcheck ใช้ `127.0.0.1` (ไม่ใช่ localhost→IPv6)

## ทำอะไรต่อ (เลือก 1 — ถาม User ก่อน, ระบุข้อแนะนำด้วย)
1. **M1: data-model doc ก่อน (แนะนำ)** — schema M1 (user_accounts, workspaces, workspace_memberships, org-role master), **resolve `tenant_id` vs `workspace_id`** (02 §4.1), apply master-table convention (02 §7). ทำก่อน dispatch M1 BE (D19 just-in-time)
2. **M1: ลงมือเลย** — ถ้าตัดสินใจไม่ทำ data-model doc แยก, brief schema ใน spec ตรง ๆ (เสี่ยง ambiguity มากกว่า)
3. **flow-doc** (option) — user flow เต็ม (สมัคร→สร้าง/ถูกเชิญ→สลับ workspace) — ใช้ตอน M1
4. **permission matrix** (option) — org × project role × action — ใช้ตอน M1/M2

## Open threads (ยังไม่ตัดสิน — อย่าลืม)
- `tenant_id` vs `workspace_id` collapse หรือคงสองคอลัมน์ — **ต้อง resolve ใน data-model doc M1** (02 §4.1)
- Postgres RLS = hardening ชั้นสอง — candidate (02 §4.4)
- M5 finance trim-candidate ถ้าจวนตัว (03 §2)
- (option เดิม) เพิ่มกฎ "1 session = 1 milestone" ใน 00-workflow — User ยังไม่ตัดสิน
- stack docker ยังรันค้างไว้ตอนปิด session (เปิด localhost:13000 ได้) — `make down` เมื่อไม่ใช้

## M1 = Identity & Tenancy (build-plan §5)
ส่ง: signup/login/logout (bcrypt, session Redis), `requireSession`; สร้าง workspace (1:1 tenant), membership, `X-Workspace-Slug` resolver + `requireTenantContext` + `requireWorkspacePermission` + isolation invariant *ของจริง*. FE: login/signup + empty-state→create/accept-invite (D20). **cross-cutting เริ่มทอ M1:** audit log + i18n code + master-table.

## เริ่ม session หน้ายังไง
อ่านตามลำดับ: **ไฟล์นี้ → 00-workflow → 01-vision → 02-architecture → 03-build-plan → DECISIONS** แล้วถาม User ว่าจะไปข้อไหนใน "ทำอะไรต่อ" (เริ่มที่ M1 data-model doc — ผ่าน `/spec` ตอน dispatch).
