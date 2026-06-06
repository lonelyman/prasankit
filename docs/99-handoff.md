# 99 — Handoff (อ่านก่อนเพื่อนเมื่อเปิด session ใหม่)

> บอก "ตอนนี้อยู่ตรงไหน + ทำอะไรต่อ". เป็น **living doc — เขียนทับทุกครั้งที่ปิด session**.
> รายละเอียดเต็มอยู่ใน docs ที่ลิงก์. log การตัดสินใจอยู่ใน [DECISIONS.md](DECISIONS.md).

## อัปเดตล่าสุด
**2026-06-06** — **M3 Phase 1 (design data-model) ปิด — committed + pushed `origin/dev` (`ec8a392`). ก้าวต่อ = M3 Phase 2 (BE implementation).**

Session นี้ทำ 2 งาน:
1. **Docs audit + แก้ doc-drift (`72538a7`)** — sweep "Sonnet"→"implementer" ใน workflow §3/§4/§5/§6 (D56), fold migration 000017 slug-regex (D57), สร้าง root README, เพิ่ม mailpit เข้า `infra-up`
2. **M3 Phase 1 design (`ec8a392`)** — multi-agent design (3 architect → synth → 3 adversarial critic → finalize) + User เคาะ 7 OQ defaults (OQ-7: **accept=terminal**). fold §M3 ลง [04-data-model.md](04-data-model.md) + D58–D64 ลง [DECISIONS.md](DECISIONS.md) + amend [02-architecture.md](02-architecture.md) §5.5 (D63)

> **บทเรียน workflow:** design/explore phase ใช้ agent schemaless (markdown) ไม่ใช่ nested StructuredOutput schema — nested schema ทำให้ agent ไม่เรียก StructuredOutput → workflow fail. บันทึกลง memory แล้ว (`workflow-schemaless-design.md`)

> **Smoke (dev) login:** `owner@prasankit.local` (owner) + `bob@prasankit.local` (user) บน ws `prasankit` · รหัส `demopass123`. **หลัง `make test` ต้อง `cd app-api && make seed`** (test TRUNCATE dev DB).

## สถานะตอนนี้
- ✅ docs ฐานครบ: [00-workflow](00-workflow.md) · [01-vision](01-vision.md) · [02-architecture](02-architecture.md) · [03-build-plan](03-build-plan.md) · [04-data-model](04-data-model.md) (§M1 + §M2 + **§M3 design ครบ**) · [DECISIONS](DECISIONS.md) (**D1–D64**) · [README](../README.md) (root entry point)
- ✅ M1 + M2 ปิดครบทั้ง milestone (BE + FE + theme + smoke ผ่าน)
- ✅ **M3 Phase 1 = design data-model done** (§M3: deliverables/submissions/submission_reviews + masters + acceptance + race/authz/migration plan)
- ⏭️ **M3 Phase 2 = BE implementation — ขั้นต่อไป**

## M3 Design snapshot (อ่านก่อน implement)

**3 ตาราง:** `deliverables` (งวด, soft-delete) · `submissions` (ส่งหลายรอบ, append-only, note/url text-only D8) · `submission_reviews` (ตรวจรับ 1/submission, append-only, verdict immutable)

**2 master:** `submission_decisions` (accepted/conditional/rejected) · `deliverable_statuses` (label dict ของ derived status — **ไม่มี FK ชี้มา**, D59)

**กฎสำคัญ (User เคาะแล้ว):**
- **accept = terminal** (D58/OQ-7) — ส่งซ้ำหลัง accept = 409; conditional/rejected = ส่งซ้ำได้
- **สถานะงวด = derived** จาก (รอบล่าสุด + review) ไม่มี column (D59/§2.6)
- **FOR UPDATE lock บน deliverable** ทั้ง submit + review (D64 — กัน race review-vs-resubmit)
- **submitted_by / reviewed_by = server-derive** จาก actor (§5.4, ไม่รับจาก body)
- **project-role gate = service-layer M3** (D63 — M3 = milestone แรกที่มี project-role enforcement จริง)
- **migration 000018–000022** FK-ordered masters→deliverables→submissions→reviews
- **ไม่มี file upload** (D8 — แนบแค่ note/url text)

