# 01 — Vision (North Star)

> **Status: Canonical.** เอกสารนี้คือ vision และ scope หลักของ Prasankit — ใช้เป็น north star ตอนตัดสินใจว่า "อะไรอยู่ในระบบ / อะไรไม่อยู่".
> เป็น **design intent** ของทั้งระบบ ไม่ใช่สถานะปัจจุบัน และ **ไม่ล็อกขอบเขต demo แรก** — ลำดับการ build และ scope ของ demo แรกอยู่ในเอกสาร build-plan (ชุดถัดไป).
> การเปลี่ยน vision ทำตาม gate เดียวกับ [00-workflow.md](00-workflow.md) §8 (User เริ่ม → Kael trade-off → option external reviewer → User decide → แก้ doc + commit รอบเดียว).

## 1. North Star

**Prasankit คือ Multi-Tenant Project Management Platform สำหรับทีม/องค์กรที่ทำงานแบบโครงการ** — รองรับทั้งงานภายในองค์กรและงานลูกค้า/งานรับจ้าง ในระบบเดียว.

แต่ละองค์กรมี workspace ของตัวเอง (tenant) ที่ข้อมูลแยกขาดจากกัน. ภายใน workspace ผู้ใช้บริหารโปรเจคได้ครบวงจรตั้งแต่ตั้งโปรเจค, จัดทีม, แตกงาน (task), ติดตามงวดงาน/สิ่งส่งมอบ, คุมการเงินของโปรเจค, เก็บเอกสาร, ประกาศเรื่องสำคัญ และดู activity log ที่ตรวจสอบย้อนหลังได้.

เป้าหมายปลายทาง: เป็นแพลตฟอร์มกลางที่ทีมโครงการใช้แทนการกระจายงานไปหลายเครื่องมือ (sheet + chat + drive + ระบบการเงินแยก) โดยยังคงความเบาและเร็วพอจะ demo และใช้งานจริงได้.

### 1.1 Positioning — ทำไมไม่ใช้ Jira/Notion เฉยๆ

Prasankit ไม่ได้แข่งเป็น "ที่จดงาน" (Trello), "ระบบ issue ของ dev" (Jira) หรือ "วิกิ/เอกสาร" (Notion). **wedge ของเราคือสิ่งที่ทั้งสามไม่มีในตัว:**

- **งวดงาน + ตรวจรับ** — ส่งหลายรอบ, ผ่าน/ตีกลับ/มีเงื่อนไข, ส่งเร็ว/ตรงเวลา/ช้า
- **การเงินโครงการ** — งบ/ค่าใช้จ่าย/ค่าปรับ พร้อมคุมสิทธิ์การมองเห็น
- **lifecycle โครงการ** ตั้งแต่เสนองานถึงดูแลหลังส่งมอบ (§4.2)
- **multi-tenant + audit-ready** สำหรับองค์กร

เป้าหมายคือเป็น **"ระบบบริหารงานโครงการ/งานรับจ้าง"** ที่งานส่งมอบกับเงินอยู่ในที่เดียว — **ไม่ไล่ให้ task ลึกเท่า Jira หรือเอกสารยืดหยุ่นเท่า Notion**. ส่วน task/เอกสารทำให้ "พอใช้ดี" ไม่ใช่ "ลึกสุด"; ถ้าเริ่มเฝือไปทางนั้นถือว่าหลุด wedge.

## 2. ทำไมต้อง rebuild (v2)

v1 ถูก archive ไว้ที่ `archive/v1-legacy` (ดู [00-workflow.md](00-workflow.md) §7). มันทำงานได้ระดับหนึ่ง แต่ **เกิดก่อน vibe-code loop** — code, spec และ docs ไม่ได้ถูกผลิตผ่าน role/gate ที่ตกลงกันใน [00-workflow.md](00-workflow.md).

v2 จึงเริ่มใหม่บนหลักการนี้:

- **ทุก production code เกิดจาก loop** — spec ก่อน, dispatch, review, แล้วค่อย commit. ไม่มี code ที่ไม่ผ่าน spec เป็นลายลักษณ์อักษร.
- **Docs เป็น single source of truth** — vision/scope/decision อยู่ใน `docs/` และเป็นของ User. code ตาม docs ไม่ใช่ docs ตาม code.
- **Vision เท่าเดิม, scope แรกเล็กลงได้** — เป้าหมายระบบยังใหญ่เท่า v1 แต่ "ทำอะไรก่อน" คุมผ่าน build-plan ไม่ใช่พยายามทำทุก module พร้อมกัน.

