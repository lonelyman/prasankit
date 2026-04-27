# ชุดที่ 10: UI / Page Structure

_Project / Workspace Management Platform_

เอกสารนี้เป็นชุดออกแบบโครงสร้างหน้าเว็บ เมนูหลัก Layout และการมองเห็นตามสิทธิ์ของระบบ Project / Workspace Management Platform

| หัวข้อ | รายละเอียด |
| --- | --- |
| Scope | UI / Page Structure เท่านั้น ไม่ลง Database, API, Docker, Deploy หรือ Code จริง |
| Goal | กำหนดโครงหน้าเว็บ เมนูหลัก Template และ UX Flow หลักของผู้ใช้ |
| MVP Focus | หน้าที่ใช้ทำงานจริง เช่น Workspace Portal, Project Detail, Task, Timeline, Document, Finance และ Team |
| Control | แยก Template ตาม Context เพื่อกันผู้ใช้สับสนระหว่าง App / Owner / Workspace / Project / Platform |

### 1. UI Structure Principle

UI แบ่งเป็น 5 กลุ่มหลักตาม Context การใช้งาน และแต่ละกลุ่มควรมี Visual Cue เช่น Header Label, Sidebar, Breadcrumb, Badge หรือ Accent Color เพื่อให้ผู้ใช้รู้ว่าตอนนี้อยู่ระดับไหนของระบบ

| Template | กลุ่มผู้ใช้ | วัตถุประสงค์ |
| --- | --- | --- |
| Public App Template | ผู้ใช้ทั่วไป / ผู้ยังไม่เข้า Workspace | แนะนำระบบ สมัคร Workspace Login Reset Password Contact / FAQ |
| Owner / Business Admin Template | ผู้สร้างระบบ / ทีมเจ้าของ App | ดูแลลูกค้า Workspace Demo Usage และ Cleanup แบบเบื้องต้น |
| Workspace Portal Template | Owner/Admin/Manager/User ของ Workspace | ศูนย์กลางทำงานหลัง Login เข้า Workspace |
| Project Detail Template | สมาชิก Project | พื้นที่ทำงานจริงของ Project ด้วย Project Header + Tabs |
| Platform Admin Template | Platform/System/Support Admin | ดูแลระบบเชิงเทคนิค Config Master Data Audit Security Cleanup Hard Delete |

หลักสำคัญ: เมนูและข้อมูลที่เห็นต้องขึ้นกับ Role / Permission และ Backend ต้องตรวจสิทธิ์ซ้ำเสมอ ไม่ใช้การซ่อนเมนูจาก Frontend อย่างเดียว

### 2. Public App UI

Public App UI คือหน้าสาธารณะของระบบ ใช้สำหรับคนทั่วไป สมัคร Workspace ยืนยัน Email เข้าสู่ Workspace และติดต่อทีมงาน

| Page | MVP Scope | หมายเหตุ |
| --- | --- | --- |
| Landing Page | แนะนำระบบแบบเบา ๆ มี Hero, Overview, Feature, Use Case, CTA, Contact/FAQ | ยังไม่ทำ Marketing Website เต็มรูปแบบ |
| Create Workspace Page | สร้าง Workspace + Owner Account, ตรวจ Slug, ส่ง Email Verification | ต้องมี PDPA / Privacy Notice และ Demo Notice |
| Verify Email Page | รองรับสำเร็จ, Token หมดอายุ, Token ไม่ถูกต้อง, ยืนยันไปแล้ว | มีทางไป Login / ขอส่ง Email ใหม่ / กลับหน้าแรก |
| Workspace Entry / Find Workspace | กรอก Workspace Slug เพื่อไป Login และค้นหาด้วย Email แบบปลอดภัย | ไม่เปิดเผยรายการ Workspace ต่อหน้าสาธารณะ |
| Login Page | ใช้ Auth กลาง แต่แยก Login Context | Workspace / Owner / Platform Admin Login |
| Forgot Password Page | กรอก Email เพื่อขอลิงก์ตั้งรหัสผ่านใหม่ | ข้อความต้องไม่เปิดเผยว่า Email มีอยู่จริงหรือไม่ |
| Reset Password Page | ตั้งรหัสผ่านใหม่จาก Token | รองรับ Token หมดอายุ / ไม่ถูกต้อง / ถูกใช้ไปแล้ว |
| Contact / FAQ Page | FAQ เบื้องต้น + ช่องทางติดต่อ | ยังไม่ทำ Ticket / Live Chat / Knowledge Base เต็มรูปแบบ |

