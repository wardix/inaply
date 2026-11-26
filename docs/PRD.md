# Product Requirements Document (PRD) — Inaply Cast (Inaply Platform)

**Versi:** 1.0 (MVP Commercial Baseline)  
**Target Segmen:** Hotel Bintang 3–4, Boutique Hotel, Resort & Villa (Indonesia & Asia Tenggara)  
**Model Bisnis:** Managed Service (OPEX per kamar/bulan: Hardware + Software CMS + Support 24/7) via Direct Sales & Kemitraan ISP/System Integrator

---

## 1. Latar Belakang & Masalah Pasar

Tamu hotel modern lebih memilih menonton layanan *streaming* pribadi mereka (Netflix, YouTube, Disney+, Spotify, Apple TV+) di layar lebar TV kamar dibandingkan saluran TV kabel konvensional. Namun, solusi yang ada saat ini memiliki masalah besar:

1. **Chromecast/Dongle Konsumen tanpa Manajemen:** Semua tamu di jaringan yang sama bisa melihat dan mengontrol TV kamar lain; akun tertinggal setelah *check-out*; rawan dicuri atau di-*reset*.
2. **Solusi Casting Enterprise yang Merepotkan:** Banyak sistem memaksa tamu memutus koneksi Wi-Fi hotel dan bergabung ke *hotspot* TV/STB, atau wajib memindai QR/memasukkan PIN berulang kali saat Wi-Fi terputus.
3. **Biaya Upgrade TV/IPTV Terlalu Tinggi:** Hotel bintang 3–4 enggan mengganti ratusan unit TV eksisting ke *Hospitality Smart TV* baru atau membayar lisensi sistem IPTV penuh yang mahal.
4. **Layar TV Tidak Menghasilkan Pendapatan:** Ketika tidak ditonton, layar TV hanya mati/gelap alih-alih mempromosikan layanan hotel dengan margin tinggi (F&B, spa, tur, *late check-out*).

---

## 2. Proporsi Nilai Utama (*Value Proposition*)

| Untuk Tamu Hotel | Untuk Manajemen Hotel | Untuk Mitra ISP / System Integrator |
|---|---|---|
| Tetap di Wi-Fi hotel utama tanpa pindah SSID | Tidak perlu ganti TV eksisting (bekerja di TV lama maupun Smart TV) | Paket *value-added service* yang menempel pada kontrak internet hotel |
| *Auto-pairing* tanpa QR/PIN jika terintegrasi portal/Wi-Fi hotel (plus fallback QR/PIN jika Wi-Fi tertutup) | Layar *idle* menjadi kanal promo **TV Ads** & **Guest Messaging** | Satu *dashboard* multi-properti untuk memantau seluruh klien hotel |
| Mendukung Netflix/YouTube/Disney+/Spotify via Google Cast resmi (HD/4K) + mirroring | Tidak menyimpan akun/kredensial tamu di TV; sesi terhapus otomatis saat *check-out* | Margin berulang (*recurring*) dengan arsitektur *gateway* yang ringan |
| Mendukung *screen mirroring* iOS/macOS non-DRM & Android/laptop | Harga jauh lebih hemat daripada sistem IPTV penuh | Mengurangi komplain Wi-Fi/TV tamu dengan isolasi jaringan terkelola |

---

## 3. Persona Pengguna

1. **Tamu Hotel (Leisure / Business / Family):** Ingin menonton konten pribadi di TV kamar dalam <15 detik tanpa instal aplikasi khusus, dengan privasi penuh.
2. **Front Office / Resepsionis:** Butuh cara cepat (1 klik atau otomatis dari PMS) untuk *check-in*, *check-out*, dan mengirim pesan penting ke kamar (mis. konfirmasi spa atau pengingat *check-out*).
3. **Marketing / General Manager Hotel:** Ingin mengubah *Welcome Screen*, menayangkan promo restoran/spa di TV kamar, melihat metrik tayang promo & penggunaan casting.
4. **IT Hotel / NOC Mitra ISP (Support 24/7):** Butuh visibilitas status *online/offline* Gateway dan STB di setiap kamar, *remote reboot*, serta log diagnostik saat tamu melapor kendala.

