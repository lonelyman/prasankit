# ชุดที่ 11
Docker / Deploy / Domain / SSL

Prasankit Project Platform

> **Status: Mixed — local dev (implemented) + production deploy (forward-looking).** Local dev stack ผ่าน `docker-compose.yml` + `make dev-up` ทำงานจริงแล้ว. Traefik production config, wildcard subdomain, Let's Encrypt, backup automation ฯลฯ ยังเป็น plan.

เอกสารนี้สรุปแนวทาง Docker, deployment, domain, SSL, backup, logging และ production readiness สำหรับระบบ Prasankit โดยยึดแนวคิด Data Safety First และเริ่มจาก Docker Compose บน VM/Server เดียวก่อน

## สารบัญ

11.1 Environment Strategy

11.2 Docker Compose Structure

11.3 Environment Variables / Config

11.4 Data Protection Rule

11.5 Domain / Subdomain / Reverse Proxy / SSL

11.6 Backup / Restore Strategy

11.7 Logging / Monitoring / Maintenance

11.8 Deploy / Update Flow

11.9 MVP vs Production Deployment Scope

## 11.1 Environment Strategy

แยก environment ตามระดับการใช้งาน เพื่อไม่ให้ข้อมูลทดลองปนกับข้อมูลจริง และทำให้การ deploy/backup ควบคุมได้ง่าย

| Environment | หน้าที่ | แนวทาง Tenant |
| --- | --- | --- |
| Local Development | ใช้สำหรับพัฒนาในเครื่อง dev | Path-based: http://localhost:3005/w/company-a-demo |
| Demo Server | ใช้สำหรับทดลอง / นำเสนอ / ให้ลูกค้าลองใช้ | Subdomain: company-a-demo.demo.yourdomain.com |
| Production Server | ใช้สำหรับลูกค้าจริง | Subdomain production และแยก server/environment จาก demo |

Decision Final

- แยก Local Development, Demo Server และ Production Server
- Demo กับ Production ควรแยก environment/server กัน เพื่อไม่ให้ข้อมูลทดลองปนกับข้อมูลจริง
- Local ใช้ path-based tenant เพื่อ dev ง่ายและไม่ต้องตั้ง DNS
## 11.2 Docker Compose Structure

ระบบเริ่มต้นใช้ Docker Compose เพื่อให้ดูแลง่ายบน VM/Server เดียว และสามารถยกระดับ production ภายหลังได้

| Service | ชื่อที่ล็อก | หน้าที่ |
| --- | --- | --- |
| Web | prasankit-web | Next.js frontend: Public App, Workspace Portal, Project Detail, Owner Console, Platform Admin |
| API | prasankit-api | Go API: Auth, Workspace, Project, Task, File, Platform API |
| PostgreSQL | prasankit-pgsql | Database หลักของระบบ |
| Redis | prasankit-redis | Rate limit, cache, session helper, queue เบื้องต้น |
| MinIO | prasankit-minio | Object storage สำหรับไฟล์และเอกสาร |
| Proxy | prasankit-proxy | Reverse proxy, subdomain routing, SSL termination |

ชื่อ volume / network ที่กำหนด

```text
prasankit_pgsql_data
prasankit_redis_data
prasankit_minio_data
prasankit_proxy_certs
prasankit_logs
prasankit_network
```

แนวทางที่ล็อก

- ใช้ docker-compose จากระบบเดิมเป็นฐานได้ เพราะมี healthcheck และ dependency ชัดเจน
- ปรับ prefix เป็น prasankit ทั้ง service, container, volume และ network
- ออกแบบแบบ Data Safety First โดยให้ PostgreSQL / Redis / MinIO มี persistent storage ที่ชัดเจน
## 11.3 Environment Variables / Config

.env เป็น template/guideline ไม่ใช่ค่าจริงสุดท้าย ค่า secret ต้องเปลี่ยนก่อนใช้งานจริงและห้าม commit ขึ้น repository