#### 2.1 Create Workspace: PDPA และ Demo Notice

1. ต้องแจ้งว่าระบบจะเก็บข้อมูลที่ผู้ใช้กรอกเพื่อสร้างบัญชี ทดลองใช้งาน และติดต่อกลับตามความจำเป็น
1. ต้องมี Checkbox: ข้าพเจ้าได้อ่านและยอมรับนโยบายความเป็นส่วนตัว
1. ต้องแจ้งว่าเป็นระบบ Demo ข้อมูลอาจถูกลบหรือรีเซ็ต และไม่ควรใช้ข้อมูลสำคัญจริงหรือข้อมูลอ่อนไหว
#### 2.2 Error / Access State Pages

| หน้า | ใช้เมื่อ |
| --- | --- |
| 404 Not Found | ไม่พบหน้า หรือไม่ต้องการเปิดเผยว่าข้อมูลนั้นมีอยู่จริง |
| 403 No Permission | ผู้ใช้อยู่ในบริบทถูกต้อง แต่ไม่มีสิทธิ์เข้าหน้านั้น |
| Token Expired | Verify Email / Reset Password / Invite Token หมดอายุ |
| Invalid Link / Invalid Token | ลิงก์หรือ Token ไม่ถูกต้อง |
| Workspace Not Found | ไม่พบ Workspace จาก slug/subdomain |
| Workspace Suspended / Deleted | Workspace ถูกระงับหรือถูกลบ |

หลักสำคัญ: ถ้าข้อมูลมีความลับหรือไม่ควรเปิดเผยว่ามีอยู่จริง ให้แสดงเป็น 404 แทน 403

### 3. Owner / Business Admin UI

Owner / Business Admin UI เป็นส่วนเสริมสำหรับผู้สร้างระบบ / ทีมเจ้าของ App ใน MVP ทำแบบเบื้องต้นเพื่อดู Workspace ลูกค้า และ Cleanup ได้ ไม่ลงลึกเป็น CRM, Billing, Subscription หรือ Business Analytics เต็มระบบ

| Page | Scope |
| --- | --- |
| Owner Dashboard แบบสรุป | ภาพรวมระบบ Demo / ลูกค้า / Workspace / Usage / Cleanup |
| Customers / Workspaces List | รายการ Workspace ทั้งหมดใน Environment นี้ |
| Workspace Detail เบื้องต้น | ดูข้อมูล Workspace, Owner, Contact, Usage, Status |
| Cleanup Candidates | Workspace ที่ยังไม่ยืนยัน Email, ไม่เคย Login, ไม่มี Activity, Pending Deletion |

Optional / Phase ถัดไป:

1. Usage / Engagement เชิงลึก
1. Business Reports
1. Customer Contacts แบบละเอียด
1. Analytics
1. CRM
1. Billing / Subscription
### 4. Workspace Portal UI

Workspace Portal UI คือหัวใจหลักของผู้ใช้งานจริงหลัง Login เข้า Workspace เป็นศูนย์กลางบริหาร Workspace และดูภาพรวมงานก่อนเข้า Project

| Page / Menu | MVP Scope |
| --- | --- |
| Workspace Dashboard | ภาพรวม Project, My Tasks, Timeline, Notification, Announcement, Activity ตามสิทธิ์ |
| Projects | รายการ Project, Search/Filter, Status Summary, Create Project, Quick Action |
| My Tasks | งานที่ Assign ให้ฉัน, Watcher, Due Soon, Overdue, Filter |
| Calendar / Timeline | Calendar View, Timeline List, Upcoming, Overdue, Filter |
| Team / Members | Member List, Invite/Add, Create Profile, Change Role, Suspend/Remove, Status |
| Reports | Project, Task, Timeline/Overdue, Finance, Usage ตามสิทธิ์ |
| Notifications / Announcements | Notification ราย user และ Announcement ระดับ Workspace/Project |
| Workspace Settings | Profile, Master Data, Config, Notification Settings, Usage/Storage, Danger Zone |

