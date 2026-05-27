# ชุดที่ 5
Master Data / Configuration / Permission Concept

Project Management Platform - System Design

> **Status: Forward-looking design intent (MVP-1+).** Permission Guard ที่ implement จริงใน MVP-0 อยู่ที่ `internal/modules/workspace/workspaceperm` (role-based map สำหรับ `workspace.view`/`workspace.manage`). Master Data และ Configuration UI/API ยังไม่ทำ. เอกสารนี้คือ design intent ที่กว้างกว่า.

เอกสารนี้สรุปแนวทาง Master Data, Configuration, Permission, Change Impact, Seed Data และ Audit Log โดยเน้นให้ระบบตรวจสอบได้ ปลอดภัย และไม่บวมเกินจำเป็นใน MVP

## 1. หลักการออกแบบของชุดที่ 5

- ออกแบบให้ระบบขยายได้ในอนาคต แต่ MVP ต้องไม่บวมเกินจำเป็น
- ค่าที่ผู้ใช้แก้บ่อยควรมีหน้า CRUD เฉพาะที่จำเป็น
- ค่าที่เปลี่ยนน้อยหรือกระทบ logic หลักให้ใช้ seed / backend admin ก่อน
- ส่วนที่เกี่ยวกับ audit, permission, security และ change impact ต้องมีตั้งแต่แรก
- ข้อมูลที่ใช้ตรวจสอบย้อนหลังต้องไม่หายเมื่อมีการแก้ master/config หรือเปลี่ยนโครงสร้างภายหลัง
## 2. Master Data Level

Master Data แบ่งเป็น 3 ระดับ เพื่อแยกความรับผิดชอบและผลกระทบของข้อมูลให้ชัดเจน

| ระดับ | ความหมาย | ผู้ดูแล | ตัวอย่าง |
| --- | --- | --- | --- |
| App Master | ค่าแม่แบบกลางของระบบ ใช้เป็น seed/default ให้ Workspace | Platform Admin | Default Project Role, Workspace Role, Project Status, Timeline Type, Priority |
| Workspace Master | ค่ามาตรฐานของ Workspace นั้น ใช้ร่วมกับทุก Project ใน Workspace | Workspace Owner / Admin | Project Position, Task Type, Finance Category, Document Folder Template, Task Workflow Template |
| Project Master | ค่าที่ใช้เฉพาะ Project นั้น | Project Owner / Manager ตามสิทธิ์ | Project Task Workflow, Finance Visibility ของ Project |

หลักการ MVP: ส่วนใหญ่ให้ปรับที่ระดับ Workspace ส่วนระดับ Project ใช้เฉพาะสิ่งที่จำเป็นจริง เช่น Task Workflow และ Finance Visibility

## 3. Master Data Inventory และขอบเขต CRUD ใน MVP

เพื่อกันระบบบวม MVP จะทำหน้า CRUD เฉพาะ Master Data ที่ผู้ใช้มีโอกาสแก้บ่อย ส่วนค่าระบบที่เปลี่ยนน้อยให้ใช้ seed หรือ backend/admin tool ก่อน

| กลุ่ม | รายการ | แนวทาง MVP | เหตุผล |
| --- | --- | --- | --- |
| ทำ CRUD ใน MVP | Project Position | มีหน้า CRUD ระดับ Workspace | แต่ละองค์กรเรียกตำแหน่งไม่เหมือนกัน |
| ทำ CRUD ใน MVP | Task Type | มีหน้า CRUD ระดับ Workspace | แต่ละทีมอาจเพิ่มประเภทงานเอง |
| ทำ CRUD ใน MVP | Task Workflow Template | มีหน้า CRUD ระดับ Workspace | ใช้เป็นแม่แบบก่อนสร้าง Project |
| ทำ CRUD ใน MVP | Finance Category / Finance Type | มีหน้า CRUD ระดับ Workspace | ต้องยืดหยุ่นกับรายการการเงิน |
| ทำ CRUD ใน MVP | Document Folder Template | มีหน้า CRUD ระดับ Workspace | แต่ละองค์กรอาจต้องการ folder เริ่มต้นต่างกัน |
| Seed ก่อน | Project Status / Project Role / Workspace Role | ยังไม่ทำหน้า CRUD | เป็นค่าระบบ กระทบ logic หลัก |
| Seed ก่อน | Timeline Type / Priority / Workspace Type | ยังไม่ทำหน้า CRUD | เปลี่ยนน้อย และยังไม่จำเป็นใน MVP |
| Seed ก่อน | Announcement Type / Notification Event Type | ยังไม่ทำหน้า CRUD | ลดความซับซ้อนของระบบแจ้งเตือน |

## 4. Reference Data ประเทศไทย

