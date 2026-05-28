# 99 — Handoff (อ่านก่อนเพื่อนเมื่อเปิด session ใหม่)

> บอก "ตอนนี้อยู่ตรงไหน + ทำอะไรต่อ". เป็น **living doc — เขียนทับทุกครั้งที่ปิด session**.
> รายละเอียดเต็มอยู่ใน docs ที่ลิงก์. log การตัดสินใจอยู่ใน [DECISIONS.md](DECISIONS.md).

## อัปเดตล่าสุด
**2026-05-28** — ปิด session: **M1 Backend เสร็จครบทุก dispatch** (account-safety 5a/5b/5c ปิดจบ). M1 BE ตอนนี้มี: auth (signup/login/logout, session, lockout) · workspace+membership+invitation · **email-verify + login-gating** · **password-reset** · **lazy session revocation**. จุดต่อไป = **M1 FE (Next.js)** เพื่อปิด M1 ทั้ง milestone. push ครบขึ้น origin/dev แล้ว.

## สถานะตอนนี้
- ✅ docs ฐานครบ: [00-workflow](00-workflow.md) · [01-vision](01-vision.md) · [02-architecture](02-architecture.md) · [03-build-plan](03-build-plan.md) · [04-data-model](04-data-model.md) · [DECISIONS](DECISIONS.md) (**D1–D36**)
- ✅ **M0 walking skeleton (dual-stack)** — boot verified
- ✅ **M1 data-model doc** — fold D24–D29
- ✅ **M1 BE — committed + reviewed ครบ (Opus review ไม่ rubber-stamp + external review ต่าง-model หลายรอบ):**
  - Foundation (migrations 000002–000007 + `pkg/{ids,securetoken,passwordhash}`) · Auth core · Email infra (Mailpit) · Tenancy backbone (4a) · Invitation (4b) — *จาก session ก่อน*
  - **5a Email verification** (`f052b64`) — signup auto-ส่ง verify email + confirm (guarded consume, flip `pending_verification→active`) + resend + **login block จนกว่า verify** (403 `email_not_verified` หลัง password ผ่าน)
  - **5b Password reset** (`e10efb8`) — request (anti-enum 204 เสมอ) + confirm (guarded consume → set password + clear lockout) + `password_changed_at`; reset **ไม่** verify email; TTL 1h
  - **5c Lazy session revocation** (`a7cbcf1`) — `requireSession`→`ResolveSessionAccount`: session ที่สร้างก่อน `password_changed_at` ถูกลบ + 401 (reset เตะอุปกรณ์เก่าทุกตัว, ไม่ต้องมี Redis index)
- ✅ decisions session นี้ fold แล้ว: **D33** (login gating) · **D34** (anti-abuse defer) · **D35** (password-reset BE) · **D36** (lazy session revocation)

## วิธีรัน / เทสต์ M1 (สำคัญ — กันงงรอบหน้า)
- `make dev-up` (infra→migrate→BE→FE→health). `make down` ปิด.
- **host ports:** API `18080`, Web `13000`, pg `15433`*, redis `16380`*, minio `19000/19001`, **Mailpit SMTP `11025` / UI-API `18025`** (`*`=local `.env` ต่างจาก example เพราะชนเครื่อง dev)
- **เทสต์ ต้องใช้ `cd app-api && make test`** (= `go test -p 1 ./...`) — **ห้าม `go test ./...` เปล่า ๆ** (integration ใช้ DB+Redis+Mailpit ร่วม + `TRUNCATE` → ขนานตีกัน flake). ⚠️ การ์ดเตือน: Sonnet สรุปว่า "เสร็จ" บางทีไม่จบจริง — **รัน `make test` + อ่าน diff เองทุกครั้ง**; เทสต์ integration ที่ "cached/skip" ให้ force-run `-count=1 -v` ดูว่า PASS จริงไม่ใช่ skip
- **Mailpit UI:** `http://localhost:18025`
- `.env` (gitignored) มี MAIL_* + **MAIL_INVITE_BASE_URL + MAIL_VERIFY_BASE_URL + MAIL_RESET_BASE_URL** ครบ; fresh clone ดู `.env.example`

