# Prasankit Context Guard

This file is the short source of truth to prevent scope drift.

## Project

Prasankit is a multi-tenant project/workspace management platform.

## Current Target

Build MVP-0 first:

- Auth
- Workspace registration and login
- Tenant resolution
- Project profile
- Project team
- Task board
- Basic file upload
- Basic activity log

## Stack

- app-web/: Next.js + React + TypeScript
- app-api/: Go Fiber + GORM
- Database: PostgreSQL
- Session/cache: Redis
- File storage: MinIO
- Reverse proxy: Traefik
- Deploy: Docker Compose

## Hard Rules

1. tenant_id must come from backend Tenant Context.
2. Frontend must not send tenant_id as a trusted value.
3. Workspace API must not include tenant_id in URL.
4. API prefix is /api/v1.
5. Success response root key is data.
6. Error response root key is error.
7. List response uses data.items and data.pagination.
8. Every list API must use pagination.
9. Every workspace/project query must filter by tenant_id.
10. All sensitive actions must pass Permission Guard.
11. File URL must not be stored as a full public URL.
12. File access must go through backend permission check and signed URL.
13. Auth uses Session-based Auth with Redis and httpOnly Cookie.
14. Reverse proxy is Traefik.

## Scope Guard

Do not implement these in MVP-0:

- Finance
- Deliverable / Timeline full workflow
- Notification engine
- Reports
- Billing
- Custom domain
- SSO
- Mobile app
- AI assistant
- E-signature
- OCR
- Advanced approval workflow
