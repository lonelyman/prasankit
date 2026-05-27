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
