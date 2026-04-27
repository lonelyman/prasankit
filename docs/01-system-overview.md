# ชุดที่ 1: ภาพรวมระบบและขอบเขต

Multi-Tenant SaaS Project Management Platform

เอกสารสรุป Decision ที่ตกลงและล็อกแล้ว

## 1. เป้าหมายของชุดที่ 1

เอกสารชุดนี้ใช้ล็อกภาพรวมและขอบเขตของระบบ ก่อนเข้าสู่การออกแบบเชิงลึกในชุดถัดไป เช่น data structure, permission detail, API, UI flow หรือ deployment detail

- กำหนดกลุ่มผู้ใช้และแนวทาง demo แรก
- กำหนดวงจรชีวิตของโปรเจค
- กำหนดประเภทของโปรเจค
- กำหนด MVP Scope ที่ต้องมีในรอบแรก
- กำหนดเมนูหลักของ Workspace และ Project Detail
- กำหนดสิ่งที่ยังไม่ทำใน MVP แรก เพื่อกัน scope บวม
## 2. สรุป Decision ที่คอมมิตแล้ว

| ข้อ | หัวข้อ | Decision |
| --- | --- | --- |
| 1 | Target Demo Group | ทีม/องค์กรที่ทำงานแบบโครงการ รองรับทั้งงานภายในองค์กรและงานลูกค้า/งานรับจ้าง |
| 2 | Project Lifecycle | Draft > Planning > Proposal > Active > Closing > Maintenance > Closed > Archived |
| 3 | Project Type | Internal Project และ Client Project |
| 4 | MVP Core | Project Profile, Team, Task, Deliverable, Finance, Document, Activity Log, Announcement |
| 5 | Workspace Menu | แยกเมนูระดับองค์กรและเมนูรายละเอียดโปรเจค |
| 6 | Out of Scope | ยังไม่ทำ feature enterprise/advanced ที่ทำให้ MVP บวม |

## 3. Target Demo Group

ระบบจะวางตำแหน่งเป็นแพลตฟอร์มสำหรับทีมและองค์กรที่ทำงานแบบโครงการ โดยไม่ล็อกเฉพาะอุตสาหกรรมใดอุตสาหกรรมหนึ่ง

- รองรับงานภายในองค์กร เช่น งานของแผนก IT, HR, Operation หรือทีมภายใน
- รองรับงานลูกค้า/งานรับจ้าง เช่น งาน implementation, consulting, software, campaign หรือ project service
- MVP แรกใช้ Template กลางสำหรับงานโครงการทั่วไป
- ระบบต้องออกแบบให้ขยายเป็น Template เฉพาะทางได้ในอนาคต เช่น IT Project, Software Development, Marketing Campaign, Construction Project, Consulting Project, Loan Case / Credit Operation หรือ Internal Operation
## 4. Project Lifecycle / Project Model

Project Status ใช้บอกสถานะภาพรวมของทั้งโปรเจค ไม่ใช่สถานะงานย่อยภายในโปรเจค

| Status | ภาษาไทย | ความหมาย |
| --- | --- | --- |
| Draft | ร่าง | เก็บข้อมูลไว้ก่อน ข้อมูลยังไม่ครบ หรือยังเป็นแนวคิดเบื้องต้น |
| Planning | วางแผน | ยังไม่เริ่มทำจริง แต่เริ่มเตรียมทีม เอกสาร แผนงาน หรืองบประมาณ |
| Proposal | เสนองาน | อยู่ระหว่างเสนอราคา ยื่นประมูล หรือรอลูกค้าตัดสินใจ |
| Active | ดำเนินงาน | เริ่มทำงานจริงแล้ว มี task, team, deliverable หรือค่าใช้จ่ายเกิดขึ้น |
| Closing | ปิดงาน | ส่งมอบงานหลักแล้วหรือใกล้จบ รอตรวจรับ เคลียร์เอกสาร หรือสรุปงาน |
| Maintenance | ดูแลหลังส่งมอบ / MA | ส่งมอบงานหลักแล้ว แต่ยังมี MA, warranty หรือ support ตามสัญญา |
| Closed | จบงาน | จบภาระผูกพันทั้งหมดแล้ว |
| Archived | จัดเก็บ | ซ่อนจากงานหลัก แต่ยังค้นย้อนหลังได้ |

