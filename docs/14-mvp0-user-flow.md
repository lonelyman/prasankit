# MVP-0 User Flow

เอกสารนี้ใช้กำหนด flow แรกที่ต้องทำให้ใช้งานได้จริงก่อนเริ่มขยาย module อื่น

## เป้าหมาย

MVP-0 ต้องตอบคำถามเดียวให้ได้:

ผู้ใช้สมัคร workspace แล้วสามารถเข้าไปสร้าง project, เพิ่มทีม, สร้าง task และเห็น activity พื้นฐานได้หรือไม่

## Flow หลัก

```text
Public App
-> Register Account
-> Verify Email
-> Login
-> Create Workspace
-> Workspace Dashboard
-> Create Project
-> Project Detail
-> Add Project Member
-> Create Task
-> Move Task Status
-> Upload File
-> View Activity Log
```

## 1. Register Account

ผู้ใช้สมัครบัญชีกลางก่อน:

- Owner Email
- Password

ระบบทำงาน:

- สร้าง user account
- สร้าง auth identity แบบ email/password
- ส่ง email verification

ผลลัพธ์ที่ต้องได้:

- account ถูกสร้างเป็น `pending_verification`
- หลัง verify email แล้ว account เป็น `active`

## 2. Create Workspace

หลัง verify email และ login สำเร็จ ผู้ใช้กรอกข้อมูลขั้นต่ำ:

- Workspace Name
- Workspace Slug
- Contact Email

ระบบทำงาน:

- สร้าง workspace
- สร้าง workspace membership ให้ owner
- สร้าง tenant_id ใน backend
- seed ค่าเริ่มต้นที่จำเป็นในรอบถัดไป

ผลลัพธ์ที่ต้องได้:

- Workspace ถูกสร้างในสถานะใช้งานได้จาก account ที่ verified แล้ว
- Owner เข้า workspace ของตัวเองได้

## 3. Login เข้า Workspace

ผู้ใช้เข้า:

```text
/w/{workspaceSlug}/login
```

ระบบทำงาน:

- resolve workspace จาก slug
- ตรวจ user account
- ตรวจ email verified
- ตรวจ workspace membership
- สร้าง session ใน Redis
- set httpOnly Cookie

ผลลัพธ์ที่ต้องได้:

- ผู้ใช้เข้า Workspace Dashboard ได้
- tenant_id ถูก resolve ฝั่ง backend เท่านั้น

## 4. Workspace Dashboard

รอบ MVP-0 ให้ dashboard เรียบง่าย:

- รายการ Project ของ workspace
- ปุ่ม Create Project
- My Tasks แบบพื้นฐาน
- Activity ล่าสุดแบบพื้นฐาน

ยังไม่ต้องมี report, chart, finance summary หรือ notification เต็มระบบ

## 5. Create Project

ข้อมูลขั้นต่ำ:

- Project Name
- Project Type

ระบบ auto set:

- Project Status = Draft
- Priority = Medium
- Created By = current user
- Project Owner = current user
- Task Workflow seed = default workflow

ผลลัพธ์ที่ต้องได้:

- Project ถูกสร้างใต้ workspace ปัจจุบัน
- คนสร้างเป็น Project Owner
- เปิด Project Detail ได้

## 6. Project Detail

Tabs ที่ต้องมีใน MVP-0:

- Overview
- Tasks
- Team
- Documents
- Activity

Tabs ที่ยังไม่ทำใน MVP-0:

- Finance
- Timeline / Deliverable แบบเต็ม
- Announcements
- Reports

## 7. Add Project Member

Project Owner เพิ่ม member จาก Workspace Members

ข้อมูลที่ต้องเลือก:

- Member
- Project Role
- Project Position ถ้ามี

กติกา:

- Project ต้องมี Project Owner อย่างน้อย 1 คน
- ห้าม remove Project Owner คนสุดท้าย
- คนที่ไม่มี active workspace membership ห้ามถูกเพิ่มเข้า project ใหม่

## 8. Task Board

ระบบต้องมี default status:

- Backlog
- To Do
- In Progress
- Review
- Testing
- Blocked
- Done
- Cancelled

Task ข้อมูลขั้นต่ำ:

- Task Title
- Task Type
- Status
- Assignee
- Due Date optional

ความสามารถ MVP-0:

- สร้าง task
- แก้ task
- ย้าย status
- assign/reassign
- comment แบบพื้นฐาน
- checklist แบบพื้นฐาน

## 9. Basic File Upload

ใช้ Project File Center แบบพื้นฐาน:

- สร้าง folder
- upload file
- เปิดไฟล์ผ่าน signed URL
- ผูกไฟล์กับ task ได้ผ่าน relation

กติกาสำคัญ:

- ห้ามเก็บ full public URL ใน database
- backend ต้องตรวจสิทธิ์ก่อนสร้าง signed URL
- delete จาก UI เป็น soft delete ก่อน

## 10. Basic Activity Log

MVP-0 ต้อง log action สำคัญ:

- workspace created
- user logged in
- project created
- project member added
- task created
- task status changed
- file uploaded

ข้อมูล log ขั้นต่ำ:

- tenant_id
- workspace_id
- project_id ถ้ามี
- actor_user_account_id
- action
- resource_type
- resource_id
- old_value_json
- new_value_json
- created_at

## Definition of Done

MVP-0 ถือว่าเสร็จเมื่อ:

1. สมัคร workspace ได้
2. ยืนยัน email ได้
3. login เข้า workspace ได้
4. สร้าง project ได้
5. เพิ่ม member เข้า project ได้
6. สร้าง task ได้
7. ย้าย task status ได้
8. upload file ได้
9. เปิดไฟล์ผ่าน signed URL ได้
10. เห็น activity log พื้นฐานได้

## ยังไม่ทำ

- Finance
- Deliverable เต็มรูปแบบ
- Notification engine
- Report dashboard
- Billing
- SSO
- Mobile app
- AI assistant
