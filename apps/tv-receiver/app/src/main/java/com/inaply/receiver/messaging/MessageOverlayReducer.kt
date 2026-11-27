package com.inaply.receiver.messaging

import com.inaply.receiver.network.BackdropMessage

/**
 * Selects the highest-priority overlay message (emergency > urgent > normal)
 * and auto-dismisses normal overlays after normalDurationSec.
 */
class MessageOverlayReducer(private val normalDurationSec: Int = 10) {
    private var activeOverlay: BackdropMessage? = null
    private var elapsedSec: Int = 0

    fun onMessages(messages: List<BackdropMessage>) {
        val overlays = messages.filter { it.showAsOverlay }
        if (overlays.isEmpty()) {
            activeOverlay = null
            elapsedSec = 0
            return
        }
        val highest = overlays.maxByOrNull { priorityRank(it.priority) }
        if (highest?.id != activeOverlay?.id) {
            activeOverlay = highest
            elapsedSec = 0
        }
    }

    fun currentOverlay(): BackdropMessage? = activeOverlay

    fun isFullScreenEmergency(): Boolean {
        return activeOverlay?.priority.equals("emergency", ignoreCase = true)
    }

    fun advanceSeconds(seconds: Int) {
        val current = activeOverlay ?: return
        if (isFullScreenEmergency() || seconds <= 0) return
        elapsedSec += seconds
        if (current.priority.equals("normal", ignoreCase = true) && elapsedSec >= normalDurationSec) {
            activeOverlay = null
            elapsedSec = 0
        }
    }

    private fun priorityRank(priority: String): Int = when (priority.lowercase()) {
        "emergency" -> 3
        "urgent" -> 2
        else -> 1
    }
}
