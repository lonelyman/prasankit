## ชุดที่ 2: Tenant / Workspace / Registration Flow

เอกสารสรุปแนวทางระบบ Project Management Platform
สถานะ: Confirmed / สำหรับใช้ต่อจากเอกสารชุดที่ 0 และชุดที่ 1

ขอบเขตเอกสาร: เอกสารนี้รวบรวมเฉพาะสิ่งที่ตกลงกันแล้วในชุดที่ 2 ไม่รวมประเด็นที่ยังไม่ได้ตัดสินใจหรือยังไม่ได้คุยในรายละเอียด

### สรุปภาพรวม

ชุดที่ 2 ใช้กำหนดแนวคิดเรื่อง Tenant, Workspace, การสมัครใช้งาน, Subdomain, Limit, Lifecycle และการเพิ่มสมาชิกเข้าระบบ โดยยึดแนวคิดว่า Workspace คือพื้นที่ทำงานหนึ่งชุดที่อาจเป็นของบริษัท หน่วยงาน ทีม หรือบุคคลทำงานคนเดียวได้

Decision ที่ล็อกแล้ว

1. Tenant เป็นคำเชิงเทคนิค ส่วน UI หลักใช้คำว่า Workspace / พื้นที่ทำงาน
1. ช่วงแรก Workspace Registration ใช้ Auto Approve แต่รองรับ Manual Approve ในอนาคต
1. Workspace มี Lifecycle และสามารถ Suspend, Pending Deletion, Soft Delete และ Hard Delete โดยมี Deletion Log ได้
1. User Account แยกจาก Workspace และหนึ่งบัญชีสามารถเข้าร่วมได้หลาย Workspace
1. Subdomain ใช้ Workspace Slug เป็นฐาน และ Demo marker เช่น -demo ต้องเป็น config
1. Usage Limit ใช้หลัก 0 = unlimited และ > 0 = limited by value
1. Demo Workspace และ Production Workspace ใช้โครงสร้างระบบเดียวกัน
1. Member Registration รองรับทั้ง Workspace Signup Page และ Admin-created User
### 1. Tenant / Workspace Definition

#### 1.1 คำหลัก

| คำ | ความหมาย |
| --- | --- |
| Tenant | คำเชิงเทคนิคของระบบ ใช้ในระดับ architecture / database / backend |
| Workspace | คำที่ใช้ใน UI หลัก หมายถึงพื้นที่ทำงานแยกของผู้สมัครหนึ่งราย |

#### 1.2 นิยาม Workspace

Workspace คือพื้นที่ใช้งานแยกของผู้สมัครหนึ่งราย โดยผู้สมัครอาจเป็นองค์กร หน่วยงาน บริษัท ทีม หรือบุคคลทำงานคนเดียว

ตัวอย่างผู้สมัคร:

1. บริษัท
1. หน่วยงาน
1. ทีม
1. แผนก
1. Freelance
1. Outsource
1. บุคคลทำงานคนเดียว
Remark: ระบบต้องไม่ออกแบบให้ผูกกับบริษัทเท่านั้น เพราะผู้ใช้งานอาจเป็นทีมเล็กหรือบุคคลที่รับงานเองได้

### 2. Workspace Registration + Lifecycle

#### 2.1 Registration Mode

Decision

1. ช่วงแรกใช้ Auto Approve ผู้สมัครลงทะเบียนแล้วระบบสร้าง Workspace ให้ใช้งานได้ทันที
1. ระบบต้องรองรับ Approval Mode: Auto Approve และ Manual Approve ในอนาคต

| Mode | ความหมาย |
| --- | --- |
| Auto Approve | สมัครแล้วสร้าง Workspace ให้ทันที เหมาะกับช่วงเปิดให้ทดลองใช้ |
| Manual Approve | สมัครแล้วรอ Platform Admin อนุมัติ เหมาะกับช่วงที่ต้องคุมคุณภาพหรือป้องกัน spam |

#### 2.2 Workspace Status