## วิธีรัน / เทสต์
- `make dev-up` (infra incl. mailpit → migrate → api → web → health). `make down` ปิด
- **host ports:** API `18080`, Web `13000`, pg `15432`*, redis `16379`*, minio `19000/19001`, Mailpit SMTP `11025` / UI `18025` (*local .env อาจต่างจาก example)
- **เทสต์:** `cd app-api && make test` (= `go test -p 1 ./...`) ห้าม `go test ./...` เปล่า ๆ (integration ใช้ DB+Redis+Mailpit ร่วม flake)
- **⚠️ `make test` ล้าง dev DB** → `cd app-api && make seed` กู้คืน (owner@/bob@/prasankit/demopass123)
- **FE dev (`pnpm dev`):** ต้องมี `app-web/.env.local` ชี้ `NEXT_PUBLIC_API_URL=http://localhost:18080`; dev :3000 โดน CORS block (อนุญาตแค่ :13000)
- **Mailpit UI:** `http://localhost:18025`

## ทำอะไรต่อ (เลือก — ถาม User ก่อน; ถามแบบ numbered list)
1. **M3 Phase 2 — BE implementation — แนะนำ (ก้าวถัดไปตาม build-plan)** — เขียน spec §M3 BE → dispatch Opus implementer แยก instance (D37) → migrations 000018–000022 + module deliverable/submission/review (domain/service/repo/handler) + test (รวม race test D64) → Kael review + trust-but-verify + manual smoke. ⚠️ service-layer project-role gate (D63) = ใหม่ในรอบนี้ — อย่าข้าม
2. **ก่อน M3 BE: เก็บ doc-drift ที่เหลือ** — ธีม B (API/permission reference `docs/05-permissions.md`) · ธีม E (architecture doc drift: per-module middleware 6 ตัว, audit layer, goose ในตาราง stack) — ไม่บล็อก M3 แต่ระบบเริ่มโต

## Open threads (ยังไม่ตัดสิน)
**M3 §M3.9 (open threads ที่ defer ไว้):**
- file/attachment upload บน submission (D8, vision §9) — M4+
- deliverable assignee/owner + reviewer-assignment column — M4
- task↔deliverable link — M4
- submission/review edit+retract — append-only D61, M4+
- timeliness master/snapshot + decision-rollup/KPI index — M5 dashboard
- lifecycle coupling project↔deliverable — D44 free, service-layer เมื่อพิสูจน์ misuse
- `workspaces.timezone` — OQ-3, revisit M5
- `requireProjectPermission` middleware — D63, M4/M5

**จาก M2 §M2.8 ที่ยังค้าง:**
- Rate-limit/cooldown auth endpoints (D34) — dedicated pass, ยังไม่มี infra
- Placeholder member + claim-by-email — M3+
- Project lifecycle state machine (D44) — M3+ พิจารณาหลัง deliverable ผูก lifecycle
- Multi-company-position (D41) — M3+ ถ้า requirement จริง

**Docs (ธีม B/E ที่ defer):**
- API/endpoint reference `docs/05-api-reference.md` (39 endpoint, ไม่มีที่รวม)
- Permission matrix (org role × project role × action) — doc สัญญาไว้ 3 ที่
- Architecture doc drift: per-module middleware 6 ตัว, audit write layer, goose ในตาราง stack

## เริ่ม session หน้ายังไง
อ่านตามลำดับ: **ไฟล์นี้ → [00-workflow](00-workflow.md) → [04-data-model §M3](04-data-model.md) → [DECISIONS D58-D64](DECISIONS.md)**

แล้วถาม User ว่าจะไปข้อไหนใน "ทำอะไรต่อ" (แนะนำ **M3 Phase 2 — BE**). ⚠️ **§M3 design ตัดสินแล้วทั้งหมด** — ไม่ต้อง design เพิ่ม เริ่ม spec + implement ได้เลย

อย่าลืม:
- **1 dispatch = 1 stack** (D56/D37) — BE dispatch แยก, FE dispatch แยก; main-thread Kael = architect/reviewer
- **trust-but-verify** ตอน implementer return — `make test -count=1` + diff + smoke e2e
- **Decision ทุกอันที่ตกลงต้อง fold เข้า docs/ + DECISIONS.md ใน round เดียว** (§9)
- **หลัง `make test` ต้อง `make seed`** (DB TRUNCATE)
- **M3 Phase 2 migration เริ่มที่ 000018** (000017 ปิดแล้ว)