#### 4.1 Workspace Portal: Role Visibility

| Role | มุมมองหลัก |
| --- | --- |
| Owner / Admin | เห็นภาพรวมทั้ง Workspace, จัดการ Members, Settings, Master Data, Reports ตามสิทธิ์ |
| Executive / Manager | เห็นภาพรวมหลาย Project และ Reports ตามสิทธิ์ |
| User / Project Member | เห็นเฉพาะ Project, Task, Timeline ที่เกี่ยวข้อง |
| Finance | เห็น Finance ตาม Finance Visibility |

#### 4.2 Member Profile / Portfolio

ระบบควรมี Member Profile / Portfolio เพื่อเก็บข้อมูลบุคคลให้ละเอียดกว่า Account และ Membership

MVP รองรับ:

1. รูปโปรไฟล์
1. Bio
1. ตำแหน่ง / แผนก
1. Skills
1. Project History จากระบบ
Optional / Phase ถัดไป:

1. Portfolio เต็มรูปแบบ
1. ผลงานพร้อมไฟล์แนบ
1. Skill Rating
1. Endorsement / รีวิวทักษะ
1. Export Portfolio
1. Public Profile
### 5. Project Detail UI

Project Detail UI คือพื้นที่ทำงานจริงของแต่ละ Project ใช้โครง Project Header + Project Tabs

| ส่วน | ข้อมูล / Scope |
| --- | --- |
| Project Header | Project Code, Name, Type, Status, Priority, Owner, Timeline สำคัญใกล้ถึง, Action ตามสิทธิ์ |
| Tabs | Overview, Tasks, Timeline, Documents, Team, Finance, Announcements, Activity, Settings |

| Tab | MVP Scope |
| --- | --- |
| Overview | Project Summary, Task Summary, Timeline/Upcoming, Document Summary, Finance ตามสิทธิ์, Recent Activity |
| Tasks | Task List, Filter, Create, Detail Drawer/Modal, Change Status, Reassign, Checklist, Comment, Attachment |
| Timeline | Timeline List, Calendar เบา ๆ, Create Item, Upcoming/Overdue Indicator, Delivery Attempts, Meeting Attendees |
| Documents | Folder Tree, File List, Upload, Move, External Link, File Detail, Soft Delete/Restore, Signed URL |
| Team | Project Member List, Add Member, Change Role, Assign Position, Remove, Member Status, Skills เบื้องต้น |
| Finance | Finance Overview, Budget, Finance Entries, Attachments, Finance Visibility |
| Announcements | Project Announcement, Create/Edit/Delete ตามสิทธิ์, Pin, Expire, Required Acknowledgement |
| Activity | Activity List, Filter by action/user/date, read-only ใน MVP |
| Settings | Task Workflow, Finance Visibility, Archive/Restore, Danger Zone แบบจำกัด |

#### 5.1 Project Team Rule

1. Project Team Tab เลือกคนจาก Workspace Members / User Profiles ไม่ใช่สร้างบัญชีใหม่เต็มรูปแบบในหน้านี้
1. ถ้า User Account = Disabled ไม่ควรให้เพิ่มเข้า Project ใหม่
1. ถ้า Workspace Membership ไม่ Active ไม่ควรให้เพิ่มเข้า Project ใหม่
1. Profile ที่ยังไม่มี Account อาจเพิ่มเป็น Project Member ได้บางกรณี เช่น ผู้ติดต่อ, outsource, placeholder แต่ต้องแสดงว่า login ไม่ได้
1. Project ต้องมี Project Owner อย่างน้อย 1 คน และห้าม Remove Project Owner คนสุดท้าย
#### 5.2 File / Document Security ใน UI

