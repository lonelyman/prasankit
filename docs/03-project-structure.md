# ชุดที่ 3: Project Structure / Project Detail Concept

เอกสารออกแบบระบบ Project Management Platform

> **Status: Forward-looking design intent (MVP-1+).** Project module เวอร์ชัน MVP-0 implement แค่บางส่วน (list/create/get/update/members/positions). ดู [docs/09-api-design.md §9.4](09-api-design.md) สำหรับสถานะ endpoint จริง และ [docs/99-current-handoff.md](99-current-handoff.md) สำหรับสิ่งที่ทำเสร็จแล้ว.

## ภาพรวมเอกสาร

เอกสารชุดนี้สรุปแนวคิดโครงสร้าง Project, หน้า Project Detail และความสัมพันธ์ของข้อมูลหลักภายใน Project ตามที่ตกลงกันแล้ว

| ลำดับ | หัวข้อ |
| --- | --- |
| 1 | Project Profile |
| 2 | Project Detail Layout |
| 3 | Project Team & Position |
| 4 | Project Task Structure |
| 5 | Timeline + Deliverable Structure |
| 6 | Project Finance Structure |
| 7 | Project Document Structure |
| 8 | Project Announcement & Activity |
| 9 | Project Settings |

## 1. Project Profile

### 1.1 Created By / Project Owner

หลักการสำคัญ: Created By ไม่เท่ากับ Project Owner

| รายการ | ความหมาย / หลักการ |
| --- | --- |
| Created By | คนที่กดสร้างโปรเจค ระบบบันทึกอัตโนมัติ เปลี่ยนไม่ได้ และใช้เพื่อ audit |
| Project Owner | Project Role หนึ่งใน Project Team เป็นผู้รับผิดชอบสูงสุดของโปรเจค |

- Project ต้องมี Project Owner อย่างน้อย 1 คนเสมอ
- Project Owner มีได้หลายคน
- คนสร้างถูกตั้งเป็น Project Owner เริ่มต้น
- สามารถเปลี่ยน โอน หรือเพิ่ม Project Owner ได้ในหน้า Project Team
- ห้ามลบ Project Owner คนสุดท้าย
### 1.2 Project Role / Project Position

| แนวคิด | ใช้ทำอะไร |
| --- | --- |
| Role | สิทธิ์ ใช้คุมการทำงานในระบบ |
| Position | หน้าที่จริง ใช้แสดงผลในทีมโปรเจค |

Project Role ที่ใช้ในระบบ

- Project Owner
- Project Manager
- Member
- Finance
- Viewer
Project Position เป็น Master Data มีรายการตั้งต้นและสามารถเพิ่มเองได้ โดยทุกค่าที่มีลักษณะเป็นรายการเลือกควรทำเป็น Master Data ไม่ fix เป็น text ตายตัว

### 1.3 Project Timeline

Project Timeline เป็นศูนย์กลางของวันสำคัญทั้งหมดในโปรเจค

| มุมมอง / หมวดหมู่ | ภาษาไทย |
| --- | --- |
| All | ทั้งหมด |
| Milestones | หมุดหมาย |
| Deliverables | งวดงาน / สิ่งส่งมอบ |
| Meetings / Appointments | ประชุม / นัดหมาย |
| Maintenance | MA / ดูแลหลังส่งมอบ |
| Other | อื่น ๆ |

Timeline Item ใช้ Field กลางร่วมกัน

- Title
- Type
- Planned Date
- Actual Date
- Responsible Person
- Note
- Reminder Option
- Related Link / Reference
- Timeline Item ไม่มี Status ให้ผู้ใช้เลือกใน MVP
- ผู้ใช้ Add / Edit / Delete ได้
- ระบบแสดง Indicator เอง เช่น ใกล้ถึงกำหนด เลยกำหนด หรือมีวันที่เกิดขึ้นจริงแล้ว
- ถ้าไม่มีข้อมูลเชื่อมโยง Delete ได้จริง
- ถ้ามีข้อมูลเชื่อมโยง ห้าม Delete จริงทันที ต้อง Cancel / Hide / Archive / Remove Link ตามกรณี
### 1.4 Project Basic Info

ตอนสร้าง Project ให้บังคับข้อมูลน้อยที่สุด