## ทำอะไรต่อ (เลือก — ถาม User ก่อน, ระบุข้อแนะนำ; ถามแบบ numbered list ในข้อความ ไม่ใช้ option-picker)
1. **M1 FE (Next.js) — แนะนำ, ปิด M1 ทั้ง milestone** — login/signup + empty-state → create/accept-invite workspace (D20). FE stack = Next.js App Router + Tailwind + pnpm + client `fetch(credentials:'include')` + `X-Workspace-Slug` + Vitest, อยู่ `app-web/` (D18/D22). BE contract นิ่ง 100% แล้ว. ผ่าน `/spec`, 1 dispatch = 1 stack (D19). FE ต้อง consume: `/auth/{signup,login,logout,me,verify-email,verify-email/resend,password-reset/request,password-reset/confirm}` + `/workspaces` + `/invitations/accept`; หน้า `/verify-email`, `/reset-password`, `/invitations/accept` รับ `?token=`
2. **flow-doc / permission matrix** (option) — user flow เต็ม หรือ matrix org×project role×action ก่อนลง M2
3. **Security/rate-limit pass** (option) — ทำ rate-limit รวมทุก auth endpoint (D34 เลื่อนไว้)

## Open threads (ยังไม่ตัดสิน — อย่าลืม)
- **Rate-limit/cooldown auth endpoints** (login/signup/resend/reset) — เลื่อนไป dedicated pass (D34), ยังไม่มี ratelimit infra
- **requireSession +1 query** (5c โหลด identity แยกเพื่อเช็ค `password_changed_at`) — ถ้าเป็น hot path เปลี่ยนเป็น single JOIN account+identity (external review 5c เสนอ; decline สำหรับ M1)
- **Session-revocation semantics เมื่อมี multi-identity/MFA** (M2+) — `FindByUserAccountID` คืน identity เดียว ตอนนี้พอ; ต้อง rethink เมื่อ 1 account มีหลาย identity
- **Dead code:** `extractVerifyTokenFromDB` ใน `auth_handler_test.go` (ตกค้างจาก 5a, ไม่เคยถูกเรียก) — ยังไม่ลบ (raw-SQL match เสี่ยง); ลบได้ทีหลัง
- **Placeholder member + claim-by-email** = M2 ([04 §8](04-data-model.md)) — profile ที่ยังไม่มี account + ผูกทีหลัง verify email
- **Assignee/approver FK → `workspace_memberships(id)` vs `user_accounts(id)`** = ตัดสิน M2 ([04 §8](04-data-model.md))
- **Owner soft-delete invariant** — ห้าม soft-delete account ที่ owns active workspace (service-level ตอนทำ account-deletion + ownership transfer, post-M1)
- **Test parallel-safety (tech-debt)** — พึ่ง `-p 1` (shared DB + global TRUNCATE). fix สะอาด = test DB แยก/scoped cleanup. ไม่ด่วน
- **`isUniqueViolation` string-match `"23505"`** — harden เป็น typed `pgconn.PgError` ได้ทีหลัง
- **RLS hardening** (02 §4.4) — candidate security pass · M5 finance = trim-candidate (03 §2)

## M1 = Identity & Tenancy (build-plan §5) — เหลืออะไร
**BE เสร็จครบแล้ว** (auth + account-safety + workspace/membership/invitation + resolver + permission gate + isolation invariant + audit log + i18n code + master-table). **เหลือแค่ M1 FE** (login/signup + empty-state create/accept-invite workspace) → ทำเสร็จ = ปิด M1 ทั้ง milestone แล้วไป M2 (Functional Project).

## เริ่ม session หน้ายังไง
อ่านตามลำดับ: **ไฟล์นี้ → 00-workflow → 01-vision → 02-architecture → 03-build-plan → 04-data-model → DECISIONS** แล้วถาม User ว่าจะไปข้อไหนใน "ทำอะไรต่อ" (แนะนำ **M1 FE** ปิด milestone — ผ่าน `/spec`, stack = Next.js ที่ `app-web/`). อย่าลืม: dispatch ทุกก้อน = spec → Sonnet → **Opus review (ไม่ rubber-stamp) + trust-but-verify รัน make test เอง** → User เคาะ → commit; เทสต์ใช้ `make test`.