| กลุ่ม Config | ตัวอย่างค่า/หน้าที่ |
| --- | --- |
| API / App | API_ENV, API_PORT, API_TIMEZONE, API_CORS_ALLOWED_ORIGINS |
| Web / Frontend | WEB_PORT, NEXT_PUBLIC_API_URL, INTERNAL_API_URL, ALLOWED_DEV_ORIGINS |
| Tenant / Domain | TENANT_RESOLUTION_MODE, DEV_TENANT_PATH_PREFIX, APP_WEB_URL, APP_DOMAIN |
| PostgreSQL | POSTGRES_PRIMARY_HOST, POSTGRES_PRIMARY_PORT, POSTGRES_PRIMARY_USER, POSTGRES_PRIMARY_NAME |
| Redis | REDIS_HOST, REDIS_PORT, REDIS_PASSWORD, REDIS_DB |
| Auth / Session | SESSION_COOKIE_NAME, SESSION_SECRET, SESSION_TTL, COOKIE_SECURE, COOKIE_SAMESITE |
| Mailer | MAILER_HOST, MAILER_PORT, MAILER_USERNAME, MAILER_FROM_ADDRESS |
| MinIO / Storage | MINIO_ROOT_USER, STORAGE_ENDPOINT, STORAGE_BUCKET, STORAGE_PRESIGN_TTL |
| Cleanup / Retention | WORKSPACE_PENDING_DELETION_DAYS, FILE_SOFT_DELETE_RETENTION_DAYS |

Config guideline สำคัญ

- NEXT_PUBLIC_API_URL ใช้สำหรับ browser
- INTERNAL_API_URL ใช้สำหรับ container-to-container
- POSTGRES_EXTERNAL_PORT / REDIS_EXTERNAL_PORT ใช้ต่อจากเครื่อง dev เช่น DBeaver/TablePlus
- STORAGE_PUBLIC_BASE_URL เป็น config ได้ แต่ห้ามเก็บ full file URL ลง database
## 11.4 Data Protection Rule

หัวข้อนี้เป็นกติกาสำคัญเพื่อป้องกันข้อมูลหายจาก lifecycle ของ container, การ prune ผิด, การลบ volume/data ผิด และการ deploy ที่ไม่มี backup

Decision Final

- ใช้ bind mount สำหรับข้อมูลสำคัญ
- ห้ามใช้ down -v / prune --volumes บน demo/production
- ใช้ deploy script แทนการพิมพ์คำสั่งเอง
- ต้องมี backup PostgreSQL + MinIO
- ควรมี VM/server snapshot ก่อน deploy สำคัญ
Path ที่แนะนำ

```text
prasankit/
├─ docker-compose.yml
├─ .env
├─ app-api/
├─ app-web/
├─ data/
│ ├─ postgres/
│ ├─ redis/
│ └─ minio/
├─ logs/
│ ├─ app-api/
│ └─ proxy/
├─ backups/
│ ├─ postgres/
│ └─ minio/
└─ deploy/
└─ traefik/
```

| ใช้ได้ตามปกติ | ห้ามใช้บน Demo / Production |
| --- | --- |
| docker compose down | docker compose down -v |
| docker compose up -d | docker volume prune |
| docker compose restart | docker system prune --volumes |
| docker compose stop | docker volume rm |
|  | rm -rf ./data |

หมายเหตุสำคัญ

- มี volume แล้วข้อมูลไม่ควรหายจาก lifecycle ปกติของ container
- แต่ข้อมูลยังหายได้ถ้ามีการลบ volume หรือ folder data โดยตรง
- bind mount เช่น ./data/postgres และ ./data/minio ช่วยให้เห็นตำแหน่งข้อมูลจริงชัดและ backup ง่ายกว่า named volume ใน demo/production
## 11.5 Domain / Subdomain / Reverse Proxy / SSL

| Environment | รูปแบบ URL | หมายเหตุ |
| --- | --- | --- |
| Local | http://localhost:3005/w/company-a-demo | ใช้ path-based tenant |
| Demo | https://company-a-demo.demo.yourdomain.com | แนะนำ wildcard *.demo.yourdomain.com |
| Production | https://company-a.yourdomain.com | แยก environment/server จาก Demo |

Routing หลัก

