package com.inaply.receiver.ui

import com.inaply.receiver.messaging.GuestMessageItem
import com.inaply.receiver.messaging.MessageDeduplicator
import com.inaply.receiver.messaging.MessageOverlayReducer
import com.inaply.receiver.messaging.PairingWidgetState
import com.inaply.receiver.session.CheckoutResetWatcher

/**
 * Coordinates periodic GatewaySyncClient state updates with MessageOverlayReducer,
 * MessageDeduplicator, CheckoutResetWatcher, and PairingWidgetState.
 */
class BackdropCoordinator(
    private val deduplicator: MessageDeduplicator,
    private val overlayReducer: MessageOverlayReducer,
    private val resetWatcher: CheckoutResetWatcher,
    private val onCheckoutReset: () -> Unit = {}
) {
    var currentOverlay: GuestMessageItem? = null
        private set

    var pairingWidgetState: PairingWidgetState = PairingWidgetState.FULL_QR_WIDGET
        private set

    fun onMessagesPolled(messages: List<GuestMessageItem>, nowMs: Long) {
        val visible = deduplicator.filterVisible(messages)
        currentOverlay = overlayReducer.reduce(visible, nowMs)
    }

    fun tickOverlay(nowMs: Long) {
        val prev = currentOverlay
        val next = overlayReducer.reduce(emptyList(), nowMs)
        if (prev != null && next == null && prev.priority == "normal") {
            deduplicator.markDismissed(prev.id)
        }
        currentOverlay = next
    }

    fun onPairingChanged(pairedCount: Int) {
        pairingWidgetState = if (pairedCount > 0) {
            PairingWidgetState.COMPACT_TOP_RIGHT
        } else {
            PairingWidgetState.FULL_QR_WIDGET
        }
    }

    fun onResetEpochPolled(resetEpoch: Int) {
        if (resetWatcher.shouldReset(resetEpoch)) {
            pairingWidgetState = PairingWidgetState.FULL_QR_WIDGET
            onCheckoutReset()
        }
    }
}
