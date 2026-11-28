package com.inaply.receiver.ui

import com.inaply.receiver.messaging.MessageDeduplicator
import com.inaply.receiver.messaging.MessageOverlayReducer
import com.inaply.receiver.network.BackdropMessage
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class BackdropCoordinatorTest {

    @Test
    fun `BackdropCoordinatorTest_DismissesExpiredNormalOverlayInDeduplicator`() {
        val deduplicator = MessageDeduplicator()
        val coordinator = BackdropCoordinator(
            deduplicator = deduplicator,
            overlayReducer = MessageOverlayReducer()
        )

        val msg = BackdropMessage(
            id = "msg-normal-1",
            title = "Welcome",
            body = "Enjoy your stay",
            priority = "normal",
            showAsOverlay = true
        )

        // First poll -> shows normal overlay
        coordinator.onMessagesPolled(listOf(msg))
        assertEquals("msg-normal-1", coordinator.currentOverlay?.id)

        // Advance time past auto-dismiss (16s) -> auto-dismissed & marked in deduplicator
        coordinator.tickOverlay(seconds = 16)
        assertNull(coordinator.currentOverlay)

        // Subsequent poll with same message -> deduplicator filters it out
        coordinator.onMessagesPolled(listOf(msg))
        assertNull(coordinator.currentOverlay)
    }

    @Test
    fun `BackdropCoordinatorTest_TriggersCheckoutResetAndRestoresQrWidget`() {
        var resetCount = 0
        val coordinator = BackdropCoordinator(
            deduplicator = MessageDeduplicator(),
            overlayReducer = MessageOverlayReducer(),
            onCheckoutReset = { resetCount++ }
        )

        // Guest is paired -> full QR widget is hidden
        coordinator.onPairingChanged(
            activePairedDevices = 1,
            pairingPin = "123456",
            pairingUrl = "http://192.168.10.1/pair"
        )
        assertFalse(coordinator.pairingWidgetState.showFullQrWidget)

        // Initial epoch 1 -> no reset
        coordinator.onResetEpochPolled(resetEpoch = 1L)
        assertEquals(0, resetCount)

        // Epoch increments to 2 -> triggers reset once & restores full QR widget
        coordinator.onResetEpochPolled(resetEpoch = 2L)
        assertEquals(1, resetCount)
        assertTrue(coordinator.pairingWidgetState.showFullQrWidget)
    }
}