- Task Status เป็น workflow แยกต่างหาก และออกแบบโดยใช้แนวคิด Jira-lite
- ระบบต้องรองรับหลายภาษา โดยเริ่มต้นด้วยภาษาไทยและอังกฤษ
- คำแสดงผล เช่น status, menu, label, message ต้องเตรียมให้แปลได้ตั้งแต่แรก
## 5. Project Type

Project Type ใช้ระบุว่าลักษณะงานเป็นงานภายในหรือเป็นงานลูกค้า/งานรับจ้าง

| Project Type | ภาษาไทย | คำอธิบาย |
| --- | --- | --- |
| Internal Project | งานภายในองค์กร | งานที่เกิดขึ้นภายในองค์กร อาจมาจากแผนกตัวเองหรือแผนกอื่นในองค์กร |
| Client Project | งานลูกค้า / งานรับจ้าง | งานที่ทำให้ลูกค้าหรือผู้ว่าจ้างภายนอก |

- กรณีงานภายในที่มาจากคนละแผนก ไม่ต้องสร้าง Project Type ใหม่
- ให้ใช้ Internal Project และระบุ Requesting Unit หรือหน่วยงานผู้ขอแทน
- Project Template เป็นส่วนขยายในอนาคต ไม่ใช่ Project Type หลักใน MVP
## 6. MVP Core Scope

MVP แรกแบ่งเป็นเฟสเพื่อให้เริ่มทำและ demo ได้เร็ว โดยไม่พยายามทำทุก module เต็มพร้อมกันตั้งแต่รอบแรก

| Phase | เป้าหมาย | Module หลัก |
| --- | --- | --- |
| MVP-0 | ทำให้ระบบใช้งาน project/task ได้จริงแบบ end-to-end | Auth, Workspace Registration/Login, Tenant Resolution, Project Profile, Project Team, Task Board, Basic File Upload, Basic Activity Log |
| MVP-1 | เติม module บริหารโครงการให้ครบภาพ demo | Deliverable/Timeline, Finance เบื้องต้น, Announcement, Notification เบื้องต้น, Basic Reports |
| MVP-2 | เพิ่มงาน operation/admin และความสามารถต่อยอด | Platform Admin เพิ่มเติม, Cleanup, Advanced Report, Notification Rule, Export/Import บางส่วน |

ตารางด้านล่างคือ Core Module ของภาพรวม MVP ทั้งชุด แต่การ implementation ให้เริ่มจาก MVP-0 ก่อน

| ลำดับ | Core Module | ขอบเขตที่ตกลง |
| --- | --- | --- |
| 1 | Project Profile | ข้อมูลหลักของโปรเจค เช่น ชื่อ ประเภท สถานะ วันเริ่ม วันสิ้นสุด เจ้าของงาน และรายละเอียด |
| 2 | Project Team / Role Model | จัดการสมาชิกในโปรเจค บทบาท และสิทธิ์พื้นฐาน |
| 3 | Project Task | Jira-lite Task Board, ลากย้ายสถานะ, สถานะแยกรายโปรเจค |
| 4 | Project Deliverable / งวดงาน | จัดการงวดงาน สิ่งส่งมอบ รอบการส่งมอบ และผลตรวจรับ |
| 5 | Project Finance | Budget, Expense, Penalty / Adjustment พร้อมควบคุมการมองเห็นข้อมูลการเงิน |
| 6 | Project Document | Project File Center สร้าง folder และ upload file ได้ |
| 7 | Project Activity Log | Timeline + Audit-ready Data |
| 8 | Project Announcement | ประกาศสำคัญของโปรเจค ยังไม่ใช่ chat |

## 7. Role Model

