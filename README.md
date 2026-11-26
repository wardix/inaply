# Inaply — Inaply Cast (Hospitality-Grade TV Casting & In-Room Revenue Platform)

Monorepo untuk platform *hospitality-grade TV casting*, layar sambutan (*welcome screen*), iklan kamar (*TV Ads*), dan komunikasi tamu untuk hotel bintang 3–4.

## Ringkasan Keputusan Arsitektur

| Domain | Keputusan |
|---|---|
| **Target Pasar & Model Bisnis** | Hotel bintang 3–4; Managed Service OPEX per kamar/bulan (*all-in* hardware + CMS + support 24/7) via penjualan langsung & kemitraan ISP/System Integrator |
| **Integrasi Jaringan (Hybrid)** | Mode utama *auto-pairing* tanpa QR/PIN (tamu tetap di Wi-Fi hotel lewat Captive Portal/RADIUS); mode *fallback* QR/PIN sekali pindai saat Wi-Fi hotel dikunci vendor lain |
| **Isolasi Kamar & Discovery** | On-Premise Gateway (`Go` di Linux) menjalankan **mDNS/SSDP Proxy per-kamar + Dynamic Firewall (`nftables`)** |
| **Perangkat Kamar (Receiver)** | STB/Dongle **Android TV / Google TV bersertifikat** (Widevine L1 + Google Cast resmi) dengan aplikasi Kiosk/Launcher (`Kotlin`) |
| **Protokol Casting & Apple** | Google Cast native (Android, iOS app seperti Netflix/YouTube/Spotify, Chrome) + modul receiver AirPlay non-DRM / WebRTC mirroring untuk *screen mirroring* iOS/macOS |
| **Konten Layar & Monetisasi** | Custom Welcome Screen per properti/segmen, TV Ads saat *idle*, Guest Messaging (*broadcast*/per-kamar + *pop-up overlay*), HDMI-CEC auto-switch |
| **Lifecycle & PMS** | Adapter PMS modular + Front Office Web Console bawaan + *auto-expire* jam *check-out* sebagai baseline; konektor API (VHP, Cloudbeds, Opera) bertahap |
| **Control Plane & Fleet** | Cloud Multi-Tenant CMS (`TypeScript` / Next.js + PostgreSQL) + Local Edge Cache di Gateway; penguncian perangkat via **Android Device Owner API** + OTA & Telemetry via Gateway |

## Struktur Monorepo

```text
.
├── docs/
│   ├── PRD.md                 # Product Requirements Document & Go-To-Market
│   └── ARCHITECTURE.md        # Blueprint arsitektur jaringan, gateway, TV app, & cloud
├── contracts/
│   └── api-v1.yaml            # Kontrak API Cloud <-> Edge Gateway <-> TV Receiver <-> PMS
├── services/
│   └── gateway/               # PoC On-Premise Gateway (Go): mDNS/SSDP proxy, session, firewall, portal, edge cache
├── apps/
│   ├── cms/                   # Blueprint & skema database multi-tenant Cloud CMS (TypeScript/PostgreSQL)
│   └── tv-receiver/           # Kerangka aplikasi Kiosk Android TV (Kotlin + Device Owner + HDMI-CEC)
└── deploy/
    ├── docker-compose.yml     # Simulasi / deployment stack lokal
    ├── gateway.service        # Unit systemd untuk On-Premise Gateway Linux
    └── nftables.conf.tmpl     # Template aturan isolasi VLAN/subnet kamar
```

## Menjalankan & Menguji PoC On-Premise Gateway (`services/gateway`)

Prasyarat: Go 1.18+ (`go version`).

```bash
cd services/gateway

# Jalankan seluruh unit & integration test (mDNS filtering, cross-room isolation, portal/PIN pairing, checkout, firewall)
go test -v ./...

# Jalankan Gateway dalam mode dry-run lokal (simulasi hotel tanpa butuh root/nftables)
go run ./cmd/gateway -addr :8080 -mdns-addr 127.0.0.1:5354 -dry-run
```

Contoh halaman demo yang tersedia saat Gateway berjalan:
- Captive Portal (Auto-Pairing): `http://localhost:8080/portal`
- Fallback QR/PIN Pairing: `http://localhost:8080/pair`
- Front Office Mini Console: `http://localhost:8080/fo`
- TV Receiver Backdrop State API: `http://localhost:8080/api/v1/tv/state?device_id=tv-301-main`

## Dokumentasi Lanjutan

- [Product Requirements Document (PRD)](./docs/PRD.md)
- [Arsitektur Sistem & Jaringan](./docs/ARCHITECTURE.md)
- [Kontrak API v1](./contracts/api-v1.yaml)
