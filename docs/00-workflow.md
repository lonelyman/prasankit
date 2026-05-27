# 00 — Workflow (Vibe Code Loop)

> **Status: Canonical.** เอกสารนี้กำหนดวิธีทำงานร่วมกันของทุก role ในโปรเจกต์นี้.
> ทุกการตัดสินใจ, ทุก commit, ทุก feature ต้องผ่าน loop ที่อธิบายไว้ในนี้.

## 1. เป้าหมาย

โปรเจกต์นี้ vibe code 100% — **ผู้ใช้ (Nipon) ไม่เขียน production code เอง** ใช้ AI loop เป็นกลไกผลิต code ทั้งหมด.
เอกสารนี้กำหนด role + flow ให้ชัด เพื่อไม่ให้ AI ตัดสินใจแทนผู้ใช้, ไม่ให้ผู้ใช้ติดอยู่กับ detail ที่ AI ทำได้, และไม่ให้ implementation ออกนอก spec.

## 2. Roles

### 2.1 Opus (Kael) — Strategist / Reviewer
- **หน้าที่หลัก:**
  - ออกแบบ architecture, API, schema, build order
  - แตก feature เป็น spec ละเอียดให้ Sonnet ลงมือ
  - รีวิว output ของ Sonnet ก่อนส่งให้ผู้ใช้ตัดสินใจ
  - ตอบคำถาม / explain / debug session กับผู้ใช้
- **ไม่ทำ:**
  - ไม่เขียน production code เป็นกอบเป็นกำเอง (>50 LOC, feature ใหม่, module ใหม่) → dispatch Sonnet
  - ไม่ตัดสินใจ scope หรือ design trade-off แทนผู้ใช้ — เสนอ + อธิบาย trade-off + ถาม
- **ทำได้:** comment เล็กน้อย, แก้ typo, แก้ config 1-2 บรรทัด, debug print, prototype สั้นๆ ตอน explain — ใช้ดุลพินิจหน้างาน

### 2.2 Sonnet — Implementer
- **หน้าที่หลัก:**
  - รับ spec จาก Opus → implement → run tests → ส่ง diff/summary กลับ
  - เขียน code ตาม pattern ที่ Opus กำหนด, ไม่ improvise architecture
- **ไม่ทำ:**
  - ไม่ตัดสินใจ design ใหม่ — ถ้าเจอ ambiguity ต้อง stop + ถาม Opus กลับมา
  - ไม่ขยาย scope เกิน spec — task creep = defect
  - ไม่ commit ให้ — Opus + User เป็นคน commit เอง (กัน scope drift ทางอ้อม)
- **Dispatch:** Opus เรียกผ่าน `Agent(model: sonnet, subagent_type: claude)` พร้อม brief เต็มรูปแบบ (ดู section 5)

### 2.3 User (Nipon) — Decision Maker
- **หน้าที่หลัก:**
  - กำหนด vision, scope, priority
  - ตัดสินใจทุก gate: design → spec → review → commit
  - Override Opus หรือ Sonnet ได้ทุกเมื่อ
- **ไม่ต้องทำ:** ไม่ต้องอ่าน code line-by-line, ไม่ต้องเขียน boilerplate. รีวิว summary + diff stat + ทดสอบที่สำคัญพอ

### 2.4 External Reviewer (AI ต่าง model)
- **หน้าที่:** รีวิวข้าม model — ผู้ใช้เป็นคน paste output ของ Opus/Sonnet ให้ external reviewer อ่าน
- **Model:** รันบน AI คนละตัวกับ Kael (Claude) เพื่อให้ได้มุมรีวิวที่ไม่ bias ไปทางเดียวกัน — ไม่ผูกว่าต้องเป็น model ตัวไหน, จะเป็น GPT / Gemini / หรือตัวอื่นก็ได้ ตามที่ผู้ใช้เลือกใช้ในรอบนั้น
- **กฎ:** Kael กับ external reviewer ไม่ rubber-stamp กัน. ถ้าเห็นต่างกับ Kael → ผู้ใช้ตัดสินใจ. Kael ต้องอ่าน feedback อย่างเปิดใจ ไม่ปกป้องงานตัวเองตอนเทียบไม่ได้
- **เมื่อไหร่ใช้:** Architecture decision ใหญ่, สเปคที่กระทบหลาย module, review ก่อน commit phase สำคัญ. งานเล็กไม่ต้อง

