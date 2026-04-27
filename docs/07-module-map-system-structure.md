# ชุดที่ 7: Module Map / System Structure

เอกสารออกแบบภาพรวม Module และโครงสร้างระบบจากมุม User Journey

| รายการ | รายละเอียด |
| --- | --- |
| สถานะเอกสาร | Confirmed / ใช้ต่อจากชุดที่ 1-6 |
| เป้าหมาย | ระบุการเดินทางของผู้ใช้งาน, Module หลัก, ระดับของ Module และ Shared Services เพื่อเตรียมต่อยอดสู่ Database Design และ API Design |
| ขอบเขต | เอกสารนี้ยังไม่ลงรายละเอียด Database, API, UI รายหน้า, Docker หรือ Code Implementation |

## 0. Roadmap Control

| ชุดเอกสาร | หัวข้อ | สถานะ |
| --- | --- | --- |
| ชุดที่ 1 | System Overview / Scope | จัดทำแล้ว |
| ชุดที่ 2 | Workspace / Registration Flow | จัดทำแล้ว |
| ชุดที่ 3 | Project Structure / Project Detail Concept | จัดทำแล้ว |
| ชุดที่ 4 | Notification / Calendar / Reminder | จัดทำแล้ว |
| ชุดที่ 5 | Master Data / Config / Permission | จัดทำแล้ว |
| ชุดที่ 6 | Tenant / Data Isolation | จัดทำแล้ว |
| ชุดที่ 7 | Module Map / System Structure | เอกสารนี้ |
| ชุดที่ 8 | Database Design | ทำถัดไป |
| ชุดที่ 9 | API Design | ทำภายหลัง |
| ชุดที่ 10 | UI / Page Structure | ทำภายหลัง |
| ชุดที่ 11 | Docker / Deploy / Infra | ทำภายหลัง |
| ชุดที่ 12 | AI Prompt / Code Generation | ทำภายหลัง |

หลักการควบคุมขอบเขต: ชุดที่ 7 ใช้อธิบายภาพรวมการใช้งานและ Module Map เท่านั้น ยังไม่ออกแบบฐานข้อมูล, API, UI รายหน้า, Docker หรือ Prompt เขียนโค้ด

## 1. Objective ของชุดที่ 7

- อธิบายระบบจากมุม User Journey ก่อน แล้วค่อย map เป็น Module
- ระบุว่า Module ไหนอยู่ระดับ App / Platform, Workspace, Project หรือ Shared Service
- ระบุหน้าที่ของแต่ละ Module โดยไม่ลงรายละเอียดเชิงเทคนิคเกินขอบเขต
- คุม MVP Scope เพื่อไม่ให้ระบบบวมเกินจำเป็น
- เตรียมข้อมูลให้ชุดถัดไปสามารถออกแบบ Database และ API ได้เป็นระบบ
## 2. User Journey หลักของระบบ

1. ผู้ใช้เข้า Public App / Landing เพื่ออ่านข้อมูลระบบและสนใจใช้งาน

2. ผู้ใช้สมัคร Workspace และยืนยัน Email

3. ผู้ใช้ Login เข้าระดับ Workspace ผ่าน subdomain หรือ path dev mode

4. ผู้ใช้เข้าสู่ Workspace Portal / Workspace Admin Panel

5. ผู้ใช้เข้า Project เพื่อจัดการงานจริง

6. Owner / Admin จัดการทีมงาน และทีมงาน login เข้าสู่ Workspace / Project

7. ผู้บริหาร / ผู้จัดการดู Report / Dashboard ตามสิทธิ์

8. ผู้ดูแลจัดการ Settings, Support, Master Data และ Config ที่จำเป็น

## 3. Module Level

| ระดับ | ความหมาย | ตัวอย่าง |
| --- | --- | --- |
| App / Platform Level | ระบบแม่ก่อนเข้า Workspace และส่วนดูแลระบบกลาง | Public App, Platform Admin, User Account, Workspace Management |
| Workspace Level | พื้นที่กลางหลัง login เข้าสู่ Workspace ใช้บริหารหลาย Project | Workspace Portal, Member, Master Data, Config, Reports |
| Project Level | พื้นที่ทำงานจริงของแต่ละ Project | Project, Task, Timeline, Finance, Document, Announcement |
| Shared / Cross-cutting Services | ระบบกลางที่หลาย Module ใช้ร่วมกัน | Tenant Context, Permission Guard, Activity Log, Reference Tracking |

