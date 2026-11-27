package com.inaply.receiver.messaging

import com.inaply.receiver.network.BackdropMessage

/**
 * Tracks dismissed message IDs so polling does not replay already-dismissed overlays.
 */
class MessageDeduplicator {
    private val dismissedIds = mutableSetOf<String>()

    fun markDismissed(messageId: String) {
        dismissedIds.add(messageId)
    }

    fun filterNewOverlays(messages: List<BackdropMessage>): List<BackdropMessage> {
        return messages.filter { it.showAsOverlay && it.id !in dismissedIds }
    }
}
