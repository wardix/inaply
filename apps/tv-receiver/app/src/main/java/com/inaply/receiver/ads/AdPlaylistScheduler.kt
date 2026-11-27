package com.inaply.receiver.ads

import com.inaply.receiver.network.BackdropAd

/**
 * Rotates active TV Ad campaigns by durationSec and preserves cached playback when offline.
 */
class AdPlaylistScheduler {
    private var playlist: List<BackdropAd> = emptyList()
    private var currentIndex: Int = 0
    private var elapsedSec: Int = 0
    var offlineFallbackActive: Boolean = false
        private set

    fun updatePlaylist(ads: List<BackdropAd>) {
        playlist = ads
        offlineFallbackActive = false
        if (playlist.isEmpty()) {
            currentIndex = 0
            elapsedSec = 0
        } else if (currentIndex >= playlist.size) {
            currentIndex = 0
            elapsedSec = 0
        }
    }

    fun onGatewayUnreachable() {
        offlineFallbackActive = true
    }

    fun currentAd(): BackdropAd? {
        if (playlist.isEmpty()) return null
        return playlist[currentIndex]
    }

    fun advanceSeconds(seconds: Int) {
        if (playlist.isEmpty() || seconds <= 0) return
        elapsedSec += seconds
        while (playlist.isNotEmpty()) {
            val duration = playlist[currentIndex].durationSec.coerceAtLeast(1)
            if (elapsedSec < duration) break
            elapsedSec -= duration
            currentIndex = (currentIndex + 1) % playlist.size
        }
    }
}
