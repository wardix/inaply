# Cloud Multi-Tenant CMS (`apps/cms`)

Kerangka *Control Plane* untuk mengelola properti hotel, mitra ISP/System Integrator, kampanye *TV Ads*, *Guest Messaging*, *Front Office*, *PMS Connectors*, serta *Fleet Monitoring* 24/7.

## Stack Target
- **Framework:** Next.js (App Router) + TypeScript
- **Database:** PostgreSQL (`db/schema.sql`) + Redis
- **Object Storage:** S3 / Cloudflare R2 (untuk video & gambar *TV Ads* serta APK OTA)

## Struktur Folder
- `db/schema.sql` — Skema PostgreSQL multi-tenant (`partners -> properties -> gateways -> rooms -> receivers`, `ad_campaigns`, `guest_messages`, `guest_stays`, `billing_subscriptions`).
- `src/types/contracts.ts` — Tipe TypeScript yang selaras dengan `contracts/api-v1.yaml`.