ข้อมูลที่อยู่ไทยควรมีในระบบเพื่อช่วยกรอกข้อมูล แต่ต้องไม่ทำให้ลูกค้าติดงานหากฐานข้อมูลไม่ครบหรือไม่ถูกต้อง

| หัวข้อ | Decision |
| --- | --- |
| แนวคิด | Autocomplete + Free Text |
| การใช้งาน | ใช้ฐานข้อมูลกลางช่วยแนะนำ แต่ไม่บังคับให้เลือกจาก master เท่านั้น |
| ข้อมูลที่เก็บจริง | เก็บเป็น string snapshot เช่น country_name, province_name, district_name, subdistrict_name, postal_code, address_text |
| Reference ID | เก็บ optional ref_id ได้ ถ้าผู้ใช้เลือกจาก autocomplete |
| กรณีกรอกเอง | เก็บ string อย่างเดียว และ ref_id เป็น null ได้ |
| CRUD | MVP ยังไม่ทำหน้า CRUD เต็ม หากต้องแก้ให้ Platform Admin จัดการผ่าน backend/admin tool ก่อน |

เหตุผล: ลูกค้าไม่ต้องรอทีมงานแก้ master data, ข้อมูลเก่าไม่พังเมื่อ master เปลี่ยน, และอนาคตสามารถทำ data cleansing/mapping ได้

## 5. Configuration Scope

Configuration คือค่าควบคุมพฤติกรรมของระบบ ต่างจาก Master Data ที่เป็นรายการให้เลือก

| ประเภท Config | เก็บที่ไหน | ตัวอย่าง | หลักการ |
| --- | --- | --- | --- |
| Environment Config | .env | APP_DOMAIN, APP_PROTOCOL, WILDCARD_DOMAIN, SSL_PROVIDER, TRAEFIK_ENTRYPOINT, MINIO_ENDPOINT, DATABASE_URL, REDIS_URL | เป็นค่า infra/runtime ไม่ควรแก้ผ่าน UI ส่วนมากต้อง restart/redeploy |
| Database Config | Database | Workspace Approval Mode, Project Code Policy, Usage Limit, Demo Marker, Notification Timing, Folder Max Depth, Pending Deletion Period | เป็นกติกาธุรกิจ แก้ผ่าน Platform/Workspace Admin ตามสิทธิ์ได้ ไม่ต้อง redeploy |

## 6. Config Level

| ระดับ | ใครแก้ได้ | รายการสำคัญ |
| --- | --- | --- |
| Platform Config | Platform Admin | Workspace Approval Mode, Default Usage Limit, Demo Marker Policy, Pending Deletion Period Default, Default Notification Timing, Reference Data Policy |
| Workspace Config | Workspace Owner / Admin | Project Code Policy, Task Workflow Template, Finance Visibility Default, Document Folder Template, Workspace Usage Limit Override, Notification Preference Default |
| Project Config | Project Owner / Manager ตามสิทธิ์ และ Workspace Owner/Admin | Project Task Workflow, Finance Visibility ของ Project |

หลักการ: Platform = ค่ากลางระบบ, Workspace = ค่าที่ลูกค้าปรับเอง, Project = ใช้เฉพาะเรื่องที่จำเป็นเพื่อลดระบบบวม

## 7. Permission Concept

| ระดับข้อมูล | สิทธิ์แก้ไข |
| --- | --- |
| App Master / Platform Config | Platform Admin |
| Workspace Master / Workspace Config | Workspace Owner / Workspace Admin |
| Project Master / Project Config | Project Owner / Project Manager เฉพาะ Project ที่มีสิทธิ์ และ Workspace Owner/Admin ตามสิทธิ์ |

- Project Manager แก้ได้เฉพาะ Project Config ของ Project ที่ตัวเองมีสิทธิ์ ไม่ควรแก้ Workspace Master
- การเปลี่ยน Master Data / Config สำคัญต้องมี Activity Log หรือ Audit Log เสมอ
## 8. Change Impact Rule

Master Data / Config ทุกตัวต้องระบุชนิดผลกระทบ เพื่อป้องกันข้อมูลเก่าเสียหรือเปลี่ยนโดยไม่ตั้งใจ

| ชนิดผลกระทบ | ความหมาย | ตัวอย่าง |
| --- | --- | --- |
| Snapshot Value | ข้อมูลเก่าไม่เปลี่ยนตาม master ใหม่ | Project Code, Finance Entry Category Name, Address Text, Document Name ตอนแนบ, Task Type Name ตอนสร้าง |
| Live Reference | ข้อมูลเก่าแสดงค่าตาม master ล่าสุด | Project Position, Task Workflow Status Name, Timeline Type Name, Priority Name |