| ประเภทข้อมูล | รายการ |
| --- | --- |
| Required | Project Name, Project Type |
| Auto Set | Project Status = Draft, Created By, Created At, Project Owner เริ่มต้น = คนสร้าง, Priority = Medium |
| Optional | Project Description, Priority, Client / Requesting Unit, Scope / Objective |

Project Code Policy

- Project Code มี Policy ระดับ Workspace
- Mode: Auto Generate หรือ Manual Input
- Default = Auto Generate
หลักการ: หลังสร้าง Project แล้วสามารถเติมข้อมูลละเอียดเพิ่มได้ และอนาคตสามารถเพิ่ม field, section, template, workflow ได้โดยไม่รื้อโครงหลัก

## 2. Project Detail Layout

Project Detail ใช้โครงแบบ Header + Tabs

| ส่วน | ข้อมูล / รายการ |
| --- | --- |
| Project Header | Project Name, Project Code, Project Type, Project Status, Priority, Project Owner, Quick Timeline, Action Button |
| Project Tabs | Overview, Tasks, Finance, Documents, Team, Timeline, Announcements, Activity, Settings |

ภาษาไทยของ Tabs: ภาพรวม, งาน, การเงิน, เอกสาร, ทีมโปรเจค, ไทม์ไลน์, ประกาศ, กิจกรรม, ตั้งค่า

หมายเหตุ: Deliverables ไม่แยกเป็นแท็บหลัก แต่เป็น Timeline Type เพื่อไม่ให้ผู้ใช้งานสับสน

## 3. Project Team & Position

ข้อมูลใน Team

- Member
- Project Role
- Project Position
- Join Date
- Status

| Project Member Status | ภาษาไทย | ความหมาย |
| --- | --- | --- |
| Active | อยู่ในโปรเจค | ยังเป็นสมาชิกในโปรเจคและใช้งานได้ |
| Removed | ถูกนำออกจากโปรเจค | ถูกนำออก แต่ยังเก็บประวัติไว้ |

- ถ้ายังไม่มี activity / task / comment / file / ข้อมูลอ้างอิง สามารถลบออกจากโปรเจคได้จริง
- ถ้าเคยมีข้อมูลอ้างอิงแล้ว ห้ามลบจริง ให้เปลี่ยนเป็น Removed
- ถ้าต้องนำกลับเข้าโปรเจค ให้เปลี่ยน Removed -> Active
- User Account Status แยกจาก Project Member Status
สิทธิ์จัดการ Project Team

- Project Owner จัดการสมาชิกในโปรเจคได้
- Project Manager จัดการสมาชิกทั่วไปได้ แต่ไม่ควรแก้ Project Owner
- Workspace Owner / Admin จัดการได้ตามสิทธิ์ระดับ Workspace
## 4. Project Task Structure

### 4.1 Assignee / Watcher

- Task มี Assignee หลัก 1 คน
- Task มี Watcher / Participant ได้หลายคน
- การส่งต่องานใช้ Reassign / Transfer Task
- ต้องเก็บประวัติว่าใครโอน โอนจากใคร โอนไปให้ใคร โอนเมื่อไหร่ และเหตุผล
### 4.2 Task Date

| Field | ภาษาไทย |
| --- | --- |
| Start Date | วันที่เริ่มงาน |
| Due Date | วันครบกำหนด |
| Completed Date | วันที่เสร็จจริง |

Reminder ยังไม่เป็น field บังคับของ Task แต่ Task ต้องรองรับการนำไปใช้กับระบบแจ้งเตือนในอนาคตโดยอ้างอิงจากวันที่ เช่น Due Date

### 4.3 Task Status / Workflow

Task Workflow ใช้แนวคิด Template + Copy to Project

ระบบมี seed workflow มาตรฐานให้

Workspace สามารถแก้ workflow template ของตัวเองได้

ตอนสร้าง Project ระบบ copy template ไปเป็น workflow ของ Project นั้น

หลังจากนั้น Project นั้นแก้ workflow ของตัวเองได้

การแก้ workflow ของ Project A ไม่กระทบ Project B

Seed ตั้งต้น

