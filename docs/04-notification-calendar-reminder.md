# Project Management Platform

ชุดที่ 4: Notification / Calendar / Reminder Concept

> **Status: Forward-looking design intent (MVP-1+).** Notification engine, Calendar และ Reminder ยังไม่ implement ใน MVP-0 (scope guard ระบุชัดใน [docs/00-context-guard.md](00-context-guard.md)). เอกสารนี้คือ design intent สำหรับ phase ถัดไป.

สรุปแนวทางระบบแจ้งเตือน ปฏิทิน และ Reminder สำหรับ MVP

| หัวข้อ | Decision |
| --- | --- |
| Notification Channel | ใช้ In-app Notification และ Email Notification |
| Notification Event | รองรับ Task, Timeline, Announcement, Project Team และ Workspace Event |
| Notification Recipient | กำหนดผู้รับจากความสัมพันธ์กับข้อมูล เช่น Assignee, Watcher, Role |
| Notification Timing | มี default timing และรองรับ Remind Before เฉพาะรายการสำคัญ |
| Calendar | แสดงวันสำคัญจาก Task และ Timeline ผ่าน Month View / List View |
| Read / History | รองรับ Unread, Read, Archived, Unread Count และ Required Announcement |
| Settings | ผู้ใช้เปิด/ปิด Email ทั่วไปได้ แต่ In-app และประกาศสำคัญปิดไม่ได้ |

## 1. Notification Channel

### Decision

ระบบแจ้งเตือน MVP ใช้ 2 ช่องทาง

### ช่องทาง

- In-app Notification = ช่องทางหลักของระบบ
- Email Notification = ใช้สำหรับเรื่องสำคัญ
### ยังไม่ทำใน MVP

- LINE
- Slack
- Microsoft Teams
- Mobile Push Notification
- SMS
## 2. Notification Event

### Event ที่รองรับใน MVP

- Task Event
- Timeline Event
- Announcement Event
- Project Team Event
- Workspace Event
### ช่องทางตามประเภท

- In-app Notification = ค่าเริ่มต้นของทุก event
- Email Notification = ใช้เฉพาะ event สำคัญ
### Event สำคัญที่ควรส่ง Email ได้

- ถูกเชิญเข้า Workspace
- ถูกมอบหมายงาน
- งานเลยกำหนด
- Timeline สำคัญใกล้ถึง
- ประกาศ Important / Critical
- Workspace Pending Deletion
### Priority / Tag

Notification ต้องพิจารณาจาก Event Type, Priority, Tag และ Due Date / Planned Date

### Remark

MVP ยังไม่ทำ Notification Rule Builder เต็มรูปแบบ แต่ต้องออกแบบให้ขยายเป็น rule-based notification ได้ในอนาคต

## 3. Notification Recipient

### หลักการ

Notification Recipient ใน MVP ใช้แบบพอดี ยังไม่ทำ rule ซับซ้อน

### ผู้รับตามความสัมพันธ์

- Assignee
- Watcher / Participant
- Project Owner / Project Manager
- Finance Role
- Workspace Owner / Admin
### Actor

Actor ไม่ควรได้รับแจ้งเตือนจาก action ที่ตัวเองเพิ่งทำ

### MVP Recipient Pattern

- Task แจ้ง Assignee / Watcher
- งานเลยกำหนด แจ้ง Assignee และ Project Owner / Manager
- Timeline สำคัญ แจ้งผู้เกี่ยวข้อง
- Finance แจ้ง Finance Role / Owner ตามสิทธิ์
- Workspace Pending Deletion แจ้ง Workspace Owner / Admin
## 4. Notification Timing

### ค่า Default กลาง

- ก่อนถึงกำหนด 1 วัน
- วันครบกำหนด
- เลยกำหนด 1 วัน
### ใช้กับ

- Task Due Date
- Timeline Planned Date
- Meeting / Appointment
- Deliverable Scheduled Date
### ช่องทางตามความสำคัญ

- เรื่องปกติ = In-app Notification
- เรื่องสำคัญ / Critical / เลยกำหนด = In-app + Email
### Remind Before

บางรายการสามารถกำหนดแจ้งเตือนล่วงหน้าเองได้ เช่น วันส่งงาน วันเสนอราคา วันยื่นประมูล วันประชุมสำคัญ วันสิ้นสุด MA

### รูปแบบ MVP

Remind Before = เลือกได้ 1 ค่า เช่น 1 วัน, 3 วัน, 7 วัน, 14 วัน, 30 วัน

### ยังไม่ทำใน MVP

- ตั้ง reminder หลายรอบต่อรายการแบบละเอียด
- Rule Builder
- แจ้งเตือนซ้ำทุกวันจนกว่าจะปิด
- ปรับ reminder รายคนละเอียด
## 5. Calendar Concept

### Decision

Calendar MVP ใช้แสดงวันสำคัญจาก Task และ Timeline มี Month View และ List View มี filter ตามประเภท และยังไม่ sync กับ Google Calendar / Outlook

### แหล่งข้อมูลหลัก

- Task
- Timeline
### มุมมอง Calendar

- Month View
- List View
### Filter ตั้งต้น

- All
- Task
- Timeline
- Meeting / Appointment
- Deliverable
- Overdue
### Remark

Deliverable และ Meeting / Appointment เป็น Timeline Item Type จึงแสดงใน Calendar ผ่านข้อมูล Timeline ได้

## 6. Notification Read / History

### Notification ทั่วไป

Status = Unread, Read, Archived

### Action

- Mark as read
- Mark all as read
- Archive notification
- Open related item
- Notification History
- Unread Count
### Unread Count

ใช้แสดงตัวเลขแจ้งเตือนที่ยังไม่ได้อ่าน เช่น ไอคอนกระดิ่ง: 5

### Required Announcement

ประกาศสำคัญที่ต้องบังคับให้ผู้ใช้กดรับทราบ โดยตั้งค่า Require Acknowledgement = true

### Mark as read vs Acknowledge

- Mark as read ใช้กับ notification ทั่วไป
- Acknowledge ใช้กับประกาศสำคัญที่ต้องยืนยันการรับทราบ
- Mark all as read ต้องไม่ถือว่าเป็นการ Acknowledge ประกาศสำคัญ
### ประวัติที่ต้องเก็บ

- ใครรับทราบ
- รับทราบเมื่อไหร่
- รับทราบประกาศไหน
## 7. Notification Settings

### Decision

- ผู้ใช้เปิด/ปิด Email Notification ของตัวเองได้
- In-app Notification เป็นค่าเริ่มต้น และปิดทั้งหมดไม่ได้
- Required Announcement ปิดไม่ได้
- Security / Account Event ปิด Email ไม่ได้
### Email ที่ผู้ใช้ปิดได้

- Task ทั่วไป
- Timeline ทั่วไป
- ประกาศทั่วไป
### Email ที่ปิดไม่ได้

- Workspace Pending Deletion
- Required Announcement
- Security / Account Event
## 8. สรุปสถานะชุดที่ 4

- Notification Channel
- Notification Event
- Notification Recipient
- Notification Timing
- Calendar Concept
- Notification Read / History
- Notification Settings
เอกสารชุดนี้ใช้เป็นแนวทางสำหรับ MVP โดยเน้นระบบแจ้งเตือนและปฏิทินแบบพอดี ไม่ทำให้ระบบบวม แต่ยังออกแบบให้ขยายเป็น rule-based notification และ calendar integration ได้ในอนาคต