- ถ้ามีข้อมูลอ้างอิงอยู่ ห้าม Delete จริง ให้ Inactive / Deprecated แทน
- Config ต้องระบุให้ชัดว่าเปลี่ยนแล้วมีผลกับข้อมูลใหม่เท่านั้น หรือมีผลทันที
- การเปลี่ยนแปลงสำคัญต้องมี Audit Log พร้อมค่าเดิมและค่าใหม่
## 9. Seed Data / Default Template

Seed Data คือค่าเริ่มต้นที่ระบบเตรียมให้ เพื่อให้ Workspace เริ่มใช้งานได้ทันที โดยไม่ต้องตั้งค่าทุกอย่างเอง

- ระบบต้องมี seed data สำหรับค่าเริ่มต้น
- เมื่อสร้าง Workspace ให้ copy seed จาก App Master ไปเป็น Workspace Master
- App Master เปลี่ยนภายหลัง ไม่กระทบ Workspace เก่าอัตโนมัติ
- Workspace ใหม่ใช้ seed ล่าสุดจาก App Master
- Workspace เก่าใช้ Workspace Master ที่เคย copy ไปแล้ว เว้นแต่มี flow อัปเดตแยกในอนาคต
สิ่งที่ควรมี Seed: Project Role, Workspace Role, Project Status, Task Type, Task Workflow Template, Timeline Type, Priority, Finance Category/Type, Document Folder Template, Announcement Type, Workspace Type

## 10. Audit Log for Master Data / Config Change

การเปลี่ยน Master Data / Config สำคัญต้องตรวจสอบย้อนหลังได้

| ข้อมูลที่ต้องเก็บใน Audit Log |
| --- |
| ใครแก้ / แก้เมื่อไหร่ / แก้ข้อมูลประเภทไหน / แก้รายการไหน |
| ค่าเดิม / ค่าใหม่ / ระดับที่แก้ เช่น Platform, Workspace, Project |
| เหตุผลถ้ามี / IP Address / User Agent |

ตัวอย่างที่ต้องเก็บ Log: Project Code Policy, Usage Limit, Finance Visibility, Task Workflow, Task Type, Finance Category/Type, Document Folder Template, Pending Deletion Period, Notification Timing

## 11. ข้อเสนอเพื่อลดความบวมของ MVP

ส่วนนี้เป็นข้อเสนอเชิงขอบเขต: หากตอนพัฒนาจริงฟังก์ชันใดทำไม่ยาก สามารถพิจารณาทำเพิ่มได้ แต่ไม่ควรบังคับเป็น MVP core ตั้งแต่แรก

| รายการที่เสนอให้ลด / ยังไม่ทำ UI เต็มใน MVP | แนวทางแทนใน MVP | หมายเหตุ |
| --- | --- | --- |
| CRUD Master Data ทุกตัว | ทำ CRUD เฉพาะตัวที่แก้บ่อย | ลดหน้าจอ setting ที่ไม่จำเป็น |
| Reference Data CRUD เต็ม | ใช้ seed + backend/admin tool ก่อน | ยังคง autocomplete + free text เพื่อไม่ให้ลูกค้าติดงาน |
| Notification Event Type CRUD | ใช้ seed/config หลังบ้านก่อน | ระบบแจ้งเตือนยังไม่ควรซับซ้อน |
| Timeline Type CRUD ราย Project | ใช้ type ตั้งต้น / Workspace Master ก่อน | ลดความซ้ำซ้อนกับ Project Settings |
| Project-level Config หลายอย่าง | ทำเฉพาะ Task Workflow และ Finance Visibility | กัน Project Settings บวม |
| Rule/config UI ละเอียดทุกตัว | ใช้ database config หรือ backend admin tool ก่อน | คง audit/security ไว้ แต่ลด UI |

## 12. สรุป Decision ชุดที่ 5

- Master Data แบ่งเป็น 3 ระดับ: App Master, Workspace Master, Project Master
- Configuration แบ่งเป็น Environment Config และ Database Config
- Database Config แบ่งเป็น Platform, Workspace และ Project เฉพาะที่จำเป็น
- Permission อิงตามระดับข้อมูล: Platform Admin, Workspace Owner/Admin, Project Owner/Manager
- Master/Config ทุกตัวต้องมี Change Impact Rule: Snapshot Value หรือ Live Reference
- ข้อมูลที่ถูกใช้งานแล้วห้ามลบจริง ให้ Inactive / Deprecated แทน
- Seed Data ใช้เพื่อเริ่ม Workspace ได้ทันที และ App Master เปลี่ยนแล้วไม่กระทบ Workspace เก่าอัตโนมัติ
- Reference Data ประเทศไทยใช้ Autocomplete + Free Text และเก็บ string snapshot + optional ref_id
- การแก้ Master Data / Config สำคัญต้องมี Audit Log
- MVP ต้องลด UI ที่ไม่จำเป็น แต่ส่วน audit/security/change impact ต้องคงไว้
