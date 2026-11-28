#!/usr/bin/env bash
set -euo pipefail

# Inaply TV Receiver - Device Owner Staging Script
# Prerequisite: Freshly factory-reset Android TV / Google TV with USB/ADB debugging enabled and no Google accounts added.

APK_PATH="${1:-./app/build/outputs/apk/release/app-release.apk}"

echo "[1/3] Installing Inaply TV Receiver APK..."
adb install -r "${APK_PATH}"

echo "[2/3] Setting Device Owner..."
adb shell dpm set-device-owner com.inaply.receiver/.mdm.HotelDeviceAdminReceiver

echo "[3/3] Staging complete. QA Checklist:"
echo "  - Verify Kiosk Home Launcher locks Home/Back/Settings"
echo "  - Verify Widevine L1 DRM playback & iOS/macOS non-DRM screen mirroring"
echo "  - Verify HDMI-CEC One-Touch Play & checkout reset_epoch"
