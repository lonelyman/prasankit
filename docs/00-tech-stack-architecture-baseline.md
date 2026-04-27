# ชุดที่ 0: Tech Stack & Architecture Baseline

Multi-Tenant SaaS Project Management Platform

Baseline สำหรับการออกแบบระบบ - ไม่ใช่เอกสาร implementation

## 1. เป้าหมายของชุดที่ 0

ชุดนี้ใช้เป็นจุดตกลงร่วมกันก่อนเข้าสู่การออกแบบระบบในชุดถัดไป โดยเน้นล็อกแนวทางเทคโนโลยี สถาปัตยกรรม และขอบเขตการ deploy เบื้องต้น

- ล็อก Tech Stack หลักของระบบ
- ล็อก Version Baseline ที่จะใช้เป็นกรอบอ้างอิง
- ล็อกแนวทาง Multi-Tenant
- ล็อกแนวทาง Demo Server สำหรับให้ลูกค้าทดลอง
- ลดความกำกวม เช่น “ตัวนี้หรืออีกตัว” ให้เหลือเป็นตัวเลือกที่ตกลงกันแล้ว
## 2. ประเภทของระบบ

ระบบนี้ออกแบบเป็น Multi-Tenant SaaS Project Management Platform

- มีเว็บหลักสำหรับให้องค์กร / หน่วยงาน / ทีม สมัครใช้งาน
- เมื่อลงทะเบียนแล้ว ระบบสร้างพื้นที่ใช้งานแยกให้แต่ละองค์กร
- แต่ละองค์กรเข้าใช้งานผ่าน subdomain ของตัวเอง
- ข้อมูลของแต่ละองค์กรต้องถูกแยกด้วย tenant_id
## 3. Tech Stack ที่ยืนยัน

| Layer | Technology / Version | หน้าที่หลัก |
| --- | --- | --- |
| Frontend | Next.js 16.x, React 19.x, TypeScript 6.x | Landing, Register, Login, Dashboard, Workspace UI |
| UI / Form | Tailwind CSS 4.x, shadcn/ui, React Hook Form 7.x, TanStack Query 5.x | UI components, form handling, data fetching |
| Runtime | Node.js 24 LTS | ใช้รันและ build ฝั่ง Next.js |
| Backend API | Go 1.26.x, Fiber v3.x | REST API, business logic, tenant isolation |
| ORM / Driver | GORM v2, pgx v5 | เข้าถึง PostgreSQL และจัดการ query layer |
| Database | PostgreSQL 18.x | ฐานข้อมูลหลักของระบบ |
| Cache / Session | Redis 8.x | session, cache, rate limit, lock, queue ในอนาคต |
| File Storage | MinIO | เก็บไฟล์ เอกสาร รูปภาพ และไฟล์แนบ |
| Infrastructure | Docker Engine 29.x, Docker Compose v5.x | ควบคุม environment และ deployment |
| Reverse Proxy | Traefik 3.6.x | routing, wildcard subdomain, SSL |
| OS | Ubuntu Server 24.04 LTS | ระบบปฏิบัติการหลักของ server |

## 4. Architecture Direction

แนวทางหลักคือแยก Frontend และ Backend ชัดเจน

- app-web/ = Next.js สำหรับ UI และ tenant workspace
- app-api/ = Go Fiber สำหรับ API และ business logic
- PostgreSQL = source of truth ของข้อมูลระบบ
- Redis = session/cache/lock
- MinIO = object storage ที่ควบคุมเอง
- Traefik = reverse proxy และจัดการ wildcard subdomain
## 5. Multi-Tenant Strategy

เลือกใช้ Shared Database + tenant_id

- ใช้ PostgreSQL ชุดเดียวในช่วงเริ่มต้น
- ทุก table สำคัญต้องมี tenant_id
- ทุก API ที่เป็นข้อมูลขององค์กรต้องตรวจ tenant_id เสมอ
- เหมาะกับ MVP, Demo และ SaaS ช่วงเริ่มต้น เพราะดูแลง่ายกว่าแยก database รายองค์กร
## 6. Auth Strategy

เลือกใช้ Session-based Auth

- Session เก็บใน Redis
- ใช้ httpOnly Cookie
- ใช้ Secure Cookie ใน production
- SameSite=Lax
- ไม่เก็บ token สำคัญไว้ใน localStorage
- MVP ไม่ใช้ token-based auth เป็น auth หลัก
- auth_sessions ใน database ใช้เก็บ session metadata / revoke history / device history เท่านั้น
## 7. File Storage Strategy

เลือกใช้ MinIO เป็น Object Storage หลัก

- ควบคุมไฟล์เองทั้งหมด
- ลดต้นทุนการเช่า Object Storage เพิ่ม
- รองรับ S3-compatible API
- เหมาะกับกรณีติดตั้ง production แยกให้ลูกค้า
- ใช้เก็บไฟล์เอกสาร ไฟล์แนบ task/project รูปภาพ และไฟล์ export บางประเภท
## 8. Demo Server Strategy

| รายการ | Baseline Demo |
| --- | --- |
| Purpose | Demo / Trial / Sales Sandbox |
| CPU | 1 vCore |
| Memory | 2 GB RAM |
| Storage | 20 GB SSD |
| Network | 100 Mbps |
| แนวคิด | ใช้ดึงคนมาลองระบบก่อน ไม่ใช่ production จริง |

เมื่อลูกค้าสนใจใช้งานจริง จะประเมินและติดตั้ง production environment แยกตามสเปคที่ตกลงกับลูกค้า

## 9. Production Direction

Baseline production เริ่มต้นที่แนะนำเมื่อใช้งานจริง

| ขนาดระบบ | สเปคแนะนำ |
| --- | --- |
| เริ่มใช้งานจริง | 4 vCPU / 8 GB RAM / 160-200 GB SSD |
| องค์กรใหญ่ขึ้น | 8 vCPU+ / 16 GB RAM+ / 500 GB SSD+ |

## 10. Decision ที่ล็อกแล้ว

- Frontend: Next.js 16.x + React 19.x + TypeScript 6.x
- Backend: Go 1.26.x + Fiber v3.x
- ORM: GORM v2
- Database: PostgreSQL 18.x
- Cache / Session: Redis 8.x
- File Storage: MinIO
- Reverse Proxy: Traefik 3.6.x
- Auth: Session-based Auth + Redis Session Store + httpOnly Cookie
- Multi-Tenant: Shared Database + tenant_id
- Deploy: Docker Compose บน Ubuntu Server 24.04 LTS
## 11. หมายเหตุ Version

Version ในเอกสารนี้เป็น baseline ณ วันที่ 25 เมษายน 2026 ใช้หลัก “latest stable ภายใน major version ที่ตกลงกัน” และไม่ใช้ beta / rc / canary เป็น baseline หลัก

## 12. แหล่งอ้างอิง Version

- Next.js: npm package latest 16.2.4
- Node.js: v24 เป็น LTS branch ปัจจุบัน
- Go: download page แสดง stable version go1.26.2
- Fiber: v3 ต้องใช้ Go 1.25+
- PostgreSQL: release notes แสดง 18.3 เป็น latest release line
- TypeScript: npm package latest 6.0.3
- Tailwind CSS: npm package latest 4.2.4
- Redis: ใช้ Redis Open Source 8.x stable line
