# 03 — Build Plan (First Cut)

> **Status: Canonical.** เอกสารนี้ล็อก **scope ของ demo แรก + ลำดับ build** ที่ vision defer ไว้ ([01-vision §10](01-vision.md), **D2**).
> derive จาก [01-vision.md](01-vision.md) (north star) + [02-architecture.md](02-architecture.md) (backend). ถ้าขัด vision → แก้ vision ก่อน (gate [00-workflow.md](00-workflow.md) §8).
> การเปลี่ยน scope/ลำดับในนี้ทำตาม gate §8 เดียวกัน — ห้าม scope creep เงียบๆ (workflow §4.4).

## 1. เป้าหมาย first cut: Proof-of-Loop

demo แรกนี้ **ภายใน** — พิสูจน์ 2 อย่างพร้อมกัน:
1. **vibe-code loop ผลิต software ที่ทำงานจริงได้** (spec → dispatch Sonnet → review → commit) — โครงสร้าง role/gate ใน workflow ใช้ได้จริงบนโค้ด ไม่ใช่แค่บนเอกสาร
2. **wedge ของ Prasankit ทำงาน end-to-end** — งวดงาน+ตรวจรับ + การเงินโครงการ (D6) — พิสูจน์ว่าไม่ใช่ Jira/Notion อีกตัว

ไม่มี deadline ภายนอก → optimize เพื่อ **ความถูกต้องของ loop + โชว์ wedge** ไม่ใช่ความเร็วเอา demo

## 2. Scope (ล็อก D2 → D17)

### อยู่ใน first cut
| Module (vision §5) | ระดับ |
| --- | --- |
| Foundation: Auth, Workspace/Tenant, Membership | เต็มพอใช้งาน |
| 1. Project Profile | CRUD + lifecycle status + type |
| 2. Project Team / Role | members + project role + positions |
| 4. Deliverable / งวดงาน | **wedge** — ส่งหลายรอบ + ตรวจรับ |
| 3. Project Task (Jira-lite) | board+list (D7) — **เพิ่มกลับตาม User** (เดิม wedge-thin-slice ตัดออก) |
| 5. Project Finance | **wedge** — **thin-slice**: budget+expense+gate (penalty/adjustment+visibility ขั้นสูง = trim ได้ถ้าจวน) |

### ตัดออกจาก first cut (ไม่ใช่ non-goal ถาวร — แค่ไม่ใช่รอบนี้)
6. Document (File Center) · 8. Announcement · 7. Activity-Log **UI** (เก็บ log หลังบ้านตั้งแต่แรกตาม §8 แต่ยังไม่ทำหน้า)

### Non-goal ถาวร
ตาม [01-vision §9](01-vision.md) — ไม่แตะทั้งสิ้น

> **หมายเหตุ scope:** Task board ถูกเพิ่มกลับโดย User (override คำแนะนำ Kael ที่ให้คงตัด). first cut จึง = wedge + task board, ใหญ่กว่า pure wedge-thin-slice เดิม — บันทึกไว้ใน D17 เพื่อ traceability.

## 3. Frontend Stack (ล็อก D14 → D18)

**Next.js + TypeScript** (React/SSR). responsive web (vision §9 ใช้ web ก่อน mobile). consume backend API ผ่าน envelope §8 ของ 02-architecture; auth ผ่าน session cookie (httpOnly) + ส่ง `X-Workspace-Slug` ต่อ request (02 §5).

## 4. Build Approach (D19)

- **1 milestone = 1 vertical slice** ที่ demo ได้ (BE+FE) — proof-of-loop ต้องเห็นของจริงทำงานทุกรอบ
- **1 Sonnet dispatch = 1 stack** — แยก task BE (Go) กับ FE (Next.js) คนละ dispatch กัน Sonnet สลับ context งงสเปก + review เร็วขึ้น (workflow §6)
- **BE นำ FE ครึ่งก้าวเสมอ** — FE ต่อ API ที่ contract นิ่งแล้ว ไม่ไล่ตาม API ที่ขยับ
- ทุก milestone จบด้วย **review (Opus) → User decide → commit** ตาม loop (workflow §3)
- **data-model doc เขียน just-in-time ต่อ milestone** (M0/M1 ก่อน) — ไม่ model ทั้ง 8 module ล่วงหน้า (กัน over-build, vision §2)