v1 ใช้เป็น **reference** ของ domain model และ pattern ที่เคยตกผลึก (lifecycle, role, status sets) — อ้างอิงได้ผ่าน `git show archive/v1-legacy:path/...` แต่ไม่ถือเป็นข้อผูกมัดทาง implementation.

## 3. Target Users

วางตำแหน่งเป็นแพลตฟอร์มสำหรับทีม/องค์กรที่ทำงานแบบโครงการ โดย**ไม่ล็อกอุตสาหกรรมเดียว**:

- **งานภายในองค์กร** — งานของแผนก IT, HR, Operation หรือทีมภายใน
- **งานลูกค้า / งานรับจ้าง** — implementation, consulting, software, campaign, project service

รอบแรกใช้ template กลางสำหรับงานโครงการทั่วไป และออกแบบให้ขยายเป็น template เฉพาะทางได้ในอนาคต (เช่น Software Development, Marketing Campaign, Construction, Consulting, Loan/Credit Operation) — template เฉพาะทางเป็น future extension ไม่ใช่ vision หลักของรอบนี้.

## 4. Product Model

### 4.1 Workspace / Tenant
แต่ละองค์กร = 1 workspace = 1 tenant. ข้อมูลทุกอย่างผูกกับ tenant และแยกขาดจาก tenant อื่น (tenant isolation). การ resolve tenant ต้องเกิดฝั่ง backend เท่านั้น ไม่เชื่อค่าจาก client.

**User ↔ Tenant:** 1 identity เป็นสมาชิกได้หลาย workspace (membership แบบ many) — tenant ถูก resolve ต่อ session ฝั่ง backend ตามว่ากำลังทำงานใน workspace ไหน. รอบนี้ **ไม่ทำ cross-tenant account switching UI** (ดู §9): รองรับใน data model แต่ยังไม่เปิด UX สลับ tenant. กลไก resolution (token/session) เป็นรายละเอียดของ `02-architecture`.

### 4.2 Project Lifecycle
Project Status บอกสถานะภาพรวมของทั้งโปรเจค (คนละเรื่องกับ Task Status):

| Status | ภาษาไทย | ความหมาย |
| --- | --- | --- |
| Draft | ร่าง | เก็บข้อมูลไว้ก่อน ยังไม่ครบ หรือเป็นแนวคิดเบื้องต้น |
| Planning | วางแผน | ยังไม่เริ่มจริง แต่เตรียมทีม เอกสาร แผน งบ |
| Proposal | เสนองาน | เสนอราคา / ยื่นประมูล / รอลูกค้าตัดสินใจ |
| Active | ดำเนินงาน | เริ่มทำจริง มี task, team, deliverable, ค่าใช้จ่าย |
| Closing | ปิดงาน | ส่งมอบหลักแล้ว/ใกล้จบ รอตรวจรับ เคลียร์เอกสาร |
| Maintenance | ดูแลหลังส่งมอบ / MA | ส่งมอบแล้ว แต่ยังมี MA / warranty / support ตามสัญญา |
| Closed | จบงาน | จบภาระผูกพันทั้งหมด |
| Archived | จัดเก็บ | ซ่อนจากงานหลัก แต่ค้นย้อนหลังได้ |

### 4.3 Project Type

| Type | ภาษาไทย | คำอธิบาย |
| --- | --- | --- |
| Internal Project | งานภายในองค์กร | งานภายใน อาจมาจากแผนกตัวเองหรือแผนกอื่น (ระบุ Requesting Unit แทนการสร้าง type ใหม่) |
| Client Project | งานลูกค้า / งานรับจ้าง | งานให้ลูกค้า/ผู้ว่าจ้างภายนอก |

## 5. Core Modules (ภาพรวมเป้าหมาย)

นี่คือ **เป้าหมายเต็มของระบบ** — ไม่ใช่ทุก module ต้องอยู่ใน demo แรก. ลำดับการ build อยู่ใน build-plan.