---

## 4. Lingkup Fungsional MVP

### 4.1 Pairing & Konektivitas Tamu (Mode Hybrid)
* **Mode 1 — Seamless Auto-Pairing (Prioritas):**
  * Tamu terhubung ke Wi-Fi utama hotel dan masuk melalui Captive Portal / RADIUS (mis. Nomor Kamar + Nama Belakang / Voucher).
  * Gateway secara otomatis mengikat IP perangkat tamu ke kamar tersebut.
  * Tamu langsung membuka aplikasi Cast (Netflix/YouTube/Spotify) dan **hanya TV kamarnya** yang muncul di daftar Cast.
* **Mode 2 — Fallback QR/PIN (Jaringan Terkunci):**
  * Jika jaringan Wi-Fi dikelola vendor lain tanpa akses portal/RADIUS (namun tetap satu subnet/routable ke Gateway), TV menampilkan kode QR dan PIN 6-digit dinamis.
  * Tamu memindai QR / membuka URL pairing lokal -> sesi kamar terbentuk di Gateway dengan isolasi yang sama.
* **Multi-Device & Multi-TV:**
  * Maksimal beberapa perangkat (konfigurabel, default 5 perangkat) per kamar dapat terhubung bersamaan.
  * Suite/vila dengan lebih dari satu TV menampilkan nama ramah per TV (mis. `"Room 501 - Living Room"`, `"Room 501 - Master Bedroom"`).

### 4.2 Pengalaman di Layar TV (*In-Room Receiver*)
* **Certified Casting + DRM:** Memutar stream dari Google Cast (Widevine L1) hingga resolusi 4K/Full HD.
* **Screen Mirroring:** Mendukung mirroring layar Android/Chrome (native Cast) dan iOS/macOS non-DRM (receiver AirPlay/WebRTC di perangkat Android TV).
* **HDMI-CEC Auto-Switch:** Saat sesi casting dimulai, perangkat membangunkan TV (*CEC Wake*) dan berpindah ke input HDMI receiver secara otomatis (bila TV mendukung CEC).
* **Idle State — Welcome Screen & TV Ads:**
  * Menampilkan logo hotel, sapaan personal (jika tersedia dari PMS/FO), informasi Wi-Fi, jam/cuaca, dan rotasi *playlist* TV Ads (gambar/video lokal tersimpan di cache).
  * Setiap iklan dapat menampilkan QR code CTA (mis. langsung ke menu pesan kamar/booking spa WhatsApp atau web hotel).
* **Guest Messaging & Pop-up Overlay:**
  * Pesan teks/banner bisa dikirim ke semua kamar (*broadcast*) atau kamar tertentu.
  * Muncul di *Welcome Screen* saat idle, atau sebagai *banner overlay* singkat saat tamu sedang menonton/casting (termasuk mode *Emergency Alert*).

### 4.3 Manajemen Siklus Tamu (*Lifecycle* & PMS)
* **Baseline Tanpa Ketergantungan PMS:**
  * Tersedia **Front Office Web Console** di CMS (check-in tamu, perpanjang, check-out manual 1 klik).
  * Aturan *auto-expire* berbasis waktu *check-out* default (mis. 12:00 / 13:00 waktu lokal hotel).
* **Adapter PMS Modular:**
  * Antarmuka standar untuk integrasi bertahap ke PMS populer di Indonesia/SEA (mis. VHP, Cloudbeds, Realta, Little Hotelier, Oracle Opera OHIP/FIAS).
  * Saat *event check-out* diterima: Gateway langsung memutus aturan akses perangkat tamu, menghapus tabel mDNS kamar, dan mengirim sinyal ke TV Receiver untuk menghentikan *playback* aktif serta kembali ke *Welcome Screen*.