## 4. Public App / Landing

หน้าที่: หน้าแรกของระบบก่อนผู้ใช้เข้าสู่ Workspace ใช้ประชาสัมพันธ์ระบบ อธิบายว่า App คืออะไร และพาผู้ใช้ไปสมัครหรือเข้าสู่ Workspace

- Hero Section
- App Overview
- Feature Highlight
- Use Case / เหมาะกับใคร
- Create Workspace CTA
- Workspace Login / Find Workspace
- Contact Us
- FAQ เบื้องต้น
ยังไม่ทำใน MVP: Pricing เต็มรูปแบบ, Blog / Content Marketing, Customer Case Study, Online Payment, Chat Support สด

## 5. Workspace Registration

หน้าที่: ขั้นตอนสมัครและสร้างพื้นที่ทำงานใหม่

1. ผู้ใช้สมัคร Workspace

2. ระบบสร้าง Owner Account + Workspace

3. Account อยู่สถานะ Pending Verification

4. ระบบส่ง Email ยืนยัน

5. ผู้ใช้กดยืนยัน Email

6. Account เป็น Active

7. Workspace เข้าใช้งานได้ตาม Auto Approve

ข้อมูลขั้นต่ำตอนสมัคร: Workspace Name, Workspace Slug, Owner Name, Owner Email, Password, Contact Email

เหตุผลที่ต้องมี Email Verification: ป้องกัน spam account, ลด workspace ขยะ, ลด email ปลอม, ช่วย cleanup demo data และประหยัดพื้นที่ระบบ

Cleanup fields: email_verified_at, last_login_at, last_activity_at, created_at

## 6. Workspace Login / Workspace Entry

หน้าที่: จุดที่ผู้ใช้เข้าสู่ Workspace ของตัวเอง

| Mode | URL ตัวอย่าง | หลักการ |
| --- | --- | --- |
| Production | https://company-a-demo.yourdomain.com/login | subdomain -> slug -> tenant_id |
| Development หลัก | http://localhost:3000/w/company-a-demo/login | path /w/:slug -> tenant_id |
| Development test subdomain | http://company-a-demo.localhost:3000/login หรือ http://company-a-demo.local.test/login | ใช้ทดสอบ behavior ให้ใกล้ production |

- Login ต้องผ่าน Tenant Resolution
- ต้องตรวจ email verified
- ต้องตรวจ Workspace Membership
- ถ้า user มีหลาย Workspace ให้เลือกจากหน้า app หลักได้
## 7. Workspace Portal / Workspace Admin Panel

Decision: Workspace หลัง Login ไม่ใช่แค่ Dashboard ส่วนตัว แต่เป็น Workspace Portal / Workspace Admin Panel ที่แสดงเมนูและข้อมูลตาม Role / Permission

- Dashboard
- Projects
- My Tasks
- Calendar / Timeline
- Team / Members
- Documents Overview
- Reports
- Notifications
- Announcements
- Workspace Settings

| กลุ่มสิทธิ์ | มุมมอง |
| --- | --- |
| Owner / Admin | เห็นและจัดการได้เกือบทั้งหมดใน Workspace เช่น ทุก Project, สมาชิก, Master Data, Config, Usage, Settings |
| Executive / Manager | เห็นภาพรวมหลาย Project เพื่อดูสถานะและรายงาน เช่น Project Summary, Task Summary, Timeline Summary, Finance Summary ตามสิทธิ์ |
| User / Project Member | เห็นเฉพาะ Project / งานที่ตัวเองเกี่ยวข้อง เช่น My Projects, My Tasks, Timeline และ Announcement ที่เกี่ยวข้อง |

## 8. Project Management Entry

Decision: Project เป็นพื้นที่ทำงานจริงของแต่ละโปรเจค ผู้ใช้เข้า Project จาก Workspace Portal ได้หลายทาง

- เข้าได้จากหน้า Projects
- Dashboard
- My Tasks
- Calendar / Timeline
- Notification
- Report
เมื่อเข้า Project ต้องเห็น: Project Header + Project Tabs

