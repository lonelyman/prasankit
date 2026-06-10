# 99 — Handoff (อ่านก่อนเพื่อนเมื่อเปิด session ใหม่)

> บอก "ตอนนี้อยู่ตรงไหน + ทำอะไรต่อ". เป็น **living doc — เขียนทับทุกครั้งที่ปิด session**.
> รายละเอียดเต็มอยู่ใน docs ที่ลิงก์. log การตัดสินใจอยู่ใน [DECISIONS.md](DECISIONS.md).

## อัปเดตล่าสุด
**2026-06-10** — **M3 Phase 2 (BE implementation) เสร็จ — review + test + smoke ผ่านครบ. ก้าวต่อ = M3 Phase 3 (FE).**

Session นี้ (User pre-approved ทุก gate):
1. **M3 BE เต็ม loop** — spec §5 → dispatch Opus implementer แยก instance (D37) → Kael review ตาม §6 + trust-but-verify (`go test -p 1 -count=1 ./...` ผ่านทั้ง suite) → manual smoke e2e ผ่าน (multi-round: submit→reject→resubmit→conditional→resubmit→accept→409 terminal + superseded/already_reviewed/non-member 404 + body submitted_by ถูก ignore)
2. **ของใหม่:** migrations `000018–000022` (masters + deliverables + submissions + submission_reviews) · module `internal/modules/deliverable` (+adapter/+handler) · 7 routes ใต้ `/projects/:projectID/deliverables` (middleware แค่ session+tenant — authz ทั้งหมด service-layer D63) · config flag `DELIVERABLE_FORBID_SELF_REVIEW` (default false, OQ-6) · race tests (concurrent submit + review-vs-resubmit interleave) ผ่านจริงบน Postgres
3. **D65 (refine D63):** org Owner/Admin bypass ไม่ครอบ submit/review — ไม่มี active `project_members` row = 422 `deliverable.not_project_member` (FK D60 ต้องการ member จริง); bypass เหลือเฉพาะ deliverable CRUD + read

> **Smoke (dev) login:** `owner@prasankit.local` (owner) + `bob@prasankit.local` (user) บน ws `prasankit` · รหัส `demopass123`. **หลัง `make test` ต้อง `cd app-api && make seed`** (test TRUNCATE dev DB). dev DB ตอนนี้มี project "M3 Smoke" (slug `m3-smoke`) + deliverable 1 งวด (accepted, 3 รอบ) ทิ้งไว้ให้ดูใน UI ได้

## สถานะตอนนี้
- ✅ docs ฐานครบ: [00-workflow](00-workflow.md) · [01-vision](01-vision.md) · [02-architecture](02-architecture.md) · [03-build-plan](03-build-plan.md) · [04-data-model](04-data-model.md) (§M1+§M2+§M3) · [DECISIONS](DECISIONS.md) (**D1–D65**) · [README](../README.md)
- ✅ M1 + M2 ปิดครบทั้ง milestone
- ✅ **M3 Phase 1 (design) + Phase 2 (BE) เสร็จ** — schema 000018–000022 ใน dev DB แล้ว (goose v22), API container rebuild แล้ว
- ⏭️ **M3 Phase 3 = FE (deliverable UI) — ขั้นต่อไป**

## M3 BE snapshot (อ่านก่อนทำ FE)
**Endpoints (ทั้งหมดต้อง session cookie + `X-Workspace-Slug`):**
- `POST/GET /api/v1/workspaces/projects/:projectID/deliverables` — create (owner/manager/org O-A) / list (member ใดก็ได้)
- `GET/PUT/DELETE .../deliverables/:deliverableID` — detail (+`submissions[]` round DESC) / update / soft-delete
- `POST .../deliverables/:deliverableID/submissions` — ส่งรอบใหม่ (member ที่ไม่ใช่ viewer; **ต้องเป็น member จริง** D65) body: `note?`, `url?`
- `POST .../submissions/:submissionID/review` — ตรวจรับ (owner/manager + member จริง) body: `decision_code` (accepted/conditional/rejected), `comment?`

**Response สำคัญ:** `deliverable_status_code` (derive D59: not_submitted/in_review/accepted/conditional/rejected) · `latest_submission{round_no, timeliness_code(no_due/early/on_time/late), late_by_days, review{decision_code,...}}` · FE map label เอง (D27)

