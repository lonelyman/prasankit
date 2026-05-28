# 99 — Handoff (อ่านก่อนเพื่อนเมื่อเปิด session ใหม่)

> บอก "ตอนนี้อยู่ตรงไหน + ทำอะไรต่อ". เป็น **living doc — เขียนทับทุกครั้งที่ปิด session**.
> รายละเอียดเต็มอยู่ใน docs ที่ลิงก์. log การตัดสินใจอยู่ใน [DECISIONS.md](DECISIONS.md).

## อัปเดตล่าสุด
**2026-05-28** — ปิด session: **M1 BE เสร็จถึง dispatch 4b** (data-model doc + 5 ก้อน BE commit ครบ). tenancy vertical ทำงานจริงทาง API: `signup → login → สร้าง workspace → เชิญ → รับเชิญ`. จุดต่อไป = **dispatch 5 Account-safety** (email-verify + password-reset) แล้วค่อย **M1 FE**. push แล้ว.

## สถานะตอนนี้
- ✅ docs ฐานครบ: [00-workflow](00-workflow.md) · [01-vision](01-vision.md) · [02-architecture](02-architecture.md) · [03-build-plan](03-build-plan.md) · [04-data-model](04-data-model.md) · [DECISIONS](DECISIONS.md) (D1–D32)
- ✅ **M0 walking skeleton (dual-stack)** — boot verified (จาก session ก่อน)
- ✅ **M1 data-model doc** ([04-data-model.md](04-data-model.md)) — ผ่าน external review รอบ 1, fold D24–D29
- ✅ **M1 BE — 5 dispatch แรก committed + reviewed (Opus รีวิวทุกก้อน, ไม่ rubber-stamp):**
  1. **Foundation** (`e0e3608`) — migrations 000002–000007 (14 ตาราง + seed master) + `pkg/{ids,securetoken,passwordhash}`
  2. **Auth core** (`6911ebf`) — signup/login/logout + opaque session (Redis) + `requireSession` + lockout + security_events. anti-enumeration (dummy bcrypt, generic error, atomic counter)
  3. **Email infra** (`8221bef`) — `email.Sender` port + stdlib SMTP adapter + **Mailpit** dev-catcher (compose) + MAIL_* config
  4. **Tenancy backbone (4a)** (`a55ec41`) — สร้าง workspace + owner membership (atomic + audit in-tx) + list-mine + `X-Workspace-Slug` resolver + `requireTenantContext` + **isolation invariant ของจริง** (+ isolation tests) + reserved-slug
  5. **Invitation (4b)** (`5ef704d`) — invite (owner/admin only ผ่าน `requireWorkspacePermission`, ส่ง email จริง best-effort) + accept (token + **email-match anti-hijack** + atomic) 
- ✅ decisions session นี้ fold แล้ว: **D24–D29** (data-model) + **D30** (SMTP/Mailpit) + **D31** (audit must-succeed in-tx) + **D32** (slug public, resolver 404/403, accept email-match)

## วิธีรัน / เทสต์ M1 (สำคัญ — กันงงรอบหน้า)
- `make dev-up` (infra→migrate→BE→FE→health). `make down` ปิด. **stack รันค้างไว้ตอนปิด session** (`prasankit_pgsql`/`prasankit_redis`/`prasankit_mailpit` healthy)
- **host ports:** API `18080`, Web `13000`, pg `15433`*, redis `16380`*, minio `19000/19001`, **Mailpit SMTP `11025` / UI-API `18025`** (`*`=local `.env` ต่างจาก example เพราะชนเครื่อง dev)
- **เทสต์ ต้องใช้ `cd app-api && make test`** (= `go test -p 1 ./...`) — **ห้าม `go test ./...` เปล่า ๆ** (integration test ใช้ DB+Mailpit ร่วมกัน + `TRUNCATE` → package ขนานตีกัน flake). test คืน DB+Mailpit ให้สะอาดหลังรัน
- **Mailpit UI:** `http://localhost:18025` (ดู email ที่ส่งตอน dev/เทสต์)
- `.env` (gitignored) มี MAIL_* + MAIL_INVITE_BASE_URL ครบแล้ว; fresh clone ดู `.env.example`

## ทำอะไรต่อ (เลือก — ถาม User ก่อน, ระบุข้อแนะนำ)
1. **M1 BE Dispatch 5: Account-safety (แนะนำ — ปิด M1 BE)** — email-verification (signup→ส่งลิงก์ยืนยัน→ flip account_status `pending_verification`→`active`) + password-reset (request→email→confirm). **consume `email.Sender` ที่มีแล้ว** (เพิ่ม MAIL_VERIFY_*/MAIL_RESET_* base URL+subject ใน config/.env — keys comment ไว้ใน `.env` แล้ว). ตาราง `auth_email_verification_tokens`/`auth_password_reset_tokens` พร้อม (มีจาก Foundation). ผ่าน `/spec`
2. **M1 FE (Next.js)** — login/signup + empty-state → create/accept-invite workspace (D20, D22). BE contract นิ่งแล้ว. เป็น dispatch แยก (1 dispatch = 1 stack, D19); จะทำ FE คู่กับ dispatch 5 หรือหลังก็ได้
3. **flow-doc / permission matrix** (option) — ถ้าอยากเขียน user flow เต็ม หรือ matrix สิทธิ์ก่อนลง M2

## Open threads (ยังไม่ตัดสิน — อย่าลืม)
- **Placeholder member + claim-by-email** = M2 (User request 2026-05-28, [04 §8](04-data-model.md)) — profile ที่ยังไม่มี account + ผูกทีหลัง verify email
- **Assignee/approver FK → `workspace_memberships(id)` vs `user_accounts(id)`** = ตัดสิน M2 ([04 §8](04-data-model.md))
- **Owner soft-delete invariant** — ห้าม soft-delete account ที่ owns active workspace (service-level ตอนทำ account-deletion + ownership transfer, post-M1, [04 §8](04-data-model.md))
- **Test parallel-safety (tech-debt)** — ตอนนี้พึ่ง `-p 1` (shared DB + global TRUNCATE). fix สะอาด = test DB แยก/scoped cleanup. ไม่ด่วน
- **`isUniqueViolation` ใช้ string-match `"23505"`** (auth + workspace repo) — harden เป็น typed `pgconn.PgError` ได้ทีหลัง
- **RLS hardening** (02 §4.4) — candidate security pass
- M5 finance trim-candidate ถ้าจวนตัว (03 §2)

## M1 = Identity & Tenancy (build-plan §5) — เหลืออะไร
ส่งแล้ว: auth (signup/login/logout, session, lockout) · workspace (1:1 tenant) + membership + invitation + resolver + `requireTenantContext`/`requireWorkspacePermission` + isolation invariant *ของจริง* + audit log + i18n code + master-table. **เหลือ:** account-safety (email-verify + password-reset) BE + **FE ทั้งหมดของ M1** (login/signup + empty-state create/accept-invite).

## เริ่ม session หน้ายังไง
อ่านตามลำดับ: **ไฟล์นี้ → 00-workflow → 01-vision → 02-architecture → 03-build-plan → 04-data-model → DECISIONS** แล้วถาม User ว่าจะไปข้อไหนใน "ทำอะไรต่อ" (แนะนำ dispatch 5 Account-safety ปิด M1 BE — ผ่าน `/spec` ตอน dispatch). อย่าลืม: dispatch ทุกก้อน = spec → Sonnet → **Opus review (ไม่ rubber-stamp)** → User เคาะ → commit; เทสต์ใช้ `make test`.
