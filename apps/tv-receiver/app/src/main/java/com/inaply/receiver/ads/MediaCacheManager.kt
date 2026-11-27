package com.inaply.receiver.ads

import java.io.File

/**
 * Manages local LAN media cache for TV Ads, avoiding redundant downloads when URL is unchanged.
 */
class MediaCacheManager(
    private val cacheDir: File,
    private val downloader: (url: String, dest: File) -> Unit
) {
    private val urlIndex = mutableMapOf<String, String>()

    init {
        if (!cacheDir.exists()) {
            cacheDir.mkdirs()
        }
    }

    fun ensureCached(filename: String, url: String): File {
        val target = File(cacheDir, filename)
        if (target.exists() && urlIndex[filename] == url) {
            return target
        }
        downloader(url, target)
        urlIndex[filename] = url
        return target
    }

    fun pruneExcept(keepFilenames: Set<String>) {
        val files = cacheDir.listFiles() ?: return
        for (file in files) {
            if (file.isFile && file.name !in keepFilenames) {
                file.delete()
                urlIndex.remove(file.name)
            }
        }
    }
}
