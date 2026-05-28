# DECISIONS — Prasankit

> **Append-only log กันลืม.** หนึ่งบรรทัดต่อ decision ที่ "ตกลงแล้ว": วันที่ + สรุปสั้น + ลิงก์ doc ที่ลงรายละเอียด.
> **กฎ:** เพิ่มล่างสุดเสมอ · ห้ามแก้/ลบบรรทัดเก่า · กลับ decision = เพิ่มบรรทัดใหม่ที่ระบุว่า supersede อันไหน.
> log นี้ไว้ "ไล่ timeline + กันตกหล่น" ไม่ใช่ที่เก็บเนื้อหา — เนื้อหาเต็มอยู่ใน doc ที่ลิงก์.

| # | วันที่ | Decision | รายละเอียด |
| --- | --- | --- | --- |
| D1 | 2026-05-27 | v2 = โปรดักต์เดิมของ v1 (multi-tenant PM platform) แต่ rebuild ใหม่ใต้ vibe-code loop. v1 เป็น reference ไม่ใช่ข้อผูกมัด | [01-vision §1–§2](01-vision.md) |
| D2 | 2026-05-27 | scope demo แรก **ยังไม่ล็อก** — ยกไป build-plan doc. vision เก็บแค่ north star | [01-vision §10](01-vision.md) |
| D3 | 2026-05-27 | User↔Tenant = **many membership** (1 identity อยู่ได้หลาย workspace, สร้างเอง/ถูกเชิญ). resolve tenant per session ฝั่ง backend. **ไม่ทำ cross-tenant switching UI** รอบนี้ | [01-vision §4.1, §9](01-vision.md) |
| D4 | 2026-05-27 | External client เข้าร่วม Client Project ผ่าน **invite เข้า workspace** (Org User + Project Viewer/Member) ไม่สร้าง client portal แยก | [01-vision §6.3](01-vision.md) |
| D5 | 2026-05-27 | Task status รายโปรเจคต้อง **map กับ 3 universal category** (To Do/In Progress/Done) เพื่อ rollup ข้ามโปรเจค | [01-vision §7](01-vision.md) |
| D6 | 2026-05-27 | Positioning: **ไม่ใช่ Trello/Jira/Notion** — wedge = งวดงาน+ตรวจรับ+การเงินโครงการ+lifecycle+audit. task/เอกสารทำให้ "พอใช้ดี" ไม่ "ลึกสุด" | [01-vision §1.1](01-vision.md) |
| D7 | 2026-05-27 | Task = **board+list view** (calendar/table future), การ์ดมี comment+checklist+attachment, **ไม่ทำ epic/sub-task hierarchy** รอบนี้ | [01-vision §7](01-vision.md) |
| D8 | 2026-05-27 | Document = **File Center (เก็บไฟล์)** ไม่ใช่ editor/wiki แบบ Notion | [01-vision §9](01-vision.md) |
| D9 | 2026-05-27 | ระบบกันลืมข้ามรอบ = docs (ตัวจริง) + DECISIONS.md (log นี้) + Kael memory (ป้ายบอกทาง). decision ที่ไม่ลง = ถือว่ายังไม่ตัดสิน | [00-workflow §9](00-workflow.md) |
| D10 | 2026-05-28 | เพิ่มระบบ handoff: `docs/99-handoff.md` (จุดเริ่ม session หน้า) + กฎ §9.1 — พอ User สั่งปิด session, Kael เช็ค commit/decision → อัปเดต handoff → ให้ prompt ส่งต่อ | [00-workflow §9.1](00-workflow.md), [99-handoff.md](99-handoff.md) |
| D11 | 2026-05-28 | v2 **inherit tech stack ของ v1** ทั้งชุด (Go+Fiber / Postgres18 / Redis8 / MinIO / bcrypt / UUIDv7 / hexagonal). rebuild = process/loop ไม่ใช่ tech. GORM คงไว้ก่อน (data-model doc อาจทบทวน hot path) | [02-architecture §2](02-architecture.md) |
| D12 | 2026-05-28 | Tenant isolation = **shared DB, shared schema, `tenant_id` column** ทุกตาราง (ของ v1). บังคับด้วย isolation invariant (repo รับ tenant_id เสมอ). Postgres RLS = future hardening ไม่ทำรอบนี้ | [02-architecture §4](02-architecture.md) |
| D13 | 2026-05-28 | Auth = **opaque session token** (สุ่ม 32B, เก็บ sha256 hash ใน Redis, cookie httpOnly, revocable — ไม่ใช่ JWT). Tenant resolution = **`X-Workspace-Slug` header** → backend เช็ค membership → TenantContext. ไม่เชื่อ tenant_id/role จาก client | [02-architecture §5](02-architecture.md) |
| D14 | 2026-05-28 | 02-architecture = **backend-first** (เหมือน v1 API-only). Frontend stack + วิธีคุย API **ยกไป build-plan** ตั้ง backend ให้แน่นก่อน | [02-architecture §1](02-architecture.md) |
| D15 | 2026-05-28 | **ไม่เก็บ enum ทุกกรณี** — controlled vocabulary ทุกตัว = master table + FK (ห้าม Postgres ENUM / `CHECK(IN)`). ทุก master table มี `code`(stable)+`is_system`+`sort_order`+i18n label+`status`; โค้ดอ้าง system row ผ่าน `code`. ต่างจาก v1 ที่ทำผสม | [02-architecture §7](02-architecture.md) |
| D16 | 2026-05-28 | **Authz (membership/org role/project role) resolve per-request ไม่ cache ใน session** — กัน stale authz, คง revoke-ทันที (เหตุผลเดียวกับเลือก session แทน JWT), เข้ากับ per-request multi-workspace. (cross-model review: reviewer เสนอ cache ใน SessionRecord, Kael ค้าน, User เลือก per-request) | [02-architecture §5.5](02-architecture.md) |
| D17 | 2026-05-28 | **ล็อก scope first cut (resolve D2)** — goal = proof-of-loop ภายใน. In: Foundation+Project+Team+Deliverable+ตรวจรับ+Finance(thin)+**Task board**. Out รอบนี้: Document/Announcement/Activity-Log UI. (Task board User override ใส่กลับจาก wedge-thin-slice เดิม) | [03-build-plan §2](03-build-plan.md) |
| D18 | 2026-05-28 | **FE stack = Next.js + TypeScript** (resolve D14) — responsive web, consume API ผ่าน session cookie + X-Workspace-Slug | [03-build-plan §3](03-build-plan.md) |
| D19 | 2026-05-28 | **Build approach** — 1 milestone = 1 vertical slice; **1 Sonnet dispatch = 1 stack** (BE นำ FE ครึ่งก้าว); M0 = dual-stack walking skeleton; data-model just-in-time ต่อ milestone. (operational จาก cross-model review) | [03-build-plan §4](03-build-plan.md) |
| D20 | 2026-05-28 | **Signup ไม่ auto-create workspace** — empty state → สร้าง/รับเชิญ (รองรับ multi-workspace D3). flow เต็มยกไป flow-doc. (reviewer เสนอ auto-create, User เลือก empty-state) | [03-build-plan §7](03-build-plan.md) |
| D21 | 2026-05-28 | **Pin Go 1.26** (refine D11) — greenfield ไม่ลอก minor ของ v1 (1.25) ดิบๆ, bump เป็น minor ล่าสุด ความเสี่ยงต่ำ ได้ compiler/stdlib/security ใหม่. go.mod `go 1.26.0`, Dockerfile `1.26-alpine`, `GOTOOLCHAIN=auto`. (M0 review: User เริ่ม proposal → bump ก่อน dispatch) | [02-architecture §2](02-architecture.md) |
| D22 | 2026-05-28 | **FE baseline (refine D18)** — App Router + Tailwind + **pnpm**; เรียก API แบบ client-side `fetch` (`credentials:'include'`) เพื่อพิสูจน์ CORS+cookie จริง; test lane = Vitest; อยู่ `app-web/`. (FE M0 spec: User เลือก pnpm over npm) | [03-build-plan §3](03-build-plan.md) |