1. ห้ามแสดงหรือเก็บ Full URL ถาวรของไฟล์
1. ผู้ใช้กดเปิดไฟล์ -> Backend ตรวจสิทธิ์ -> สร้าง signed URL / temporary URL -> เปิดไฟล์
1. Attachment คือ relation กับไฟล์ใน Project File Center ไม่ใช่ copy file ซ้ำเสมอ
### 6. Platform Admin UI

Platform Admin UI ใช้สำหรับผู้ดูแลระบบเชิงเทคนิค / operation ไม่ใช่หน้าฝั่งลูกค้า และไม่ใช่ Owner Business Console

| Menu | MVP Scope |
| --- | --- |
| Platform Dashboard | ภาพรวมระบบเชิงเทคนิค |
| Workspace Management | ดูและจัดการ Workspace ตามสิทธิ์สูง |
| Platform Config | ตั้งค่ากลางของระบบ |
| App Master Data | จัดการ seed/master กลาง |
| Reference Data | จัดการข้อมูลอ้างอิง เช่น ที่อยู่ |
| File / Storage Cleanup | จัดการไฟล์ soft deleted, orphan, replaced ตาม policy |
| Audit Logs | ดูประวัติ action สำคัญ |
| Security Events | ดูเหตุการณ์ด้าน security |
| Hard Delete Zone | พื้นที่ action ความเสี่ยงสูง ต้องแยกชัดและมี log |

หลักสำคัญ:

1. ใช้ได้เฉพาะ Platform Admin / System Admin / Support Admin ตามสิทธิ์
1. ไม่ควรใช้ Platform Admin แทน Workspace Portal
### 7. UI Role Visibility Summary

| User / Role | เห็นอะไร |
| --- | --- |
| Public User | Public App UI |
| Owner / Business Admin | Owner Console |
| Platform Admin | Platform Admin UI |
| Workspace Owner / Admin | Workspace Portal + Settings + Team + Reports ตามสิทธิ์ |
| Executive / Manager | ภาพรวม Workspace / Project / Reports ตามสิทธิ์ |
| Project Member | Project / Task / Timeline ที่เกี่ยวข้อง |
| Finance | Finance ตาม Finance Visibility |
| Viewer | ดูอย่างเดียวตามสิทธิ์ |

ทุกเมนูและข้อมูลใน UI ต้องขึ้นกับ Role / Permission และต้องตรวจสิทธิ์จาก Backend ด้วย

### 8. MVP vs Future UI Scope

| Phase | UI Scope |
| --- | --- |
| MVP-0 | Public App UI แบบเบา, Create Workspace, Verify Email, Login, Forgot/Reset Password, Workspace Portal เบื้องต้น, Project Detail, Project/Team/Task/Documents พื้นฐาน, Activity พื้นฐาน, Error / Access State Pages |
| MVP-1 | Timeline / Deliverable, Finance เบื้องต้น, Notification / Announcement, Calendar, Reports แบบพื้นฐาน, Workspace Settings ที่จำเป็น, Platform Admin เบื้องต้น |
| Future / Optional | Marketing Website เต็มรูปแบบ, CRM / Billing, Advanced Analytics, Custom Report Builder, Workflow Automation, Permission Builder ละเอียด, Portfolio เต็มรูปแบบ, Support Ticket / Live Chat, Knowledge Base เต็มระบบ, Online Payment |

### 9. Final Decision

1. ชุดที่ 10 UI / Page Structure เน้นหน้าที่ใช้ทำงานจริงก่อน
1. Template ต้องแยกตาม Context: Public, Owner, Workspace, Project, Platform
1. Workspace Portal และ Project Detail เป็นหัวใจหลักของผู้ใช้งานจริง
1. Owner / Business Admin UI ทำแบบเบื้องต้น ไม่บวมเป็น CRM/Analytics เต็มระบบ
1. Platform Admin UI ใช้สำหรับ operation และ action ความเสี่ยงสูงเท่านั้น
1. ระบบต้องมี Error / Access State Pages กลาง และใช้ 404 แทน 403 เมื่อไม่ควรเปิดเผยว่าข้อมูลมีอยู่จริง
