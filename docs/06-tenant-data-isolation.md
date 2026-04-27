# ชุดที่ 6: Tenant / Data Isolation

Project Management Platform - Core Architecture

| เอกสารนี้สรุปแนวทาง Multi-tenant, Tenant Resolution, Tenant Context, Data Access Rule, Workspace Lifecycle, Backup/Restore, File Lifecycle และ Security Guardrails ที่ตกลงกันแล้ว |
| --- |

## 1. วัตถุประสงค์ของชุดที่ 6

ชุดนี้ใช้กำหนดวิธีแยกข้อมูลของแต่ละ Workspace / Tenant อย่างปลอดภัย ก่อนเข้าสู่ Database Design และ API Design

- ให้แต่ละ Workspace แยกข้อมูลออกจากกันชัดเจน
- ป้องกันข้อมูลข้าม Workspace
- กำหนดหลักการ resolve tenant จาก URL
- กำหนดกติกา query / create / update / delete ด้วย tenant_id
- กำหนด lifecycle ของ Workspace และข้อมูลไฟล์ใน MinIO
## 2. Multi-tenant Strategy

| Decision: ใช้ Shared Database + tenant_id |
| --- |

ระบบจะใช้ฐานข้อมูลร่วมกัน แต่ข้อมูลที่เป็นของ Workspace ต้องแยกด้วย tenant_id

### หลักการ

- ทุก table ที่เป็นข้อมูลของ Workspace ต้องมี tenant_id
- ทุก query ของข้อมูล Workspace ต้อง filter tenant_id เสมอ
- tenant_id ต้องมาจาก backend Tenant Context ไม่รับจาก frontend โดยตรง
### ตัวอย่าง table ที่ต้องมี tenant_id

- projects
- tasks
- timeline_items
- files
- finance_entries
- announcements
- notifications
- activity_logs
### ตัวอย่าง table ที่อาจไม่ต้องมี tenant_id

- users
- platform_configs
- app_masters
- reference_countries
- reference_provinces
แต่ table ความสัมพันธ์กับ Workspace เช่น workspace_memberships ต้องมี tenant_id / workspace_id เพื่อใช้ตรวจสิทธิ์

## 3. Tenant Resolution

| Decision: ระบบต้องหา Workspace / Tenant จาก URL ที่ผู้ใช้เข้ามา |
| --- |

### Production Mode

ระบบจริงใช้ subdomain เป็นหลัก เช่น https://company-a-demo.yourdomain.com

| ขั้นตอน | รายละเอียด |
| --- | --- |
| 1 | อ่าน subdomain เช่น company-a-demo |
| 2 | นำ slug ไปค้นหา workspace.slug |
| 3 | เมื่อพบ Workspace จะได้ tenant_id, workspace_id และ workspace_status |
| 4 | ใช้ tenant_id นี้กับทุก request ภายใน Workspace |

### Development Mode

สำหรับ local development ให้รองรับ path-based tenant resolution เพื่อให้พัฒนาได้ง่ายโดยไม่ต้องตั้ง local DNS

| Environment | URL ตัวอย่าง | วิธีอ่าน slug |
| --- | --- | --- |
| Production | https://company-a-demo.yourdomain.com | subdomain = company-a-demo |
| Development | http://localhost:3000/w/company-a-demo | path /w/:slug = company-a-demo |

### Config ที่ต้องมี

Production: TENANT_RESOLUTION_MODE=subdomain

Development: TENANT_RESOLUTION_MODE=path และ DEV_TENANT_PATH_PREFIX=/w

## 4. Tenant Context

| Decision: Backend ต้องสร้าง Tenant Context จาก URL / slug และใช้ context นี้กับ query, permission และ activity log |
| --- |

### Tenant Context ควรมี

- tenant_id
- workspace_id
- workspace_slug
- workspace_status
- user_id
- user_workspace_role
### ใช้เพื่อ

- กรองข้อมูลทุก query ด้วย tenant_id
- ตรวจสิทธิ์ user ใน Workspace นั้น
- ป้องกัน user ข้าม Workspace
- ใช้บันทึก activity log / audit log

| tenant_id ห้ามรับจาก client โดยตรงเป็นหลัก ต้อง resolve ฝั่ง backend เท่านั้น |
| --- |

## 5. Data Access Rule

| Decision: Workspace data ทุก table ต้องมี tenant_id และทุก query ต้อง filter tenant_id |
| --- |

### หลักการสำคัญ

