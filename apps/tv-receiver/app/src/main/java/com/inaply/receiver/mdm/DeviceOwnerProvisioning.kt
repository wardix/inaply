package com.inaply.receiver.mdm

/**
 * Generates Android Enterprise QR Device Owner provisioning JSON payload
 * for staging Google TV / Android TV STBs.
 */
object DeviceOwnerProvisioning {
    private const val ADMIN_COMPONENT = "com.inaply.receiver/.mdm.HotelDeviceAdminReceiver"

    fun generateJson(apkDownloadUrl: String, apkChecksumSha256: String): String {
        return """
            {
              "android.app.extra.PROVISIONING_DEVICE_ADMIN_COMPONENT_NAME": "$ADMIN_COMPONENT",
              "android.app.extra.PROVISIONING_DEVICE_ADMIN_PACKAGE_DOWNLOAD_LOCATION": "$apkDownloadUrl",
              "android.app.extra.PROVISIONING_DEVICE_ADMIN_PACKAGE_CHECKSUM": "$apkChecksumSha256",
              "android.app.extra.PROVISIONING_SKIP_ENCRYPTION": true
            }
        """.trimIndent()
    }
}
