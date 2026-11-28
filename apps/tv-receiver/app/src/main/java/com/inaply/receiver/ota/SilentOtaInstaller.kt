package com.inaply.receiver.ota

import java.io.File
import java.security.MessageDigest

/**
 * Verifies APK SHA-256 checksum before invoking Device Owner PackageInstaller.
 */
class SilentOtaInstaller(
    private val packageInstaller: (apkFile: File) -> Boolean
) {
    fun installIfVerified(apkFile: File, expectedSha256: String): Boolean {
        if (!apkFile.exists() || expectedSha256.isBlank()) return false
        val md = MessageDigest.getInstance("SHA-256")
        apkFile.inputStream().use { input ->
            val buffer = ByteArray(8192)
            while (true) {
                val read = input.read(buffer)
                if (read <= 0) break
                md.update(buffer, 0, read)
            }
        }
        val digest = md.digest()
        val actualSha256 = digest.joinToString("") { "%02x".format(it) }
        if (!actualSha256.equals(expectedSha256.trim(), ignoreCase = true)) {
            return false
        }
        return packageInstaller(apkFile)
    }
}
