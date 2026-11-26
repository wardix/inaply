# Android TV Receiver (`apps/tv-receiver`)

Aplikasi Kiosk/Launcher native **Kotlin** untuk perangkat **Android TV / Google TV bersertifikat** (Widevine L1 + Google Cast resmi).

## Tanggung Jawab Utama
1. **Device Owner Kiosk Lockdown (`DeviceOwnerManager.kt`):**
   - Menjadikan aplikasi sebagai Home Launcher dan mengunci tombol Settings/Factory Reset/Play Store bagi tamu.
   - Mendukung *silent APK update* melalui Gateway lokal.
2. **Welcome Screen, TV Ads & Guest Messaging (`BackdropState.kt`):**
   - Menampilkan sapaan tamu personal, rotasi gambar/video *TV Ads* (diambil dari Gateway LAN cache), serta pesan tamu / *pop-up overlay*.
   - Menampilkan QR/PIN *fallback* ketika diperlukan.
3. **HDMI-CEC Auto-Switch (`HdmiCecController.kt`):**
   - Membangunkan TV dan memindahkan input HDMI saat ada sesi *casting* atau peringatan darurat.
4. **Checkout & Reset Handler (`GatewaySyncClient.kt`):**
   - Memantau `reset_epoch` dari Gateway. Saat Front Office/PMS memicu *check-out*, aplikasi menghentikan media aktif, membersihkan data sementara, dan kembali ke *Welcome Screen*.
5. **Protokol Casting & Mirroring:**
   - **Google Cast:** Menggunakan kapabilitas Cast Receiver resmi perangkat Google TV/Android TV (Widevine L1).
   - **iOS/macOS Mirroring Non-DRM:** Diintegrasikan melalui modul receiver AirPlay non-DRM / WebRTC pada perangkat Android TV.
