package com.inaply.receiver.mdm

import org.junit.Assert.assertTrue
import org.junit.Test

class DeviceOwnerProvisioningTest {

    @Test
    fun DeviceOwnerProvisioningTest_GeneratesValidProvisioningPayload() {
        val json = DeviceOwnerProvisioning.generateJson(
            apkDownloadUrl = "https://cdn.inaply.local/receiver-v1.apk",
            apkChecksumSha256 = "abcdef1234567890"
        )

        assertTrue(json.contains("com.inaply.receiver/.mdm.HotelDeviceAdminReceiver"))
        assertTrue(json.contains("abcdef1234567890"))
        assertTrue(json.contains("https://cdn.inaply.local/receiver-v1.apk"))
    }
}