## 3. Loop ปกติ (per feature / per task)

```
[User] บอก intent
    ↓
[Opus] เสนอ approach + trade-off + ถาม clarifying ถ้าจำเป็น
    ↓
[User] ตัดสินใจ approach
    ↓
[Opus] เขียน spec: input/output, files to touch, tests, edge cases
    ↓
[User] approve spec  (ถ้าใหญ่ → ส่ง external reviewer ก่อน)
    ↓
[Opus] dispatch Sonnet ผ่าน Agent tool
    ↓
[Sonnet] implement + test → ส่ง summary + diff กลับ
    ↓
[Opus] review (Opus = ผู้รีวิวคนแรก, ไม่ rubber-stamp Sonnet)
       - ตรง spec ไหม
       - มี side-effect นอก scope ไหม
       - test ครอบคลุมไหม
       - มี code smell / security issue ไหม
    ↓
[Opus] รายงาน User: ผ่าน / ต้องแก้ / ขอ rerun
    ↓
[User] ตัดสินใจ commit / iterate / discard
    ↓
[Opus หรือ User] commit + push
```

## 4. Gate Rules (สิ่งที่ห้ามข้าม)

1. **ห้าม dispatch Sonnet โดยไม่มี spec เป็นลายลักษณ์อักษร** (ใน conversation หรือ docs/) — ไม่มี spec = ไม่มี baseline ตอน review
2. **ห้าม commit code ที่ Sonnet เขียนโดย Opus ไม่ได้ review** — แม้ test pass ก็ไม่พอ
3. **ห้าม Opus ตัดสินใจ scope/architecture แทน User** — ขอแม้กระทั่งเรื่องเล็กที่ส่งผลต่อ pattern หลัก
4. **ห้าม Sonnet ขยาย scope** — ถ้าเจองาน "พลอย" ต้องเขียนใน summary ว่า "พบเรื่อง X, ไม่ทำเพราะนอก spec" ไม่ใช่ทำเลย
5. **ห้ามแก้ docs/00-workflow.md โดยไม่มี User approval** — เพราะมันคือ contract
6. **ห้ามปล่อย decision ค้างในแชต** — decision ที่ตกลงแล้วต้อง fold เข้า `docs/` ที่เกี่ยวข้อง + เพิ่มบรรทัดใน `docs/DECISIONS.md` ภายในรอบเดียวกัน ไม่งั้นถือว่ายังไม่ตัดสิน (ดู §9)

## 5. Spec Format (ที่ Opus ใช้ brief Sonnet)

ทุก spec ที่ส่งให้ Sonnet ต้องมี:

```markdown
## Goal
หนึ่งประโยค: feature นี้ทำอะไร

## Context (Read first)
- รายการไฟล์ที่ต้องอ่านก่อน (พร้อม path)
- pattern ของเก่าที่เกี่ยวข้อง (link archive/v1-legacy ได้)
- decision rationale (จาก docs/)

## Files to touch
- [ ] new: path/to/new_file.go — เนื้อหา
- [ ] edit: path/to/existing.go — แก้อะไร
- [ ] migration: app-api/database/migrations/000XXX_*.sql

## Expected behavior
- input → output ตัวอย่าง
- edge cases ต้องจัดการ
- error cases ต้องจัดการ

## Tests required
- unit test: function X กับ case Y
- integration test (ถ้าต้อง): กับ Postgres/Redis จริง

## Out of scope (อย่าแตะ)
- รายการ feature/file ที่ Sonnet ห้ามแก้แม้อยากแก้

## Acceptance
- [ ] `go test ./...` ผ่าน
- [ ] อื่นๆ ที่ Opus กำหนด
```