| Status | ภาษาไทย | ความหมาย |
| --- | --- | --- |
| Active | ใช้งานอยู่ | Workspace ใช้งานปกติ |
| Suspended | ระงับการใช้งาน | ถูกปิดชั่วคราวโดย Platform Admin หรือระบบ |
| Pending Deletion | รอลบ | อยู่ในช่วงรอก่อนลบจริง สามารถยกเลิกได้ก่อน Hard Delete |
| Deleted | ลบแล้ว | ถูกลบในเชิงระบบหรือถูก soft delete แล้ว |

Inactive ไม่ใช้เป็น Workspace Status หลัก ให้ใช้เป็น report / indicator จาก last_login_at หรือ last_activity_at แทน เพื่อไม่ให้สถานะเชิง access control ปนกับสถานะเชิงพฤติกรรมการใช้งาน

#### 2.3 ข้อมูลที่ระบบต้องตรวจสอบได้

1. สมัครเมื่อไหร่
1. Login ล่าสุดเมื่อไหร่
1. Activity ล่าสุดเมื่อไหร่
1. มี Project กี่รายการ
1. มี User กี่คน
1. ใช้พื้นที่ไฟล์เท่าไหร่
#### 2.4 Workspace Close / Delete Flow

Workspace Owner สามารถขอปิด Workspace ได้ แต่ไม่ควร Hard Delete เองทันที

1. Workspace Owner กดขอปิด Workspace
1. ระบบแสดงคำเตือน
1. Owner ยืนยัน
1. Workspace เปลี่ยนเป็น Pending Deletion
1. ระบบส่งแจ้งเตือน
1. ระยะเวลารอก่อน Hard Delete เป็น config
1. Owner สามารถยกเลิกคำขอปิดได้ก่อน Hard Delete
1. Platform Admin เป็นผู้กด Hard Delete เอง
Remark: MVP ยังไม่ทำ Auto Hard Delete การลบถาวรต้องทำโดย Platform Admin เท่านั้น

#### 2.5 Pending Deletion Period

ระยะเวลารอก่อน Hard Delete ต้องเป็น Config ไม่ fix เป็น 30 วัน เพื่อรองรับนโยบาย กฎหมาย หรือข้อตกลงกับลูกค้า

ตัวอย่างค่า config:

1. 30 วัน
1. 60 วัน
1. 90 วัน
1. 120 วัน
1. หรือกำหนดเองตาม policy
#### 2.6 Deletion Log

ถ้ามีการ Hard Delete ต้องเก็บ Deletion Log เสมอ

ข้อมูลที่ควรเก็บ:

1. Workspace ID
1. Workspace Name
1. Subdomain
1. Deleted By
1. Deleted At
1. Deletion Reason
1. Previous Status
1. Last Login At
1. Last Activity At
1. Project Count
1. User Count
1. Storage Usage
#### 2.7 User Account vs Workspace

Decision

1. User Account แยกจาก Workspace
1. การปิด Workspace ไม่ลบ User Account
1. การลบ User Account ไม่ลบ Workspace โดยอัตโนมัติ
1. User Account หนึ่งบัญชีสามารถเข้าร่วมได้หลาย Workspace

| แนวคิด | ความหมาย |
| --- | --- |
| User Account | ตัวตนสำหรับ login กลางของคนหนึ่งคน เช่น email, password, account status |
| Workspace | พื้นที่ทำงานของบริษัท / ทีม / หน่วยงาน / บุคคล |
| Workspace Membership | ความสัมพันธ์ว่าคนนี้อยู่ Workspace ไหน และมี Role อะไร |
| Workspace Profile | ข้อมูลบุคคล/โปรไฟล์ภายใน Workspace นั้น ใช้แสดงผลและอ้างอิงงาน อาจมีหรือไม่มี User Account ก็ได้ |

Remark: ถ้าผู้ใช้มี Workspace เดียว ให้เข้า Workspace นั้นอัตโนมัติ หากมีหลาย Workspace ค่อยแสดงหน้าเลือก Workspace