ระบบต้องแยกสิทธิ์ระดับองค์กรออกจากสิทธิ์ระดับโปรเจค และต้องแยก Role ออกจากตำแหน่งงานจริง

### 7.1 Organization Role

| Role | ภาษาไทย | ขอบเขต |
| --- | --- | --- |
| Owner | เจ้าของ | ทำได้ทุกอย่างในองค์กร |
| Admin | แอดมิน / ผู้ดูแลระบบ | ดูแลระบบ กรอกข้อมูล จัดการข้อมูลพื้นฐาน และช่วยบริหาร workspace |
| Executive | ผู้บริหาร | ดูภาพรวมข้ามโปรเจค เห็น dashboard/report ระดับองค์กรตามสิทธิ์ |
| User | ผู้ใช้งาน | เห็นเฉพาะโปรเจคหรืองานที่ตัวเองเกี่ยวข้อง |

### 7.2 Project Role

| Role | ภาษาไทย | ขอบเขต |
| --- | --- | --- |
| Project Owner | เจ้าของโปรเจค | รับผิดชอบภาพรวมของโปรเจค |
| Project Manager | ผู้จัดการโปรเจค | จัดการงาน ทีม งวดงาน และความคืบหน้า |
| Member | สมาชิกโปรเจค | ทำงานในโปรเจคตามที่ได้รับมอบหมาย |
| Finance | การเงินโปรเจค | ดู/จัดการข้อมูลการเงินของโปรเจคตามสิทธิ์ |
| Viewer | ผู้ดูอย่างเดียว | ดูข้อมูลที่ได้รับอนุญาต โดยค่าเริ่มต้นไม่ควรเห็นข้อมูลการเงิน |

- Role = สิทธิ์การใช้งานในระบบ
- Project Position = ตำแหน่ง/หน้าที่ในโปรเจค
- Company Position = ตำแหน่งงานในบริษัท
- ทั้ง 3 เรื่องนี้ต้องแยกกัน เช่น Infrastructure Manager อาจเป็น Technical Lead ในโปรเจค และมี Project Role เป็น Project Manager
## 8. Project Task

- ใช้แนวคิด Jira-lite
- รองรับ Task Board และการลากย้ายสถานะ
- Task Status ต้องแยกตามรายโปรเจค
- ระบบมี Default Task Status ให้ตอนสร้างโปรเจค แต่แต่ละโปรเจคสามารถเพิ่ม แก้ไข หรือเรียงลำดับ status ของตัวเองได้
- Project Status กับ Task Status เป็นคนละระบบ
- Task สามารถผูกกับ Deliverable ได้

| Default Task Status |
| --- |
| Backlog |
| To Do |
| In Progress |
| Review |
| Testing |
| Blocked |
| Done |
| Cancelled |

## 9. Project Deliverable / งวดงาน

- Deliverable = งวดงาน / สิ่งส่งมอบของโปรเจค
- รองรับวันกำหนดส่ง วันที่ส่งจริง วันที่ตรวจรับ และผลการตรวจรับ
- รองรับส่งเร็ว ตรงเวลา ส่งช้า และส่งแล้วไม่ผ่าน
- รองรับการส่งหลายรอบ โดยแยก Delivery Attempt เป็นประวัติของแต่ละครั้งที่ส่ง
- ส่งเร็ว/ตรงเวลา/ส่งช้า ไม่ใช่สถานะหลัก แต่เป็น Delivery Timing ที่คำนวณจากวันกำหนดส่งเทียบกับวันที่ส่งครั้งแรก
- Deliverable สามารถผูกกับ Task และ Document ได้ใน MVP-1
- การเชื่อม Deliverable กับ Finance เป็น future/optional ผ่าน Reference Tracking ไม่ใช่ scope ของ MVP-0

| Deliverable Status | ภาษาไทย |
| --- | --- |
| Planned | วางแผนไว้ |
| In Progress | กำลังดำเนินการ |
| Submitted | ส่งมอบแล้ว / รอตรวจรับ |
| Accepted | ตรวจรับแล้ว |
| Rejected | ไม่ผ่าน / ตีกลับ |
| Cancelled | ยกเลิก |