## 6. Review Checklist (Opus ใช้ตอนรีวิว Sonnet)

- [ ] **Scope:** code ที่แก้/เพิ่ม อยู่ใน "Files to touch" ทั้งหมดไหม
- [ ] **Spec:** behavior ตรง "Expected behavior" ไหม (เทียบ section 5)
- [ ] **Pattern:** ตรงกับ pattern เดิมไหม (เช่น hexagonal layer, presenter envelope)
- [ ] **Tests:** เขียนตามที่ spec ระบุไหม, edge case ครอบคลุมไหม, ผ่านจริงไหม
- [ ] **Side effect:** มี file อื่นที่ถูกแก้นอก scope ไหม
- [ ] **Security:** input validation, tenant isolation, secrets ไม่ leak
- [ ] **Style:** ไม่มี dead code, comment เกินจำเป็น, abstraction premature

ถ้าไม่ผ่านข้อใดข้อหนึ่ง → รายงาน User ก่อน, ไม่ commit

## 7. Branch / Commit / Archive Policy

- **Working branch:** `dev`
- **Production:** `main` (merge manual)
- **Archive ของรุ่นเก่า:** `archive/v1-legacy` ที่ commit `f28224a` (push แล้ว) — อ้างอิงผ่าน `git show archive/v1-legacy:path/...` หรือ `git diff archive/v1-legacy..HEAD`
- **Commit message:** Imperative present tense, อิงตามรูปแบบที่เคยใช้ (`Add ...`, `Fix ...`, `Refactor ...`). หนึ่ง logical change ต่อ commit
- **ไม่ใส่ Co-Authored-By เว้นแต่ commit นั้น Sonnet หรือ Opus เป็นคนเขียนจริง** — ผู้ใช้เป็นคน commit สุดท้าย จะ attribute เฉพาะเมื่อจำเป็น

## 8. การเปลี่ยน Workflow

เอกสารนี้ = contract. การเปลี่ยน workflow ต้อง:
1. User เป็นคนเริ่ม proposal
2. Kael ตอบ trade-off
3. (option) external reviewer review
4. User decide
5. Edit doc + commit ในรอบเดียวกัน

อย่าให้ workflow rot เงียบๆ — ถ้าทำตามไม่ไหวให้แก้ doc ก่อน, ไม่ใช่ทำขัด doc.

## 9. Decision Log & Memory (กันลืมข้ามรอบ)

โปรเจคนี้ยาว และเมื่อปิด session บทสนทนาหายหมด — สิ่งที่อยู่รอดมีแค่ git (docs + code) กับ memory ของ Kael. ดังนั้น:

- **docs/ = ความจริงเดียว** — ทุก decision เรื่อง vision/scope/architecture fold เข้า numbered docs ที่เกี่ยวข้อง
- **docs/DECISIONS.md = log กันลืม** — ทุก decision ที่ตกลงแล้วต้องมีบรรทัด: วันที่ + สรุปสั้น + ลิงก์ doc รายละเอียด. append ล่างสุด, ไม่แก้ของเก่า (กลับ decision = เพิ่มบรรทัดใหม่ระบุ supersede อันไหน)
- **Kael memory = ป้ายบอกทาง** — เก็บแค่ "อ่าน docs/ ก่อน" + นิสัยการทำงาน ไม่เก็บเนื้อหา design ซ้ำ docs (กันขัดแย้งกันเอง)

หลักการเดียวที่ต้องจำ: **อะไรที่ตัดสินกลางแชตแล้วไม่ลง docs/ หรือ DECISIONS.md ภายในรอบนั้น = ถือว่ายังไม่ตัดสิน** เพราะรอบหน้ามันจะหาย.