- ห้าม query ด้วย id อย่างเดียว
- Read / Update / Delete ต้องมี tenant_id จาก Tenant Context
- Create ต้อง set tenant_id จาก backend Tenant Context
- ไม่รับ tenant_id จาก request body

| ไม่ควรทำ | ควรทำ |
| --- | --- |
| WHERE id = task_id | WHERE id = task_id AND tenant_id = current_tenant_id |
| รับ tenant_id จาก frontend | set tenant_id จาก Tenant Context ฝั่ง backend |
| update ด้วย id เดี่ยว | update ด้วย id + tenant_id |

## 6. User / Workspace Membership Rule

| Decision: User Account หนึ่งบัญชีเข้าร่วมได้หลาย Workspace และสิทธิ์ต้องคิดแยกตาม Workspace |
| --- |

### หลักการ

- User Account = ตัวตนของคนหนึ่งคน
- Workspace Membership = ความสัมพันธ์ว่า user อยู่ workspace ไหน และมี role อะไร
- ทุก request ต้องเช็คว่า user เป็น Active member ของ Workspace นั้น
- ถ้าไม่ใช่ Active member ห้ามเข้าถึงข้อมูล Workspace
### ขั้นตอนตรวจสิทธิ์ต่อ request

- Resolve Tenant Context
- ตรวจว่า user เป็นสมาชิกของ Workspace นี้หรือไม่
- ตรวจว่า membership status = Active หรือไม่
- ตรวจ role / permission ตาม action ที่จะทำ
### Membership Status

| Status | ภาษาไทย | ความหมาย |
| --- | --- | --- |
| Active | ใช้งานอยู่ | เข้าใช้งาน Workspace ได้ตามสิทธิ์ |
| Removed | ถูกนำออก | ไม่สามารถเข้า Workspace นี้ได้ แต่ประวัติยังอยู่ |
| Suspended | ถูกระงับ | ถูกระงับเฉพาะ Workspace นี้ |

## 7. Workspace Status / Access Rule

| Decision: Workspace Status หลักคือ Active, Suspended, Pending Deletion, Deleted |
| --- |

| Status | ความหมาย | Access Rule |
| --- | --- | --- |
| Active | ใช้งานอยู่ | สมาชิกเข้าใช้งานได้ตามสิทธิ์ |
| Suspended | ถูกระงับ | ผู้ใช้ทั่วไปเข้าไม่ได้ แสดงหน้าระงับการใช้งาน |
| Pending Deletion | รอลบ | Owner เห็นสถานะและสามารถ Cancel Deletion ได้ ผู้ใช้ทั่วไปไม่ควรใช้งานปกติ |
| Deleted | ลบแล้ว / Soft Delete | เข้าใช้งานไม่ได้ แต่ข้อมูลยังอยู่จนกว่า Platform Admin จะ Hard Delete |

### Hard Delete

Hard Delete ไม่ใช่ Workspace Status หลัก แต่เป็น action ของ Platform Admin เพื่อ cleanup ข้อมูลจริงภายหลัง

### Flow การปิด Workspace

Active → Owner ขอปิด Workspace → Pending Deletion → ครบเวลาตาม config → Deleted / Soft Delete → Platform Admin Hard Delete ภายหลัง

ถ้า Owner เปลี่ยนใจ: Pending Deletion → Cancel Deletion → Active

| Inactive ไม่ใช้เป็น Workspace Status ให้ใช้เป็น Report / Indicator จาก last_login_at / last_activity_at แทน |
| --- |

## 8. Backup / Restore Tenant

| Decision: MVP ใช้แนวทาง Backup / Restore ระดับ VM / Server ก่อน |
| --- |

### แนวทาง MVP

- Backup ทั้ง VM / Server
- Backup database รวม
- Backup file storage รวม เช่น MinIO volume
- Restore โดยผู้ดูแลระบบ / Platform Admin
- ยังไม่ทำ restore ราย Workspace
### ยังไม่ทำใน MVP

- ปุ่ม restore ให้ลูกค้าใช้เอง
- restore เฉพาะ tenant_id
- export / import ราย Workspace
- backup policy ราย Workspace
### แนวทางอนาคต

- Export ข้อมูลราย Workspace
- Restore ราย Workspace
- Backup แยก tenant
- Archive tenant ก่อน Hard Delete
## 9. Tenant Data Deletion / Cleanup