| ส่วน | ข้อมูล / แท็บ |
| --- | --- |
| Project Header | Project Name, Project Code, Project Type, Project Status, Priority, Project Owner, Timeline สำคัญใกล้ถึง, Action ตามสิทธิ์ |
| Project Tabs | Overview, Tasks, Finance, Documents, Team, Timeline, Announcements, Activity, Settings |

## 9. Team / Member Management

ต้องแยก 2 ระดับ: Workspace Member และ Project Member

| ระดับ | หน้าที่ | Role |
| --- | --- | --- |
| Workspace Member | เชิญสมาชิก, สร้างสมาชิกให้, กำหนด Workspace Role, ระงับ/นำออกจาก Workspace, ดูสถานะบัญชี | Owner, Admin, Executive, User |
| Project Member | เพิ่มคนเข้า Project, กำหนด Project Role, กำหนด Project Position, Active/Removed ใน Project | Project Owner, Project Manager, Member, Finance, Viewer |

หลักสำคัญ: User Account Status ≠ Workspace Member Status ≠ Project Member Status

### 9.1 User Account Status และพนักงานลาออก

| Status | ภาษาไทย | ความหมาย |
| --- | --- | --- |
| Active | ใช้งานอยู่ | login ได้ และเพิ่มเข้า Workspace / Project ได้ |
| Suspended | ระงับชั่วคราว | login ไม่ได้ชั่วคราว ข้อมูลเดิมยังอยู่ และเปิดกลับมาได้ |
| Disabled | ปิดใช้งานแล้ว | login ไม่ได้แล้ว ไม่ควรถูกดึงเข้า Project ใหม่ ข้อมูลเดิมยังอยู่ |

| Status Reason | ภาษาไทย |
| --- | --- |
| Resigned | ลาออก |
| Account Restricted | จำกัดการใช้งานบัญชี |
| Other | อื่น ๆ |

- Status ใช้ควบคุมการเข้าใช้งานระบบ
- Status Reason ใช้อธิบายเหตุผลเพิ่มเติมแบบกลาง ๆ
- ปิดบัญชีผู้ใช้ไม่ใช่การลบประวัติการทำงาน
- ถ้าคนลาออกแล้วยังมี Task ค้าง ระบบควรให้ Owner / Manager เห็นรายการเพื่อ Reassign
### 9.2 Membership History / Rejoin

- การลบ / ยกเลิกสมาชิก ไม่ใช่การลบประวัติ
- Workspace Membership = Removed และ Project Membership = Removed ได้ แต่ Activity Log และข้อมูลที่เคยทำยังอยู่
- ถ้ากลับมาใหม่ แนวทาง MVP ใช้ Removed -> Active แต่ต้องมี Audit Log ชัดเจน
- ระบบควรแสดงข้อความว่า ผู้ใช้นี้เคยเป็นสมาชิกมาก่อน และระบบยังคงประวัติเดิมไว้เพื่อการตรวจสอบ
## 10. Report / Dashboard

Decision: Report / Dashboard MVP ใช้เพื่อดูภาพรวมของ Workspace และ Project โดยมีรายงานพื้นฐาน ยังไม่ทำระบบ BI หรือ Custom Report Builder

- Project Summary
- Task Summary
- Timeline / Overdue Summary
- Finance Summary
- Usage / Storage Summary

| กลุ่มสิทธิ์ | การมองเห็นรายงาน |
| --- | --- |
| Owner / Admin / Executive | ดูภาพรวมข้าม Project ได้ |
| Project Owner / Project Manager | ดูเฉพาะ Project ที่เกี่ยวข้อง |
| User | ดูเฉพาะงาน / Project ของตัวเอง |
| Finance | ดู Finance ตามสิทธิ์ที่กำหนด |

ยังไม่ทำใน MVP: BI Dashboard เต็มรูปแบบ, Custom Report Builder, Export Excel รายงานซับซ้อน, กราฟ drill-down ลึก, KPI configurable ขั้นสูง

## 11. Admin / Settings / Support

Decision: MVP มี Platform Admin สำหรับดูแลระบบกลาง มี Workspace Settings สำหรับ Owner/Admin และมี Contact Support แบบเบา ๆ