## 5. Milestones

| M | ส่ง (vertical slice) | Entities / master table หลัก | พิสูจน์ |
| --- | --- | --- | --- |
| **M0** | **Walking skeleton (dual-stack)** — scaffold Go hexagonal + Next.js, docker-compose (pg/redis/minio), goose migration, `GET /health` ผ่านทุก layer + เรียกจาก FE ได้, presenter envelope + request-id + error-handler, test แรก | (extensions migration) | **loop machinery + stack boot + ท่อ BE↔FE** |
| **M1** | **Identity & Tenancy** — signup/login/logout (bcrypt, session Redis), `requireSession`; สร้าง workspace (1:1 tenant), membership, `X-Workspace-Slug` resolver + `requireTenantContext` + `requireWorkspacePermission` + isolation invariant *ของจริง*. FE: login/signup + empty-state → create/accept-invite workspace (D20) | user_accounts, workspaces, workspace_memberships, **org role (master)** | **multi-tenant แกนหลัก** |
| **M2** | **Functional Project** — project CRUD (tenant-scoped) + lifecycle + type; team/role + positions. FE: project list/detail/create + team mgmt | projects, **project status/type (master)**, project_members, **project role (master)**, positions | core entity ที่ใช้งานได้จริง |
| **M3** | **Deliverable + ตรวจรับ (wedge ½)** — งวดงาน, กำหนดส่ง, ส่งจริงหลายรอบ, ตรวจรับ accept/reject/conditional. FE: deliverable list + submit/review | deliverables, submissions, **deliverable status (master)** | **wedge ครึ่งแรก** |
| **M4** | **Task board (Jira-lite)** — board+list view (D7), การ์ดมี comment/checklist/attachment, task ผูก deliverable ได้ (§7). FE: board ลากย้ายสถานะ + list | tasks, **task status รายโปรเจค (master, tenant-scoped)** map 3 universal category (D5), comments, checklists, attachments | task ที่ map rollup ได้ |
| **M5** | **Finance thin-slice (wedge ½)** — budget, expense + visibility gate by project role (Viewer default ไม่เห็น, vision §6.2). FE: finance panel (gated) | budgets, expenses | **wedge ครึ่งหลัง + finance visibility** |

**Done criteria ต่อ milestone:** BE test ผ่าน (รวม case "tenant อื่นมองไม่เห็น" §4.3) · FE flow ใช้ได้จริง · ผ่าน review checklist (workflow §6) · committed

**ลำดับ M3→M4→M5 ปรับได้:** task ผูก deliverable จึงวาง M4 หลัง M3. Finance (M5) ไว้ท้ายเพราะเป็น trim-candidate ถ้าจวนตัว

## 6. Cross-cutting (ทอตั้งแต่ M1)

- **Audit log** — service เขียน log เชิงโครงสร้าง (02 §6) ทุก mutating action ตั้งแต่ M1 — แม้ยังไม่มีหน้า UI
- **i18n** — label/status ผ่าน `code` (02 §6, §7) ตั้งแต่แรก ไทย+อังกฤษ
- **Master-table convention (D15)** — ทุก status/type/role เป็น master table ไม่ใช่ enum

## 7. Onboarding flow note (D20)

signup **ไม่ auto-create workspace** — ผู้ใช้ใหม่เจอ empty state แล้วเลือก "สร้าง workspace" หรือ "รับคำเชิญ". รองรับ "อยู่หลาย workspace" ตาม D3 ได้สะอาด.
**flow เต็ม** (สมัคร → สร้าง/ถูกเชิญ → สลับ workspace) = **flow-doc** (เอกสารถัดไป ถ้า User เลือกทำ) — build-plan นี้ล็อกแค่จุด M1

## 8. ความสัมพันธ์กับเอกสารอื่น

- [01-vision.md](01-vision.md) — north star; first cut เป็น subset ของ §5 (ไม่ขัด §10)
- [02-architecture.md](02-architecture.md) — stack/layers/tenant/conventions ที่ทุก milestone ยึด
- **data-model doc (ถัดไป)** — schema per milestone (เริ่ม M0/M1), apply master-table convention (02 §7)
- **flow-doc (option)** — user flow เต็ม (§7)
- **permission matrix (ถัดไป)** — org × project role × action (vision §6) — ใช้ตอน M1/M2