- Backlog
- To Do
- In Progress
- Review
- Testing
- Blocked
- Done
- Cancelled
- เพิ่มสถานะเองได้
- แก้ชื่อสถานะได้
- เรียงลำดับสถานะได้
- ลาก Task ย้ายสถานะได้
- ซ่อน / ปิดใช้สถานะที่ไม่ใช้แล้วได้
- ห้ามลบสถานะที่มี Task ใช้งานอยู่
### 4.4 Task Type

Task Type เป็น Master Data ระดับ Workspace

| Seed Task Type | ภาษาไทย |
| --- | --- |
| Task | งานทั่วไป |
| Bug / Issue | ปัญหา / บั๊ก |
| Change Request | คำขอเปลี่ยนแปลง |
| Support | งานซัพพอร์ต |

- Workspace เพิ่ม / แก้ไข / ปิดใช้งาน Task Type ได้
- ทุก Project ใน Workspace ใช้ Task Type ชุดเดียวกัน
- ห้ามลบ Task Type ที่มี Task ใช้งานอยู่
### 4.5 Task Detail Extension

Task รองรับส่วนเสริมพื้นฐาน

- Checklist
- Comment
- Attachment
Attachment รองรับไฟล์ รูปภาพ และลิงก์ โดยไฟล์ที่แนบกับ Task ต้องอยู่ภายใต้ Project File Center ไม่ใช่ไฟล์ลอย ๆ แยกจากโปรเจค

อนาคตสามารถเพิ่ม Time Log, Approval, Subtask, Dependency, Related Task, Work Log, QA Result, Customer Feedback

### 4.6 Task Relation

Task Relation เป็นการเชื่อมโยงเพื่อให้เห็นความเกี่ยวข้อง / ความสำคัญ ไม่ใช่การบังคับ workflow

- เชื่อมกับ Deliverable / งวดงาน ได้
- เชื่อมกับ Related Task / งานที่เกี่ยวข้อง ได้
- เชื่อมได้แต่ไม่บังคับ ใช้เพื่อให้เห็นบริบทของงาน
## 5. Timeline + Deliverable Structure

Timeline เป็นศูนย์กลางของวันสำคัญทั้งหมด และ Deliverable เป็น Timeline Item Type หนึ่ง

โครงสร้าง: Project Timeline > Timeline Item > Type = Deliverable > Delivery Attempt

| Timeline Type ตั้งต้น | ภาษาไทย |
| --- | --- |
| Milestone | หมุดหมาย |
| Deliverable | งวดงาน / สิ่งส่งมอบ |
| Meeting / Appointment | ประชุม / นัดหมาย |
| Maintenance | MA / ดูแลหลังส่งมอบ |
| Contract | สัญญา |
| Other | อื่น ๆ |

Meeting / Appointment ใช้สำหรับนัดคุยงาน นัดเก็บข้อมูล นัดประชุมทีม นัดประชุมลูกค้า นัดตรวจงาน หรือนัดส่งเอกสาร

### Delivery Attempt

ใช้เฉพาะ Timeline Item Type = Deliverable

- Attempt No.
- Submitted Date
- Submitted By
- Result
- Reviewed Date
- Reviewed By
- Remark
- Attachments / Documents

| Result | ภาษาไทย |
| --- | --- |
| Pending Review | รอตรวจรับ |
| Accepted | ผ่าน |
| Rejected | ไม่ผ่าน / ตีกลับ |
| Accepted with Condition | ผ่านแบบมีเงื่อนไข |

UI ของ Deliverable ใช้ Parent / Child: Parent = Timeline Item Type: Deliverable, Child = Delivery Attempt แต่ละรอบ และควรแสดงแบบ Card / Accordion / Expandable List

### Deliverable Summary Status

ระบบคำนวณจาก Delivery Attempt ล่าสุด ผู้ใช้ไม่ต้องเลือกเอง

- Not Submitted
- Pending Review
- Accepted
- Rejected
- Accepted with Condition
- Cancelled
Delivery Timing เป็น indicator ไม่ใช่สถานะหลัก: On Time, Early, Late, Overdue

MVP-0 ยังไม่ทำ Deliverable เต็มรูปแบบ และ MVP-1 ยังไม่เชื่อม Deliverable กับ Finance โดยตรง หากต้องเชื่อมในอนาคตให้ใช้ Reference Tracking / relation layer แทนการผูก schema แข็งตั้งแต่แรก

## 6. Project Finance Structure