| ส่วน | สิ่งที่มีใน MVP |
| --- | --- |
| Platform Admin | Workspace Management, Workspace Status / Lifecycle, Usage / Storage Overview, Platform Config ที่จำเป็น, Reference Data เบื้องต้น, Cleanup / Hard Delete, Audit Log |
| Workspace Admin / Settings | Workspace Profile, Workspace Members, Workspace Master Data, Workspace Config, Project Code Policy, Task Workflow Template, Document Folder Template, Finance Type / Category, Notification Preference เบื้องต้น |
| Support | Contact Support, Email / Line Official / Contact Form, FAQ |

ยังไม่ทำใน MVP: Billing เต็มระบบ, Support Ticket เต็มระบบ, Advanced Analytics, Role Permission Builder แบบละเอียด, Workflow Automation, Custom Field Builder เต็มรูปแบบ

## 12. Shared Services Map

Decision: Shared Services ต้องเป็นแกนกลาง ไม่กระจาย logic ซ้ำในแต่ละ Module

| Shared Service | หน้าที่ |
| --- | --- |
| Tenant Context | รู้ว่า request นี้อยู่ Workspace ไหน |
| Permission Guard | ตรวจว่า user มีสิทธิ์ทำ action นี้ไหม |
| Activity Log / Audit | เก็บประวัติว่าใครทำอะไร เมื่อไหร่ |
| Reference Tracking | เก็บว่าข้อมูลชิ้นนี้ถูกเอาไปใช้ที่ไหน |
| Notification Engine | สร้าง In-app / Email Notification จาก event สำคัญ |
| File Lifecycle / Cleanup | จัดการ soft delete, replaced file, orphan file, cleanup MinIO |
| Master / Config Engine | อ่านค่า master/config ตามระดับ App / Workspace / Project |

- การลบไฟล์ต้องเช็ค Reference Tracking ก่อน
- การเปิด Project ต้องผ่าน Tenant Context + Permission Guard
- การแก้ข้อมูลสำคัญต้องสร้าง Activity Log
- การแจ้งเตือนต้องวิ่งผ่าน Notification Engine
## 13. Summary: Module Map ตามระดับ

| ระดับ | Module หลัก |
| --- | --- |
| App / Platform | Public App / Landing, Platform Login, Platform Admin, User Account, Workspace Management, App Master Data, Platform Config, Reference Data, Platform Audit, File Storage Management |
| Workspace | Workspace Login, Workspace Portal, Workspace Profile, Workspace Members, Workspace Role / Permission, Workspace Master Data, Workspace Config, Usage Limit, Notification Preference, Reports |
| Project | Project, Project Team, Task, Timeline / Deliverable / Meeting, Finance, Document / File Center, Announcement, Notification, Calendar, Project Settings |
| Shared Services | Tenant Context, Permission Guard, Activity Log / Audit, Reference Tracking, Notification Engine, File Lifecycle / Cleanup, Master / Config Engine |

## 14. MVP Scope และ Future Scope

| Phase | Scope |
| --- | --- |
| MVP-0 | Public App แบบเบา, Workspace Registration, Email Verification, Workspace Login, Tenant Resolution, Workspace Portal เบื้องต้น, Project Management, Team / Member, Task Board, Basic File Upload, Basic Activity Log, Shared Services สำคัญ |
| MVP-1 | Timeline / Deliverable, Finance เบื้องต้น, Notification / Announcement, Calendar, Basic Reports, Platform Admin เบื้องต้น |
| Future / ยังไม่เน้นใน MVP | Billing เต็มระบบ, Support Ticket เต็มระบบ, Advanced Analytics, Custom Report Builder, Role Permission Builder, Workflow Automation, Custom Field Builder, File Version Control, Calendar Sync, External Chat/Notification Integration |

## 15. ข้อสรุปสำหรับชุดที่ 7

- ระบบจะอธิบายจากมุม User Journey ก่อน แล้ว map เป็น Module
- Workspace Portal เป็นศูนย์กลางหลัง Login ไม่ใช่ Dashboard ส่วนตัวอย่างเดียว
- Project เป็นพื้นที่ทำงานจริงของแต่ละโปรเจค
- Team / Member ต้องแยก User Account, Workspace Member และ Project Member ชัดเจน
- Shared Services เป็นแกนกลางด้าน security, audit, reference, notification และ file lifecycle
- ชุดถัดไปควรเป็นชุดที่ 8: Database Design โดยอิง Module Map ชุดนี้