```text
/ -> prasankit-web
/api/v1/* -> prasankit-api
*.demo.yourdomain.com -> prasankit-web
```

Tenant Resolution

```text
company-a-demo.demo.yourdomain.com
-> slug = company-a-demo
-> tenant_id / workspace_id
```

SSL

- ใช้ Let’s Encrypt
- ถ้าใช้ wildcard subdomain แนะนำ DNS Challenge
- Reverse proxy ล็อกใช้ Traefik เพื่อให้ตรงกับ Tech Stack Baseline และรองรับ wildcard subdomain / SSL
## 11.6 Backup / Restore Strategy

PostgreSQL และ MinIO ต้อง backup คู่กัน เพราะ DB เก็บ metadata ส่วน MinIO เก็บ object จริง ถ้า backup ไม่สัมพันธ์กันจะทำให้ไฟล์เปิดไม่ได้หรือมีไฟล์ที่ DB ไม่รู้จัก

| ระดับ | PostgreSQL | MinIO | Server/VM |
| --- | --- | --- | --- |
| MVP / Demo | pg_dump daily, pg_dump ก่อน deploy ใหญ่, pg_dump ก่อน cleanup/hard delete | sync/copy daily และ backup คู่กับ PostgreSQL | snapshot ก่อน deploy สำคัญ |
| Production Option | pgBackRest, WAL Archive, PITR, Full/Differential/Incremental, Encryption, Offsite Backup | Sync ไป backup storage แยก หรือ MinIO replication | VM backup รายวัน, snapshot, retention policy |

pgBackRest

- ใช้เมื่อระบบเข้าสู่ Production จริง มีข้อมูลลูกค้าจริง และต้องการกู้ย้อนหลังเป็นช่วงเวลาได้
- ช่วยรองรับ WAL Archive และ Point-in-Time Recovery หรือ PITR
- ยังไม่บังคับใช้ใน MVP/Demo เพื่อไม่เพิ่มความซับซ้อนเร็วเกินไป
Retention guideline

- Daily backup เก็บ 7-14 วัน
- Weekly backup เก็บ 4-8 สัปดาห์
- Monthly backup เก็บ 3-6 เดือน
- Demo อาจ retention สั้นกว่า Production
Restore test

- ต้องทดสอบ restore PostgreSQL
- ต้องทดสอบ restore MinIO
- ต้องเปิดระบบแล้วไฟล์ยังเปิดได้
- Backup ที่ไม่เคย restore ถือว่ายังไม่มั่นใจ
## 11.7 Logging / Monitoring / Maintenance

| Log | ใช้ตรวจอะไร |
| --- | --- |
| API Log | request_id, method, path, status_code, duration_ms, user_account_id, tenant_id/workspace_id, error |
| Web Log | Next.js build/runtime/route/SSR/API fetch error |
| Proxy / Access Log | domain/subdomain, path, status code, response time, client IP |
| Database / Storage Service Log | PostgreSQL, Redis, MinIO health/error |

ห้าม log ข้อมูล sensitive

- password
- session id / session cookie
- token / secret ที่ใช้ยืนยันตัวตนหรือ reset password
- reset token
- signed URL เต็ม
- sensitive config
Maintenance task

- backup database
- backup/sync MinIO
- cleanup file ที่ soft deleted เกิน retention
- cleanup workspace demo ที่หมดอายุ
- cleanup log เก่า
- rotate backup
- ตรวจ disk usage
Production Option

- Prometheus / Grafana
- Loki / ELK / OpenSearch
- Uptime Kuma
- Alert ผ่าน Email / Line / Slack
- PostgreSQL slow query monitoring
## 11.8 Deploy / Update Flow

Flow ที่ล็อก

- ตรวจสถานะ service ปัจจุบัน
- Backup PostgreSQL
- Backup / Sync MinIO
- Pull code หรือ update image
- Build / Pull container
- Run migration ถ้ามี
- Restart service
- ตรวจ healthcheck
- ตรวจ log

Fresh server bootstrap สำหรับ environment ใหม่:

```text
1. Copy .env.example to .env and edit real environment values
2. Start PostgreSQL / Redis / MinIO
3. Run goose migrations
4. Start or rebuild API
5. Check /api/v1/health/ready
```

