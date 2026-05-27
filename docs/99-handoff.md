# 99 — Handoff (อ่านก่อนเพื่อนเมื่อเปิด session ใหม่)

> บอก "ตอนนี้อยู่ตรงไหน + ทำอะไรต่อ". เป็น **living doc — เขียนทับทุกครั้งที่ปิด session**.
> รายละเอียดเต็มอยู่ใน docs ที่ลิงก์. log การตัดสินใจอยู่ใน [DECISIONS.md](DECISIONS.md).

## อัปเดตล่าสุด
**2026-05-28** — ปิด session: เขียน 02-architecture + 03-build-plan เสร็จ (ผ่าน cross-model review 2 รอบ), ล็อก scope first cut + FE stack

## สถานะตอนนี้
- ✅ [00-workflow.md](00-workflow.md) — loop contract — committed+pushed
- ✅ [01-vision.md](01-vision.md) — north star — committed+pushed
- ✅ [02-architecture.md](02-architecture.md) — backend: stack(inherit v1) / hexagonal / tenant isolation / auth / no-enum convention — committed+pushed
- ✅ [03-build-plan.md](03-build-plan.md) — first cut scope + ลำดับ milestone M0–M5 — committed+pushed
- ✅ [DECISIONS.md](DECISIONS.md) — D1–D20 ครบ
- ⛔ **ยังไม่มีโค้ด** — เพิ่งวางเอกสารฐาน+architecture+build-plan เสร็จ ยังไม่ dispatch Sonnet รอบแรก

## first cut ที่ล็อกแล้ว (D17)
goal = **proof-of-loop ภายใน** + โชว์ wedge. Milestone:
`M0 skeleton(dual-stack) → M1 Identity&Tenancy → M2 Functional Project → M3 Deliverable+ตรวจรับ → M4 Task board → M5 Finance(thin)`
กฎ build: 1 milestone = vertical slice, **1 Sonnet dispatch = 1 stack** (BE นำ FE ครึ่งก้าว), data-model just-in-time (D19)

## ทำอะไรต่อ (เลือก 1 — ถาม User ก่อน)
1. **data-model doc** — schema ของ M0/M1 ก่อน (auth/workspace/membership/org-role), resolve `tenant_id` vs `workspace_id`, apply master-table convention (02 §7). ทำก่อนลงมือ M0 = มี schema baseline ให้ Sonnet
2. **ลงมือ M0 จริง** — dispatch Sonnet รอบแรก (scaffold Go hexagonal + Next.js + docker + goose + health + test). proof-of-loop ของจริงเริ่มตรงนี้
3. **flow-doc** (option) — user flow เต็ม (สมัคร → สร้าง/ถูกเชิญ → สลับ workspace)
4. **permission matrix** (option) — org × project role × action (ใช้ตอน M1/M2)

## Open threads (ยังไม่ตัดสิน — อย่าลืม)
- `tenant_id` vs `workspace_id` จะ collapse หรือคงสองคอลัมน์ — รอ data-model doc (02 §4.1)
- Postgres RLS เป็น hardening ชั้นสอง — candidate (02 §4.4)
- M5 trim-candidate ถ้าจวนตัว: penalty/adjustment + finance visibility ขั้นสูง (03 §2)
- (option เดิม) แก้ 00-workflow เพิ่มกฎ "1 session = 1 milestone" — User ยังไม่ตัดสิน

## เริ่ม session หน้ายังไง
อ่านตามลำดับ: **ไฟล์นี้ → 00-workflow → 01-vision → 02-architecture → 03-build-plan → DECISIONS** แล้วถาม User ว่าจะไปข้อไหนใน "ทำอะไรต่อ".