- Overview
- Budget
- Expenses
- Penalty / Adjustment
- Finance Visibility
MVP ใช้เพื่อบริหารการเงินระดับโปรเจค ยังไม่ใช่ระบบบัญชีเต็มรูปแบบ

Finance Entry รองรับค่าใช้จ่าย ค่าปรับ ส่วนลด รายการปรับยอด และรายการอื่น ๆ

Finance Category / Finance Type ต้องเป็น Master Data

ยังไม่ทำใน MVP-0: Finance เต็มรูปแบบ

ยังไม่ทำใน MVP-1: Invoice, Payment Tracking, ภาษี, หัก ณ ที่จ่าย, บัญชีลูกหนี้ / เจ้าหนี้, P/L เต็มระบบ, เชื่อม Deliverable กับ Finance โดยตรง

## 7. Project Document Structure

Project Document = Project File Center

- รองรับ Folder
- รองรับ Subfolder
- รองรับ File
- รองรับ External Link
Folder / Subfolder

- รองรับหลายชั้น
- Max Folder Depth = 5 ชั้น
- สร้าง / Rename / Move Folder ได้
- ย้าย Folder / File ได้ภายใน Project เดียวกัน
- MVP ยังไม่รองรับย้ายข้าม Project
การลบ File / Folder

- ถ้าไม่มีข้อมูลอ้างอิง Delete ได้จริง
- ถ้ามีข้อมูลอ้างอิง ห้าม Delete จริง ให้ Archive / Hide / Remove Link
ต้องมี Reference Tracking กลาง เพื่อเก็บว่าข้อมูลถูกนำไปใช้ที่ไหน เริ่มใช้กับ File / Document, Task, Timeline Item, Finance Entry, Announcement

MVP ยังไม่ทำ File Version Control โดยให้ upload file เป็นไฟล์ใหม่ ตั้งชื่อไฟล์แยกเวอร์ชันเอง และใช้ Folder ช่วยจัดระเบียบ เช่น draft / final

## 8. Project Announcement & Activity

### 8.1 Announcement

Announcement ไม่ใช่ Chat ใช้สำหรับแจ้งเรื่องสำคัญของโปรเจค

| Role | สิทธิ์ประกาศ |
| --- | --- |
| Project Owner | สร้าง / แก้ไข / ปักหมุด / ซ่อน / ลบประกาศได้ |
| Project Manager | สร้าง / แก้ไข / ปักหมุด / ซ่อนประกาศได้ |
| Member | สร้างประกาศได้ และแก้ / ลบได้เฉพาะประกาศของตัวเอง |
| Finance | สร้างประกาศได้ และแก้ / ลบได้เฉพาะประกาศของตัวเอง |
| Viewer | อ่านประกาศได้อย่างเดียว |

ประเภทประกาศ: Normal Announcement, Pinned Announcement, Expiring Announcement

ประกาศหมดอายุแล้วไม่ลบจริง แต่ซ่อนจากหน้าประกาศหลัก และยังดูย้อนหลังได้

### 8.2 Activity

Activity = ระบบบันทึกประวัติ ไม่ใช่ที่ให้ผู้ใช้โพสต์ข้อความเอง

แสดงเป็น Timeline ใน Project Detail แต่หลังบ้านต้องเก็บแบบ Audit-ready

- Actor
- Action
- Resource Type
- Resource ID
- Old Value
- New Value
- Timestamp
- IP Address
- User Agent
- Result
- Tenant ID
- Project ID
## 9. Project Settings

Project Settings MVP มีเฉพาะค่าควบคุมพฤติกรรมของโปรเจค ไม่ใช่ที่แก้ข้อมูลทั่วไปของ Project Profile

- Task Workflow
- Finance Visibility
- Archive / Danger Zone
ไม่ทำใน MVP

- Project Access / Visibility: ยังไม่ทำเป็น setting แยก ใช้ Workspace Role + Project Member ไปก่อน
- Timeline Type Config: ยังไม่ทำราย Project ใช้ Type ตั้งต้น / Master ระดับ Workspace ไปก่อน
## สรุปสถานะชุดที่ 3

- Project Profile
- Project Detail Layout
- Project Team & Position
- Project Task Structure
- Timeline + Deliverable Structure
- Project Finance Structure
- Project Document Structure
- Project Announcement & Activity
- Project Settings
