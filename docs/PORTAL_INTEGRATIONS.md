# Wi-Fi Controller & Captive Portal Integrations

Inaply Gateway supports zero-friction **Auto-Pairing** via vendor-agnostic `PortalAdapter` webhooks at:

`POST /api/v1/portal/webhook/<vendor>`

All adapters validate the shared secret (`X-Inaply-Token` or `Authorization: Bearer <secret>`) and normalize payloads to `store.PairViaPortal(room, guestName, guestIP, guestMAC)`.

## Supported Vendors

| Vendor | Endpoint | Payload Fields |
| :--- | :--- | :--- |
| **MikroTik Hotspot** | `/api/v1/portal/webhook/mikrotik` | `room`, `guest_name`, `ip`, `mac` |
| **Ubiquiti UniFi** | `/api/v1/portal/webhook/unifi` | `room_number`, `guest_name`, `client_ip`, `client_mac` |
| **Ruijie Cloud/Gateway** | `/api/v1/portal/webhook/ruijie` | `roomNo`, `guestName`, `userIp`, `userMac` |
| **TP-Link Omada** | `/api/v1/portal/webhook/omada` | `room`, `name`, `clientIp`, `clientMac` |