### 3. Subdomain Strategy

#### 3.1 หลักการ

Decision

1. ใช้ Workspace Slug เป็นฐานของ subdomain
1. ระบบ auto-generate slug จากชื่อ Workspace
1. ผู้สมัครแก้ slug ได้ก่อนสร้าง Workspace
1. Workspace มี mode: Demo / Production
1. ถ้าเป็น Demo ให้ระบบใส่ demo marker ตาม config
1. ค่าเริ่มต้นใช้ suffix: -demo
1. Production ไม่ใส่ demo marker

| ประเภท | ตัวอย่าง |
| --- | --- |
| Demo | company-a-demo.yourdomain.com |
| Production | company-a.yourdomain.com |

#### 3.2 Config Strategy

แบ่ง config เป็น 2 กลุ่ม คือ Environment Config และ Database Config

| กลุ่ม Config | ตัวอย่างค่า | เหตุผล |
| --- | --- | --- |
| Environment Config | APP_DOMAIN, APP_PROTOCOL, SUBDOMAIN_ENABLED, WILDCARD_DOMAIN, SSL_PROVIDER, TRAEFIK_ENTRYPOINT | เกี่ยวกับ DNS / SSL / Traefik / domain เปลี่ยนผิดแล้วระบบเข้าไม่ได้ ไม่ควรให้แก้ผ่าน UI ทั่วไป |
| Database Config | workspace_default_mode, demo_marker_enabled, demo_marker_text, demo_marker_position, allow_user_edit_slug, approval_mode, reserved_slugs, slug_min_length, slug_max_length | เป็นกติกาของระบบ เปลี่ยนตามนโยบายได้ ไม่ต้อง redeploy และ Platform Admin ปรับได้จากหลังบ้าน |

Remark: การใส่ -demo ต้องเป็น config เพื่อให้เปลี่ยนรูปแบบได้ในอนาคต เช่น company-a-demo หรือ demo-company-a

### 4. Usage Limit Config

Decision

1. Usage Limit Config: 0 = ไม่จำกัด
1. Usage Limit Config: มากกว่า 0 = จำกัดตามค่าที่กำหนด
#### 4.1 Limit ที่ระบบควรรองรับ

1. max_workspaces_per_user
1. max_members_per_workspace
1. max_projects_per_workspace
1. max_storage_mb_per_workspace
1. max_file_size_mb
#### 4.2 ตัวอย่าง Config

ช่วงเปิด Demo แรก

max_workspaces_per_user = 0
max_members_per_workspace = 0
max_projects_per_workspace = 0
max_storage_mb_per_workspace = 0
max_file_size_mb = 0

ตัวอย่างอนาคต Free Plan

max_workspaces_per_user = 2
max_members_per_workspace = 10
max_projects_per_workspace = 5
max_storage_mb_per_workspace = 1024
max_file_size_mb = 20

Remark: ค่าพื้นที่จัดเก็บให้ใช้หน่วย MB เป็นหลัก เช่น 1024 MB = 1 GB

### 5. Demo Workspace vs Production Workspace

Decision

1. Demo Workspace กับ Production Workspace ใช้โครงสร้างระบบเดียวกัน
1. Demo / Production เป็น mode หรือ label ของ Workspace ไม่ใช่ระบบคนละชุด
1. หากอนาคตต้องการจำกัด feature หรือ resource ให้ใช้ Usage Limit Config / Plan Config แทนการแยกระบบ
ความต่างหลักมีแค่:

1. Mode / label
1. Subdomain marker เช่น -demo
1. Usage limit config
1. การนำไปใช้เชิงธุรกิจ
### 6. Workspace Profile

Decision

Workspace Profile ต้องมีข้อมูลพื้นฐานสำหรับระบุตัวตนของพื้นที่ทำงาน แต่ตอนสมัครให้บังคับกรอกเท่าที่จำเป็น เพื่อไม่ให้เริ่มใช้งานยากเกินไป

#### 6.1 Required ตอนสมัคร