### 4.4 Cloud Multi-Tenant CMS & Fleet Management
* **Hierarki Multi-Tenant:** `Platform Admin -> Partner/ISP -> Hotel Property -> Room -> Device`.
* **Manajemen Konten (CMS):** Editor layout Welcome Screen, manajemen *playlist* TV Ads (jadwal tayang, segmentasi kamar VIP/Corporate/Leisure), dan pengiriman *Guest Message*.
* **Device Lockdown & OTA:**
  * Aplikasi Android TV berjalan sebagai **Android Device Owner** (Kiosk mode: menonaktifkan tombol Home/Settings/Play Store bagi tamu).
  * *Heartbeat* berkala ke Gateway/Cloud, *remote reboot*, dan *silent APK OTA update* melalui Gateway lokal agar hemat *bandwidth* internet hotel.

---

## 5. Kebutuhan Non-Fungsional

1. **Privasi & Isolasi Mutlak:** Zero cross-room discovery (mDNS/SSDP/DIAL). Kredensial streaming tamu tidak pernah dikirim/disimpan di sistem hotel.
2. **Ketahanan Offline (*Edge Resilience*):** Jika koneksi internet hotel ke Cloud terputus, *pairing* lokal, *casting* lokal (untuk konten lokal/mirroring), *Welcome Screen*, dan *TV Ads* yang sudah di-*cache* harus tetap berjalan normal di Gateway.
3. **Kinerja & Skalabilitas Gateway:** Satu unit mini PC Gateway (2–4 vCPU, 2–4 GB RAM) harus mampu melayani properti hingga 250–300 kamar (~500–800 perangkat simultan) dengan latensi respon mDNS < 20 ms di LAN.
4. **Penanganan MAC Randomization:** Sesi tidak boleh rusak hanya karena perangkat tamu mengacak MAC saat berpindah AP; identifikasi utama pada lapisan Gateway memadukan IP aktif, lease tracking/portal token, dan peremajaan sesi.

---

## 6. Batasan Teknis & Risiko yang Diketahui

* **Apple AirPlay 2 FairPlay DRM:** Tanpa sertifikasi Apple MFi komersial yang mahal dan panjang, *screen mirroring* iOS/macOS terbatas pada konten non-DRM (foto, presentasi, browser, video lokal). Namun, aplikasi iOS populer (Netflix, YouTube, Spotify, Disney+) tetap berjalan penuh melalui **Google Cast SDK** di dalam masing-masing aplikasi iOS.
* **TV Non-CEC:** Pada TV hotel lama yang mematikan atau tidak mendukung HDMI-CEC, pergantian input TV tetap memerlukan tombol *Source/Input* di remote TV, atau konfigurasi *Hotel Mode* TV agar *power-on source* otomatis ke HDMI receiver.

---

## 7. Peta Jalan (*Roadmap*) Produk

| Fase | Fokus Utama | Target Pencapaian |
|---|---|---|
| **Fase 1 (Minggu 1–4): Core PoC & Gateway** | PoC Gateway Go (mDNS/SSDP proxy, dynamic firewall, portal & QR/PIN, edge cache, FO webhook) + uji isolasi antar-kamar | Validasi arsitektur jaringan di lab & 1 properti pilot |
| **Fase 2 (Minggu 5–10): Receiver & Cloud CMS MVP** | Aplikasi Android TV Kotlin (Device Owner, Backdrop, TV Ads, Messaging, CEC) + Cloud CMS Multi-Tenant + OTA via Gateway | Pilot berbayar di 1–2 hotel (30–100 kamar) |
| **Fase 3 (Bulan 3–5): PMS Connectors & Scale** | Konektor PMS (VHP/Cloudbeds/Opera), Partner Billing & Metering dashboard, analitik tayangan iklan & adopsi casting | Ekspansi komersial lewat mitra ISP/System Integrator |
| **Fase 4 (Bulan 6+): Ekosistem Tambahan** | Ekstensi ke *Digital Signage* lobi (modul Sign), ruang rapat (modul Present), atau *Light IPTV* (modul Live TV/VOD) | Upsell ARPU per properti |