**Error codes ที่ FE ต้อง handle:** 404 `project.not_found`/`deliverable.not_found`/`submission.not_found` (รวม non-member probe-collapse D42) · 403 `deliverable.forbidden` · 409 `deliverable.accept_terminal` (ส่งซ้ำหลัง accept) / `submission.superseded` / `submission.already_reviewed` · 422 `deliverable.not_project_member` (D65) / `deliverable.invalid_master_code` · 400 `validation.invalid_input`

**กฎที่ implement แล้ว:** accept=terminal (D58) · status+timeliness = derive ไม่มี column (D59/D62, UTC date) · append-only (D61) · FOR UPDATE serialization (D64) · submitted_by/reviewed_by server-derive (D60)

## วิธีรัน / เทสต์
- `make dev-up` (infra incl. mailpit → migrate → api → web → health). `make down` ปิด
- **host ports:** API `18080`, Web `13000`, pg `15433`*, redis `16379`*, minio `19000/19001`, Mailpit SMTP `11025` / UI `18025` (*local .env — pg จริงคือ **15433** ไม่ใช่ 15432)
- **เทสต์:** `cd app-api && make test` (= `go test -p 1 ./...`) ห้าม `go test ./...` เปล่า ๆ
- **⚠️ `make test` ล้าง dev DB** → `cd app-api && make seed` กู้คืน
- **FE dev (`pnpm dev`):** ต้องมี `app-web/.env.local` ชี้ `NEXT_PUBLIC_API_URL=http://localhost:18080`; dev :3000 โดน CORS block (อนุญาตแค่ :13000)

## ทำอะไรต่อ (เลือก — ถาม User ก่อน; ถามแบบ numbered list)
1. **M3 Phase 3 — FE deliverable UI — แนะนำ (ปิด M3 ให้ครบ)** — spec FE → dispatch implementer แยก instance: หน้า deliverables ใน project (list + status badge + timeliness), submit form (note/url), review form (decision+comment), submissions history timeline. ใช้ M3 BE snapshot ข้างบนเป็น contract
2. **เก็บ doc-drift ที่ค้าง** — ธีม B (API reference `docs/05-api-reference.md` — ตอนนี้ 46 endpoint แล้ว) · ธีม E (architecture drift: per-module middleware, audit layer, goose ในตาราง stack)

## Open threads (ยังไม่ตัดสิน)
**M3 §M3.9 (defer):** file upload (M4+) · assignee/reviewer-assignment (M4) · task↔deliverable (M4) · edit/retract (M4+) · timeliness master/KPI index (M5) · lifecycle coupling (D44) · `workspaces.timezone` (OQ-3, M5) · `requireProjectPermission` middleware (M4/M5, D63)
**จาก M2 ที่ยังค้าง:** rate-limit auth (D34) · placeholder member (M3+) · lifecycle state machine (D44) · multi-company-position (D41)
**ใหม่ (สังเกตจาก M3 BE review, ยังไม่ตัดสิน):** service validate ความยาวด้วย Go `len()` (bytes) แต่ DB CHECK ใช้ `length()` (chars) — ข้อความไทยโดน limit เร็วกว่าที่ DB ยอม ~3 เท่า. เป็น pattern เดิมตั้งแต่ M2 (project module) ไม่ใช่ regression — ถ้าจะแก้ควรแก้ทั้งระบบ (รอบ dedicated)
**Docs (ธีม B/E):** API reference · permission matrix · architecture drift

## เริ่ม session หน้ายังไง
อ่านตามลำดับ: **ไฟล์นี้ → [00-workflow](00-workflow.md) → §M3 BE snapshot ข้างบน → [04-data-model §M3](04-data-model.md) ถ้าต้องลึก**

แล้วถาม User ว่าจะไปข้อไหนใน "ทำอะไรต่อ" (แนะนำ **M3 Phase 3 — FE**)

อย่าลืม:
- **1 dispatch = 1 stack** (D56/D37) — FE dispatch แยก; main-thread Kael = architect/reviewer
- **trust-but-verify** ตอน implementer return — test -count=1 + diff + smoke e2e
- **Decision ทุกอันต้อง fold เข้า docs/ + DECISIONS.md ใน round เดียว** (§9)
- **หลัง `make test` ต้อง `make seed`**
- **migration ถัดไปเริ่มที่ 000023**