ห้ามถือว่า API startup แทน migration ได้เอง เพราะ schema ต้องถูกจัดการด้วย goose SQL migration แยกจาก GORM

Local/dev shortcut:

```bash
make dev-up
```
- ทดสอบเปิด web / api / file
คำสั่งที่ใช้ได้

```text
docker compose pull
docker compose build
docker compose up -d
docker compose ps
docker compose logs -f prasankit-api
```

Migration Rule

- ก่อน migration ต้อง backup DB
- migration ต้องรันจาก version control
- ห้ามแก้ schema manual โดยไม่มีบันทึก
- ถ้า migration เสี่ยง ต้องมี rollback plan
Decision Final

- ก่อน deploy ต้อง backup PostgreSQL และ MinIO
- ใช้ docker compose up -d เป็นหลัก
- ใช้ deploy script เพื่อลดการพิมพ์คำสั่งผิด
- ห้ามใช้คำสั่งที่ลบ volume/data บน demo/production
- ต้องตรวจ healthcheck/log หลัง deploy
## 11.9 MVP vs Production Deployment Scope

| MVP / Demo | Production Option | ยังไม่จำเป็นใน MVP |
| --- | --- | --- |
| Docker Compose บน VM/Server เดียว<br>Web + API + PostgreSQL + Redis + MinIO<br>Reverse Proxy + SSL<br>Bind mount data ชัดเจน<br>pg_dump + MinIO sync<br>Healthcheck + deploy script + log พื้นฐาน | แยก Demo/Production environment<br>pgBackRest + WAL Archive + PITR<br>Offsite Backup<br>Monitoring / Alert<br>Log aggregation<br>SSL/Domain management ที่เป็นระบบ<br>VM backup / snapshot policy<br>Security hardening | Kubernetes<br>Multi-node cluster<br>Auto scaling<br>Blue/Green deployment<br>CDN เต็มรูปแบบ<br>Observability stack เต็มระบบ |

Decision รวมของชุดที่ 11

- Deployment MVP ใช้ Docker Compose บน VM/Server เดียวก่อน
- เน้นความง่ายในการดูแล ความปลอดภัยของข้อมูล และ backup/restore
- Production ค่อยยกระดับ backup, monitoring, security, offsite backup และ PostgreSQL PITR
- ยังไม่จำเป็นต้องใช้ Kubernetes หรือ cluster ตั้งแต่แรก
## Appendix A: docker-compose guideline snippet

```text
services:
prasankit-pgsql:
image: postgres:18-alpine
container_name: prasankit_pgsql
environment:
POSTGRES_USER: ${POSTGRES_PRIMARY_USER}
POSTGRES_PASSWORD: ${POSTGRES_PRIMARY_PASSWORD}
POSTGRES_DB: ${POSTGRES_PRIMARY_NAME}
ports:
- "${POSTGRES_EXTERNAL_PORT}:5432"
volumes:
- ./data/postgres:/var/lib/postgresql/data
healthcheck:
test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_PRIMARY_USER} -d ${POSTGRES_PRIMARY_NAME}"]
interval: 5s
timeout: 5s
retries: 5
prasankit-redis:
image: redis:7.4-alpine
container_name: prasankit_redis
command: redis-server --requirepass ${REDIS_PASSWORD} --appendonly yes
ports:
- "${REDIS_EXTERNAL_PORT}:6379"
volumes:
- ./data/redis:/data
prasankit-minio:
image: minio/minio:latest
container_name: prasankit_minio
command: server /data --console-address ":9001"
volumes:
- ./data/minio:/data
prasankit-api:
build: ./app-api
container_name: prasankit_api
env_file:
- .env
depends_on:
prasankit-pgsql:
condition: service_healthy
prasankit-redis:
condition: service_started
prasankit-minio:
condition: service_started
prasankit-web:
build: ./app-web
container_name: prasankit_web
env_file:
- .env
depends_on:
- prasankit-api
```

หมายเหตุ: snippet นี้เป็น guideline ต้องปรับ image tag, path, healthcheck และ proxy ตาม environment จริง