| Delivery Result | ภาษาไทย |
| --- | --- |
| Pending Review | รอตรวจรับ |
| Accepted | ผ่านการตรวจรับ |
| Rejected | ไม่ผ่าน / ตีกลับ |
| Accepted with Condition | ผ่านแบบมีเงื่อนไข |

## 10. Project Finance

Finance เป็นแท็บหลักใน Project Detail และแยกเป็นแท็บย่อยภายใน

- Overview = ภาพรวมการเงินของโปรเจค
- Budget = งบประมาณ / มูลค่างาน / กรอบการเงิน
- Expenses = ค่าใช้จ่ายที่เกิดขึ้นจริง
- Penalty / Adjustment = ค่าปรับ / ส่วนลด / ค่าเสียหาย / รายการปรับยอดตามสัญญา
- Budget, Expense และ Penalty / Adjustment แยกกัน แต่สัมพันธ์กันได้เพื่อสรุปภาพรวมในอนาคต เช่น ใช้เกินงบ กำไร ขาดทุน หรือคงเหลือ
- ข้อมูลการเงินต้องสามารถซ่อนหรือจำกัดสิทธิ์การมองเห็นได้ เพราะบางหน่วยงานหรือบางโปรเจคถือเป็นข้อมูลลับ
- MVP ยังไม่ทำบัญชีเต็มระบบ เช่น ภาษี ใบแจ้งหนี้ รับชำระเงิน หรือ P/L เต็มรูปแบบ
## 11. Project Document

- ใช้แนวคิด Project File Center
- เป็นศูนย์รวมไฟล์ของโปรเจค
- ผู้ใช้สร้าง folder และตั้งชื่อ folder ได้เอง
- Upload file เข้า folder ได้
- ระบบสร้าง folder ตั้งต้นให้เมื่อสร้างโปรเจค
- ไฟล์สามารถผูกกับ Project, Task และ Deliverable ได้

| Folder ตั้งต้น |
| --- |
| 01_Proposal |
| 02_Contract |
| 03_Requirement |
| 04_Design |
| 05_Meeting Notes |
| 06_Deliverables |
| 07_Acceptance |
| 08_Finance |
| 09_Maintenance |
| 99_Others |

- MVP ยังไม่ทำ Online Document Editor, OCR, Version Control เต็มรูปแบบ, E-signature หรือ Approval เอกสารหลายชั้น
## 12. Project Announcement

- ใช้สำหรับประกาศเรื่องสำคัญของโปรเจค
- รองรับการปักหมุดประกาศ
- รองรับระดับความสำคัญ
- แนบเอกสารที่เกี่ยวข้องได้
- ยังไม่ทำ Group Chat, Real-time Chat, Chat Thread หรือ Read Receipt ใน MVP
## 13. Project Activity Log

Activity Log ใน MVP จะแสดงเป็น Timeline ให้ผู้ใช้งานดูง่าย แต่ข้อมูลหลังบ้านต้องเก็บแบบ Audit-ready ตั้งแต่แรก

- Actor: ใครทำ
- Action: ทำอะไร
- Resource Type: ทำกับข้อมูลประเภทไหน
- Resource ID: ทำกับรายการไหน
- Timestamp: ทำเมื่อไหร่
- Old Value: ค่าเดิม
- New Value: ค่าใหม่
- IP Address
- User Agent / Device
- Result: สำเร็จหรือไม่สำเร็จ
- Tenant ID
- Project ID
- ระบบไม่ควรเก็บ log เป็นข้อความลอย ๆ อย่างเดียว แต่ต้องเก็บข้อมูลเชิงโครงสร้าง
- แนวทางหลักคือ Security-by-design, Audit-ready, Access Control, Change Tracking และ Data Protection
- MVP ยังไม่รับรองว่าผ่าน ISO หรือ compliance เต็มรูปแบบ แต่ต้องออกแบบให้ต่อยอดสู่การตรวจสอบมาตรฐานได้
- แนวทางอ้างอิงในอนาคต ได้แก่ OWASP ASVS, OWASP Top 10 และ ISO/IEC 27001:2022
## 14. Main Workspace Menu