1. Workspace Name — ชื่อพื้นที่ทำงาน
1. Workspace Slug — ชื่อสำหรับ Subdomain
1. Owner Account — บัญชีเจ้าของ Workspace
1. Contact Email — อีเมลติดต่อหลัก
#### 6.2 Optional / กรอกเพิ่มภายหลัง

1. Workspace Type — ประเภทพื้นที่ทำงาน
1. Contact Phone — เบอร์ติดต่อ
1. Logo — โลโก้
1. Description — รายละเอียด
#### 6.3 Workspace Type ที่รองรับ

| Workspace Type | ภาษาไทย |
| --- | --- |
| Personal | บุคคล |
| Team | ทีม |
| Department | แผนก / หน่วยงาน |
| Organization | องค์กร |
| Company | บริษัท |
| Freelance / Outsource | ฟรีแลนซ์ / รับงานภายนอก |

### 7. Workspace Owner / Member Registration Flow

#### 7.1 หลักคิด

| เรื่อง | ความหมาย |
| --- | --- |
| Account Registration | การสมัครบัญชีผู้ใช้ |
| Workspace Registration | การสร้างพื้นที่ทำงานใหม่ |

Implementation note ปัจจุบัน:

```text
Account Registration
-> Verify Email
-> Login
-> Workspace Registration
```

ดังนั้น `Owner Account` ใน Workspace Registration หมายถึงบัญชีที่ login และ verified แล้ว ไม่ใช่การสร้าง account ใหม่ซ้ำใน endpoint สร้าง workspace

Remark: สมาชิกทั่วไปไม่ต้องสมัคร Workspace ใหม่เพื่อเข้าทีม

#### 7.2 วิธีเพิ่มสมาชิกเข้า Workspace

Decision

ระบบรองรับ 2 วิธีหลัก:

1. Workspace Signup Page
1. Admin-created User

| วิธี | รายละเอียด | เหมาะกับ |
| --- | --- | --- |
| Workspace Signup Page | แต่ละ Workspace มีหน้าสมัครสมาชิกของตัวเอง เช่น company-a.yourdomain.com/register หรือ /signup | ให้ลูกทีมสมัครเข้ามาเองใน Workspace นั้น |
| Admin-created User | Owner / Admin สร้างสมาชิกให้ โดยกรอกชื่อ email/username กำหนด Workspace Role และตั้ง temporary password ได้ | องค์กรที่ต้องการให้ผู้ดูแลเตรียมรายชื่อไว้ก่อน |

#### 7.3 Workspace Signup Mode

1. Open Signup
1. Require Approval
1. Invite Only
1. Disabled
#### 7.4 Password Security

Decision

1. ถ้า password ถูกสร้างหรือแก้ไขโดยผู้อื่นที่ไม่ใช่เจ้าของบัญชี ระบบต้องบังคับให้ผู้ใช้เปลี่ยนรหัสผ่านเมื่อ login ครั้งถัดไป
1. Admin ตั้งได้เฉพาะ temporary password
1. ระบบไม่ควรแสดง password เดิม
1. ระบบไม่ควรเก็บ password แบบอ่านกลับได้
#### 7.5 Role / Membership

1. User Account หนึ่งบัญชีสามารถอยู่ได้หลาย Workspace
1. Workspace หนึ่งมีสมาชิกได้หลายคน
1. Workspace Membership ใช้กำหนด role ใน Workspace
1. Project Membership ใช้กำหนด role ใน Project
### สรุปชุดที่ 2

คอมมิตแล้ว:

1. Tenant / Workspace Definition
1. Workspace Registration + Lifecycle
1. Subdomain Strategy
1. Usage Limit Config
1. Demo Workspace vs Production Workspace
1. Workspace Profile
1. Workspace Owner / Member Registration Flow
เอกสารชุดที่ 2 นี้ถือเป็น baseline สำหรับการออกแบบ Workspace Flow และการจัดการสมาชิก ก่อนเข้าสู่ชุดถัดไป เช่น Data Model, Permission Detail, หรือ Project Module Specification