| Decision: MVP ใช้ manual cleanup โดย Platform Admin และ Hard Delete ต้องลบทั้ง DB data และ file storage ที่เกี่ยวข้อง |
| --- |

### Flow

- Owner ขอปิด Workspace
- Workspace เข้า Pending Deletion
- ครบเวลาตาม config
- Workspace เป็น Deleted / Soft Delete
- Platform Admin Cleanup / Hard Delete ภายหลัง
### Hard Delete ต้องลบเป็นชุดเดียวกัน

- Workspace record
- Projects / Tasks / Timeline / Finance
- Documents metadata
- Files ใน MinIO
- Announcements / Notifications
- Reference tracking
- Activity logs ตาม policy
### Deletion Log ก่อน Hard Delete

- tenant_id
- workspace_id
- workspace_name
- slug
- deleted_by
- deleted_at
- reason
- data_summary เช่น จำนวน project / file / storage

| ห้ามลบแค่ database แล้วปล่อยไฟล์ค้าง และห้ามลบแค่ไฟล์แล้วปล่อย metadata ค้าง |
| --- |

## 10. File Lifecycle Policy

| Decision: File Lifecycle ต้องแยกประเภทไฟล์ และ MinIO object ไม่ควรถูกลบทันทีทุกกรณี |
| --- |

### Business Document File

ใช้กับเอกสารโปรเจค, ไฟล์แนบ Task, ไฟล์ Finance, ไฟล์ Deliverable

- ลบจาก UI = Soft Delete ก่อน
- ไฟล์จริงใน MinIO ยังไม่ลบทันที
- ลบจริงตอน Hard Delete หรือ Cleanup ตาม policy
- เหมาะกับข้อมูลที่ต้องตรวจสอบย้อนหลัง
### Replaceable Media File

ใช้กับรูปโปรไฟล์, โลโก้ Workspace, cover, avatar

- อัปโหลดใหม่แทนไฟล์เดิมได้
- ไฟล์เก่าถือว่าไม่ใช้งานแล้ว
- mark เป็น unused / replaced
- cleanup ได้ตาม policy
- ไม่ต้องทำ version history ใน MVP
### File Cleanup Policy ระดับ App

- cleanup ไฟล์ soft deleted เกิน X วัน
- cleanup ไฟล์ replaced / unused เกิน X วัน
- cleanup orphan file ที่ไม่มี metadata อ้างอิง
- cleanup ไฟล์ของ Workspace ที่ Hard Delete แล้ว
### Config ที่ควรมี

- file_soft_delete_retention_days
- replaced_file_retention_days
- orphan_file_retention_days

| ห้ามลบ MinIO object ถ้า metadata / activity / reference ยังจำเป็นต้องใช้งาน และห้ามปล่อย orphan file จำนวนมากจนกินพื้นที่โดยไม่มีเจ้าของ |
| --- |

## 11. Tenant Security Guardrails

| Decision: ต้องมีกติกาป้องกันข้อมูลข้าม tenant ในระดับ backend / query / test |
| --- |

- Backend resolve tenant เอง
- tenant_id มาจาก Tenant Context
- ทุก query ของ Workspace data ต้อง filter tenant_id
- Create / Update / Delete ต้องตรวจ tenant_id
- Activity Log ต้องมี tenant_id
- ต้องมี test case กันข้อมูลข้าม tenant
### Test Case สำคัญ

User A อยู่ Workspace A พยายามเปิดข้อมูลของ Workspace B ระบบต้องปฏิเสธ

## 12. สรุป Decision ชุดที่ 6

| หัวข้อ | Decision |
| --- | --- |
| Multi-tenant Strategy | Shared Database + tenant_id |
| Tenant Resolution | Production ใช้ subdomain / Development ใช้ path /w/:slug |
| Tenant Context | Backend resolve และใช้ tenant_id จาก context เท่านั้น |
| Data Access Rule | ทุก query ของ Workspace data ต้อง filter tenant_id |
| Membership Rule | User หนึ่งบัญชีอยู่ได้หลาย Workspace แต่ต้องเป็น Active member |
| Workspace Status | Active, Suspended, Pending Deletion, Deleted |
| Backup / Restore | MVP มองระดับ VM / Server ก่อน |
| Deletion / Cleanup | Hard Delete โดย Platform Admin และต้องมี Deletion Log |
| File Lifecycle | แยก Business Document File กับ Replaceable Media File |
| Security Guardrails | ต้องมี query guard และ test case กันข้อมูลข้าม tenant |