### 14.1 Organization Level Menu

| EN | TH |
| --- | --- |
| Dashboard | แดชบอร์ด |
| Projects | โปรเจค |
| Tasks | งาน |
| Documents | เอกสาร |
| Team | ทีมงาน |
| Reports | รายงาน |
| Settings | ตั้งค่า |

### 14.2 Project Detail Menu

| EN | TH |
| --- | --- |
| Overview | ภาพรวม |
| Tasks | งาน |
| Deliverables | งวดงาน / สิ่งส่งมอบ |
| Finance | การเงิน |
| Documents | เอกสาร |
| Team | ทีมโปรเจค |
| Announcements | ประกาศ |
| Activity | กิจกรรม |
| Settings | ตั้งค่าโปรเจค |

- เมนู Finance แสดงตามสิทธิ์การเงิน
- Reports แสดงตามสิทธิ์ โดย Executive เห็นภาพรวมข้ามโปรเจค ส่วน User เห็นเฉพาะข้อมูลที่เกี่ยวข้อง
- Settings แสดงตามสิทธิ์ Owner, Admin, Project Owner หรือ Project Manager ตามบริบท
- เรื่อง subdomain ยังไม่ล็อกในชุดนี้ เช่น อาจใช้ company-a.yourdomain.com หรือ company-a-demo.yourdomain.com
- แนวทางตั้งชื่อ subdomain จะคุยแยกภายหลังว่า auto-generate, ให้ผู้สมัครกรอกเอง หรือใช้ทั้งสองแบบร่วมกัน
## 15. Out of Scope รอบแรก

รายการต่อไปนี้ยังไม่ทำใน MVP แรก เพื่อกัน scope บวมและทำให้ demo แรกไปถึงเร็วขึ้น

Real-time Chat

Payment Gateway

Subscription / Billing Plan

Custom Domain ของลูกค้า

SSO / Login ผ่านระบบองค์กร

Mobile App

Advanced Approval Workflow

Gantt Chart ขั้นสูง

Automation Rule

AI Assistant

E-signature

OCR / ค้นหาข้อความในไฟล์

Invoice / Payment Tracking เต็มระบบ

Compliance Certification เต็มรูปแบบ

- สำหรับ MVP ใช้ Project Announcement แทน Real-time Chat
- ระบบใช้ Responsive Web ก่อน ยังไม่ทำ Mobile App
- Finance รอบแรกมี Budget, Expense และ Penalty / Adjustment แต่ยังไม่ทำบัญชีเต็มระบบ
- Activity Log ต้อง Audit-ready แต่ยังไม่ claim ว่าผ่านมาตรฐาน compliance เต็มรูปแบบ
- MVP-0 ยังไม่บังคับทำ Finance, Deliverable, Notification และ Reports เต็มรูปแบบ ให้เริ่มหลัง Project/Task/File/Activity พื้นฐานใช้งานได้ก่อน
## 16. สรุปชุดที่ 1

ชุดที่ 1 ล็อกภาพรวมและขอบเขตของระบบเรียบร้อย โดยระบบจะเริ่มจาก Multi-Tenant Project Management Platform สำหรับทีม/องค์กรที่ทำงานแบบโครงการ รองรับทั้งงานภายในและงานลูกค้า/งานรับจ้าง

- MVP แรกเน้น project lifecycle, team, task, deliverable, finance, document, announcement และ activity log
- Task ใช้แนวคิด Jira-lite และสถานะงานแยกตามรายโปรเจค
- Deliverable รองรับส่งหลายรอบและผลตรวจรับ
- Finance แยก Budget, Expense และ Penalty / Adjustment พร้อมควบคุมการมองเห็น
- Document ใช้แนวคิด Project File Center
- Activity Log ต้องแสดงง่ายแต่เก็บข้อมูลแบบ Audit-ready
- ระบบรองรับหลายภาษา เริ่มจากไทยและอังกฤษ
