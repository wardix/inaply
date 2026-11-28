package com.inaply.receiver.ota

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import org.junit.rules.TemporaryFolder
import java.io.File
import java.security.MessageDigest

class SilentOtaInstallerTest {

    @get:Rule
    val tempFolder = TemporaryFolder()

    @Test
    fun SilentOtaInstallerTest_VerifiesSha256BeforePackageInstall() {
        val apkBytes = "inaply-tv-receiver-v2.apk".toByteArray()
        val digest = MessageDigest.getInstance("SHA-256").digest(apkBytes)
        val validSha = digest.joinToString("") { "%02x".format(it) }

        val apkFile = File(tempFolder.root, "update.apk").apply { writeBytes(apkBytes) }
        var installCalled = 0

        val installer = SilentOtaInstaller { _ ->
            installCalled++
            true
        }

        // Invalid checksum -> rejected before PackageInstaller
        assertFalse(installer.installIfVerified(apkFile, "0000badsha256"))
        assertEquals(0, installCalled)

        // Valid checksum -> proceeds to PackageInstaller
        assertTrue(installer.installIfVerified(apkFile, validSha))
        assertEquals(1, installCalled)
    }
}
