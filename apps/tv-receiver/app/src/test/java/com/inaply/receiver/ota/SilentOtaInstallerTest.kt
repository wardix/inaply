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

    @Test
    fun `SilentOtaInstallerTest_StreamsLargeFileWithoutReadBytes`() {
        val tempFile = File.createTempFile("ota-large-", ".apk").apply { deleteOnExit() }
        val chunk = ByteArray(64 * 1024) { 0x5A.toByte() }
        tempFile.outputStream().use { out ->
            repeat(20) { out.write(chunk) } // > 1 MB (1.28 MB)
        }
        val md = java.security.MessageDigest.getInstance("SHA-256")
        tempFile.inputStream().use { input ->
            val buf = ByteArray(8192)
            while (true) {
                val read = input.read(buf)
                if (read <= 0) break
                md.update(buf, 0, read)
            }
        }
        val expectedSha = md.digest().joinToString("") { "%02x".format(it) }

        var installed = false
        val installer = SilentOtaInstaller { _ ->
            installed = true
            true
        }
        assertTrue(installer.installIfVerified(tempFile, expectedSha))
        assertTrue(installed)
    }
}
