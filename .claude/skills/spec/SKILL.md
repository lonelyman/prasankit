---
name: spec
description: Write a Sonnet-ready implementation spec for Prasankit and dispatch it through the vibe-code loop. Produces the workflow §5 spec format (Goal / Context / Files / Behavior / Tests / Out-of-scope / Acceptance), enforces the §4 gates (no written spec = no dispatch, no scope/architecture decided for the User, one stack per dispatch), injects project context (02-architecture patterns, 03-build-plan scope), and sets up the §6 review handoff. Trigger on /spec and proactively whenever about to dispatch Sonnet to implement a task — i.e. before any Agent(model: sonnet) implementation call.
---

# Spec & Dispatch

The heart of the loop (workflow §3): User intent → approach approved → **spec** → dispatch Sonnet → review → commit. This skill is the discipline you carry when you brief Sonnet — not advice to hand back to the User.

`docs/00-workflow.md` §5 (spec format) and §6 (review checklist) are **canonical**. This skill reproduces them as a working scaffold; if it ever drifts from the doc, the doc wins.

## 1. Gate check — before you write a single line of spec (workflow §4)

- **Approved approach exists?** The User must have decided the approach. If not → propose + trade-off + ask. Do **not** spec yet (loop §3, gate §4.3).
- **No scope/architecture decided for the User** (§4.3). If the task forces an unmade decision, surface it and stop.
- **One stack per dispatch** (D19). A task is **backend (Go)** *or* **frontend (Next.js)** — never both in one spec. If it spans both, split into two specs and **dispatch BE first** (BE leads FE by half a step).

If any gate fails, fix it before proceeding. No written spec = no dispatch (§4.1).

## 2. Write the spec — reproduce workflow §5 faithfully

```markdown
## Goal
หนึ่งประโยค: feature นี้ทำอะไร

## Context (Read first)
- ไฟล์ที่ต้องอ่านก่อน (พร้อม path)
- pattern เดิมที่เกี่ยวข้อง (อ้าง archive/v1-legacy ได้ผ่าน `git show`)
- decision rationale (ลิงก์ docs/ + D# ที่เกี่ยว)

## Files to touch
- [ ] new: path/... — เนื้อหา
- [ ] edit: path/... — แก้อะไร
- [ ] migration: app-api/database/migrations/000XXX_*.sql

## Expected behavior
- input → output ตัวอย่าง
- edge cases / error cases ที่ต้องจัดการ

## Tests required
- unit: function X กับ case Y
- integration (ถ้าต้อง): กับ Postgres/Redis จริง
- tenant isolation: case "tenant อื่นมองไม่เห็น" (02 §4.3) — บังคับสำหรับ repo ที่แตะ tenant-scoped table

## Out of scope (อย่าแตะ)
- file/feature ที่ Sonnet ห้ามแก้แม้อยากแก้

## Acceptance
- [ ] BE: `go build ./...` + `go vet ./...` + `gofmt -l` (ต้องว่าง) + `go test ./...` ผ่าน
- [ ] FE: `pnpm build` + `pnpm lint` + `pnpm test` + typecheck ผ่าน
- [ ] ถ้า dispatch แตะ Docker/compose: **`docker build` จริงต้องผ่าน** (ไม่ใช่แค่ `docker compose config`) — boot จริงถึงเจอ gotcha ที่ unit/config มองข้าม (M0 FE เจอ 3 ตัว: ขาด `.dockerignore`, Next standalone ไม่ bind `0.0.0.0`, healthcheck `localhost`→IPv6)
- [ ] อื่นๆ ที่ Opus กำหนด
```

## 3. Inject project context every time (don't make Sonnet rediscover it)

- **Patterns → `docs/02-architecture.md`:** hexagonal layers (§3, incl. GORM `WithContext` hygiene), isolation invariant (§4.3), tenant resolution/auth (§5), master-table convention — no enum (§7), presenter envelope + pagination (§8).
- **Scope → `docs/03-build-plan.md`:** which milestone this serves, what is explicitly out.
- **Reference → `archive/v1-legacy`:** prior implementation of the same pattern (`git show archive/v1-legacy:app-api/...`). Reference, **not** binding.
- **Decisions → `docs/DECISIONS.md`:** cite the D# whose rationale the task depends on.

## 4. Dispatch

- `Agent(model: sonnet, subagent_type: claude)` with the **full spec** as the prompt.
- Sonnet implements + tests → returns diff + summary. **Sonnet does not commit** (§2.2).
- If Sonnet hits ambiguity it must stop and ask back, not improvise architecture (§2.1).

## 5. Review before reporting to the User (workflow §6)

You (Opus) are the first reviewer — do **not** rubber-stamp Sonnet. Check:

- [ ] **Scope** — only files in "Files to touch" changed
- [ ] **Spec** — behavior matches "Expected behavior"
- [ ] **Pattern** — matches 02-architecture (layer boundaries, envelope, isolation, master tables)
- [ ] **Tests** — written as specified, edge cases covered, actually pass, tenant-isolation case present
- [ ] **Side effect** — nothing changed outside scope
- [ ] **Security** — input validation, tenant isolation, no leaked secrets
- [ ] **Style** — no dead code, no premature abstraction, no needless comments

Then report the User: **pass / needs-fix / rerun**. Never commit code Sonnet wrote that you have not reviewed (§4.2) — passing tests is not enough.

## Operating rules

- **No written spec = no dispatch** (§4.1). The spec is the review baseline; without it there is nothing to review against.
- **One stack per dispatch; BE leads FE half a step** (D19).
- **Sonnet must not expand scope** (§4.4). If it finds adjacent work, it reports "found X, did not do — out of spec" — it does not do it.
- **Fold any decision made while spec'ing into `docs/` + `DECISIONS.md` the same round** (§4.6) — otherwise next session it's gone.
- **workflow §5/§6 are the source of truth.** If this skill and the doc disagree, follow the doc and flag the drift.
