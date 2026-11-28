package com.inaply.receiver.ads

import java.io.File
import java.util.Properties

/**
 * Manages local LAN media cache for TV Ads, persisting URL index across reboots
 * and downloading atomically via temp files.
 */
class MediaCacheManager(
    private val cacheDir: File,
    private val downloader: (url: String, dest: File) -> Unit
) {
    private val indexFile = File(cacheDir, INDEX_FILENAME)
    private val urlIndex = mutableMapOf<String, String>()

    init {
        if (!cacheDir.exists()) {
            cacheDir.mkdirs()
        }
        loadIndex()
    }

    fun ensureCached(filename: String, url: String): File {
        require(
            filename.isNotBlank() &&
                !filename.contains('/') &&
                !filename.contains('\\') &&
                !filename.contains("..") &&
                filename != INDEX_FILENAME
        ) {
            "Invalid cache filename: $filename"
        }
        val target = File(cacheDir, filename)
        require(target.canonicalFile.parentFile == cacheDir.canonicalFile) {
            "Path traversal detected for filename: $filename"
        }
        if (target.exists() && urlIndex[filename] == url) {
            return target
        }

        val tempFile = File.createTempFile("$filename.", ".tmp", cacheDir)
        try {
            downloader(url, tempFile)
            if (target.exists()) {
                target.delete()
            }
            if (!tempFile.renameTo(target)) {
                tempFile.copyTo(target, overwrite = true)
                tempFile.delete()
            }
        } catch (t: Throwable) {
            tempFile.delete()
            throw t
        }

        urlIndex[filename] = url
        saveIndex()
        return target
    }

    fun pruneExcept(keepFilenames: Set<String>) {
        val files = cacheDir.listFiles() ?: return
        var changed = false
        for (file in files) {
            if (file.isFile && file.name != INDEX_FILENAME && file.name !in keepFilenames) {
                file.delete()
                if (urlIndex.remove(file.name) != null) {
                    changed = true
                }
            }
        }
        if (changed) {
            saveIndex()
        }
    }

    private fun loadIndex() {
        if (!indexFile.exists()) return
        val props = Properties()
        indexFile.inputStream().use { props.load(it) }
        for (name in props.stringPropertyNames()) {
            urlIndex[name] = props.getProperty(name)
        }
    }

    private fun saveIndex() {
        val props = Properties()
        for ((k, v) in urlIndex) {
            props.setProperty(k, v)
        }
        val tempIndex = File.createTempFile(INDEX_FILENAME, ".tmp", cacheDir)
        tempIndex.outputStream().use { props.store(it, "Inaply MediaCacheManager URL index") }
        if (indexFile.exists()) {
            indexFile.delete()
        }
        if (!tempIndex.renameTo(indexFile)) {
            tempIndex.copyTo(indexFile, overwrite = true)
            tempIndex.delete()
        }
    }

    companion object {
        private const val INDEX_FILENAME = "cache-index.properties"
    }
}
