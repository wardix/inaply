package com.inaply.receiver.ads

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import org.junit.rules.TemporaryFolder
import java.io.File

class MediaCacheManagerTest {

    @get:Rule
    val tempFolder = TemporaryFolder()

    @Test
    fun MediaCacheManagerTest_ReusesCachedFileWhenURLUnchanged() {
        var downloadCount = 0
        val cache = MediaCacheManager(tempFolder.root) { url, dest ->
            downloadCount++
            dest.writeText("media-bytes-for-$url")
        }

        val first = cache.ensureCached("spa.mp4", "http://192.168.10.1/assets/spa.mp4")
        assertTrue(first.exists())
        assertEquals(1, downloadCount)

        // Same URL -> reuses cached file without re-downloading
        val second = cache.ensureCached("spa.mp4", "http://192.168.10.1/assets/spa.mp4")
        assertEquals(first.absolutePath, second.absolutePath)
        assertEquals(1, downloadCount)

        // Prune obsolete files when ad list changes
        val oldFile = File(tempFolder.root, "old-promo.mp4").apply { writeText("old") }
        cache.pruneExcept(setOf("spa.mp4"))
        assertFalse(oldFile.exists())
        assertTrue(first.exists())
    }

    @Test
    fun `MediaCacheManagerTest_PersistsUrlIndexAcrossInstances`() {
        var downloadCount = 0
        val firstInstance = MediaCacheManager(tempFolder.root) { _, dest ->
            downloadCount++
            dest.writeText("video-bytes")
        }
        firstInstance.ensureCached("spa.mp4", "http://192.168.10.1/assets/spa.mp4")
        assertEquals(1, downloadCount)

        // New instance on the same cacheDir (simulating STB reboot) must NOT re-download
        val secondInstance = MediaCacheManager(tempFolder.root) { _, dest ->
            downloadCount++
            dest.writeText("video-bytes-2")
        }
        val cached = secondInstance.ensureCached("spa.mp4", "http://192.168.10.1/assets/spa.mp4")
        assertTrue(cached.exists())
        assertEquals(1, downloadCount)
    }
}