| # | Module | ขอบเขตเป้าหมาย |
| --- | --- | --- |
| 1 | Project Profile | ข้อมูลหลักของโปรเจค: ชื่อ, type, status, วันเริ่ม/สิ้นสุด, เจ้าของ, รายละเอียด |
| 2 | Project Team / Role | สมาชิกโปรเจค, บทบาท, สิทธิ์พื้นฐาน (ดู §6) |
| 3 | Project Task | Jira-lite board, ลากย้ายสถานะ, status แยกรายโปรเจค (ดู §7) |
| 4 | Deliverable / งวดงาน | งวดงาน/สิ่งส่งมอบ, กำหนดส่ง, ส่งจริง, ตรวจรับ, ส่งหลายรอบ |
| 5 | Project Finance | Budget, Expense, Penalty/Adjustment พร้อมคุมการมองเห็น |
| 6 | Project Document | File Center: สร้าง folder, upload, ผูกกับ project/task/deliverable |
| 7 | Activity Log | Timeline ดูง่าย + ข้อมูลหลังบ้าน audit-ready |
| 8 | Announcement | ประกาศสำคัญของโปรเจค (ปักหมุด, ระดับความสำคัญ) — ไม่ใช่ chat |

## 6. Role Model

ระบบแยก **3 มิติ** ที่คนมักปนกัน:

- **Role** = สิทธิ์การใช้งานในระบบ
- **Project Position** = หน้าที่ในโปรเจค (เช่น Technical Lead)
- **Company Position** = ตำแหน่งงานในบริษัท

ทั้งสามแยกกันได้อิสระ เช่น *Infrastructure Manager* (company position) อาจเป็น *Technical Lead* (project position) และมี role เป็น *Project Manager*.

### 6.1 Organization Role

| Role | ภาษาไทย | ขอบเขต |
| --- | --- | --- |
| Owner | เจ้าของ | ทำได้ทุกอย่างในองค์กร |
| Admin | ผู้ดูแลระบบ | ดูแลระบบ, จัดการข้อมูลพื้นฐาน, ช่วยบริหาร workspace |
| Executive | ผู้บริหาร | ดูภาพรวมข้ามโปรเจค, dashboard/report ระดับองค์กรตามสิทธิ์ |
| User | ผู้ใช้งาน | เห็นเฉพาะโปรเจค/งานที่ตัวเองเกี่ยวข้อง |

### 6.2 Project Role

| Role | ภาษาไทย | ขอบเขต |
| --- | --- | --- |
| Project Owner | เจ้าของโปรเจค | รับผิดชอบภาพรวมของโปรเจค |
| Project Manager | ผู้จัดการโปรเจค | จัดการงาน, ทีม, งวดงาน, ความคืบหน้า |
| Member | สมาชิกโปรเจค | ทำงานตามที่ได้รับมอบหมาย |
| Finance | การเงินโปรเจค | ดู/จัดการข้อมูลการเงินตามสิทธิ์ |
| Viewer | ผู้ดูอย่างเดียว | ดูข้อมูลที่อนุญาต โดย default ไม่เห็นข้อมูลการเงิน |

### 6.3 External Collaborators / Client Access
การดึงลูกค้า/คนนอกเข้าร่วม Client Project ทำผ่าน **invite เข้า workspace** — ไม่สร้าง client portal แยก:

- มอบ **Org Role = User** (เห็นเฉพาะสิ่งที่เกี่ยวข้อง)
- มอบ **Project Role = Viewer หรือ Member** เฉพาะโปรเจคนั้น
- ระบบ gate ด้วย Project Role → คนนอกไม่เห็นข้อมูลการเงิน (default) และไม่เห็นโปรเจคอื่นใน workspace เดียวกัน

permission matrix รายละเอียดอยู่ในเอกสารสิทธิ์ (ชุดถัดไป).

## 7. Task Model

