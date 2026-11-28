package com.inaply.receiver.ui

import com.inaply.receiver.messaging.MessageDeduplicator
import com.inaply.receiver.messaging.MessageOverlayReducer
import com.inaply.receiver.messaging.PairingWidgetState
import com.inaply.receiver.network.BackdropMessage
import com.inaply.receiver.session.CheckoutResetWatcher

/**
 * Coordinates periodic GatewaySyncClient state updates with MessageOverlayReducer,
 * MessageDeduplicator, CheckoutResetWatcher, and PairingWidgetState.
 */
class BackdropCoordinator(
    private val deduplicator: MessageDeduplicator,
    private val overlayReducer: MessageOverlayReducer,
    private val onCheckoutReset: () -> Unit = {}
) {
    private val resetWatcher = CheckoutResetWatcher(initialEpoch = 0L) {
        pairingWidgetState = PairingWidgetState.fromGateway(
            activePairedDevices = 0,
            pairingPin = pairingWidgetState.pairingPin,
            pairingUrl = pairingWidgetState.pairingUrl
        )
        onCheckoutReset()
    }

    var currentOverlay: BackdropMessage? = null
        private set

    var pairingWidgetState: PairingWidgetState = PairingWidgetState.fromGateway(
        activePairedDevices = 0,
        pairingPin = "",
        pairingUrl = ""
    )
        private set

    fun onMessagesPolled(messages: List<BackdropMessage>) {
        val newOverlays = deduplicator.filterNewOverlays(messages)
        overlayReducer.onMessages(newOverlays)
        currentOverlay = overlayReducer.currentOverlay()
    }

    fun tickOverlay(seconds: Int) {
        val prev = overlayReducer.currentOverlay()
        overlayReducer.advanceSeconds(seconds)
        val next = overlayReducer.currentOverlay()
        if (prev != null && next == null && prev.priority == "normal") {
            deduplicator.markDismissed(prev.id)
        }
        currentOverlay = next
    }

    fun onPairingChanged(
        activePairedDevices: Int,
        pairingPin: String = "",
        pairingUrl: String = ""
    ) {
        pairingWidgetState = PairingWidgetState.fromGateway(
            activePairedDevices = activePairedDevices,
            pairingPin = pairingPin,
            pairingUrl = pairingUrl
        )
    }

    fun onResetEpochPolled(resetEpoch: Long) {
        resetWatcher.onResetEpoch(resetEpoch)
    }
}