- ใช้แนวคิด **Jira-lite** — board + ลากย้ายสถานะ. **มีได้หลาย view เริ่มที่ board + list** (calendar/table = future)
- **การ์ดงานมี comment, checklist, attachment** — เป็นที่คุยงานระดับ task (คนละเรื่องกับ Announcement ระดับโปรเจค). การแตกย่อยลึกแบบ epic/sub-task = future, รอบนี้ใช้ checklist แทน
- **Task Status แยกตามรายโปรเจค** — ระบบ seed default ให้ตอนสร้างโปรเจค แต่แต่ละโปรเจคเพิ่ม/แก้/เรียงลำดับเองได้
- **ทุก status (รวม custom) ต้อง map กับ 1 ใน 3 universal category: `To Do` / `In Progress` / `Done`** — ให้ระบบ rollup ความคืบหน้าข้ามโปรเจคและทำ Executive dashboard ได้ โดยไม่ล็อก custom status รายโปรเจค
- Project Status (§4.2) กับ Task Status เป็นคนละระบบ
- Task ผูกกับ Deliverable ได้

Default Task Status: `Backlog → To Do → In Progress → Review → Testing → Blocked → Done → Cancelled`

## 8. หลักการข้ามระบบ (Cross-cutting)

- **i18n ตั้งแต่แรก** — เริ่มไทย + อังกฤษ. status, menu, label, message ต้องแปลได้
- **Audit-ready Activity Log** — เก็บเชิงโครงสร้าง (actor, action, resource type/id, old/new value, timestamp, IP, user agent, result, tenant id, project id) ไม่ใช่ข้อความลอย
- **Finance visibility control** — ข้อมูลการเงินซ่อน/จำกัดสิทธิ์การมองเห็นได้ เพราะบางโปรเจคถือเป็นข้อมูลลับ
- **Security-by-design** — tenant isolation, access control, change tracking. ออกแบบให้ต่อยอดสู่มาตรฐานได้ (อ้างอิงอนาคต: OWASP ASVS, OWASP Top 10, ISO/IEC 27001:2022) แต่ยังไม่ claim compliance เต็มรูปแบบ

## 9. Out of Scope (North-Star Non-Goals)

สิ่งที่ **ไม่อยู่ในวิสัยทัศน์รอบนี้** แม้ระบบจะโตเต็มที่ — กันไม่ให้ scope ขยายเงียบๆ:

Real-time Chat · Payment Gateway · Subscription/Billing · Custom Domain ของลูกค้า · SSO/Login ผ่านระบบองค์กร · Cross-tenant Account Switching UI · Mobile App (ใช้ Responsive Web ก่อน) · Advanced Approval Workflow · Gantt ขั้นสูง · Automation Rule · AI Assistant · Online Document Editor / Wiki · E-signature · OCR · Invoice/Payment Tracking เต็มระบบ · Compliance Certification เต็มรูปแบบ

- รอบ MVP ใช้ **Announcement แทน Chat**
- Finance รอบนี้มี Budget/Expense/Penalty-Adjustment แต่ **ไม่ทำบัญชีเต็มระบบ** (ภาษี, ใบแจ้งหนี้, รับชำระ, P/L เต็ม)
- Activity Log audit-ready แต่ยัง **ไม่ claim compliance**
- Document = **File Center (เก็บไฟล์)** ไม่ใช่ in-app editor/วิกิแบบ Notion

## 10. Vision vs. First Cut

เอกสารนี้คือ **north star เต็ม** — §5 list ครบทุก module เป้าหมาย. แต่ **ขอบเขต demo แรกยังไม่ล็อก** (ตามที่ตกลง).

> **อะไรทำก่อน-หลัง, demo แรกครอบคลุมแค่ไหน, ลำดับ build → อยู่ในเอกสาร build-plan (ชุดถัดไป).**
> เมื่อใดก็ตามที่ build-plan เลือก subset ของ §5 มาทำก่อน ต้องไม่ขัดกับ vision นี้ — ถ้าขัด แปลว่าต้องแก้ vision ก่อน (gate §8 ของ workflow) ไม่ใช่ปล่อยให้ build-plan เดินสวน vision.

## 11. ความสัมพันธ์กับเอกสารอื่น

- [00-workflow.md](00-workflow.md) — วิธีทำงานร่วมกัน (role, loop, gate). vision นี้อยู่ใต้ gate การเปลี่ยนแปลงเดียวกัน
- **02+ (ชุดถัดไป)** — architecture/stack, data model, build-plan ฯลฯ จะอ้างอิง vision นี้เป็นฐาน
- `archive/v1-legacy` — reference ของ domain model เดิม ไม่ใช่ข้อผูกมัด implementation
