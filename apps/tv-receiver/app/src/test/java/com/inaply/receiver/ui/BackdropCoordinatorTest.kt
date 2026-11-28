package com.inaply.receiver.ui

import com.inaply.receiver.messaging.GuestMessageItem
import com.inaply.receiver.messaging.MessageDeduplicator
import com.inaply.receiver.messaging.MessageOverlayReducer
import com.inaply.receiver.messaging.PairingWidgetState
import com.inaply.receiver.session.CheckoutResetWatcher
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class BackdropCoordinatorTest {

    @Test
    fun `BackdropCoordinatorTest_DismissesExpiredNormalOverlayInDeduplicator`() {
        val deduplicator = MessageDeduplicator()
        val coordinator = BackdropCoordinator(
            deduplicator = deduplicator,
            overlayReducer = MessageOverlayReducer(),
            resetWatcher = CheckoutResetWatcher()
        )

        val msg = GuestMessageItem(
            id = "msg-normal-1",
            title = "Welcome",
            body = "Enjoy your stay",
            priority = "normal"
        )

        // First poll at t=1000 -> shows normal overlay
        coordinator.onMessagesPolled(listOf(msg), nowMs = 1000L)
        assertEquals("msg-normal-1", coordinator.currentOverlay?.id)

        // Advance time past auto-dismiss (e.g. 16s later) -> auto-dismissed & marked in deduplicator
        coordinator.tickOverlay(nowMs = 17_000L)
        assertNull(coordinator.currentOverlay)
        assertTrue(deduplicator.isDismissed("msg-normal-1"))

        // Subsequent poll with same message -> must NOT re-show
        coordinator.onMessagesPolled(listOf(msg), nowMs = 18_000L)
        assertNull(coordinator.currentOverlay)
    }

    @Test
    fun `BackdropCoordinatorTest_TriggersCheckoutResetAndRestoresQrWidget`() {
        var resetCount = 0
        val coordinator = BackdropCoordinator(
            deduplicator = MessageDeduplicator(),
            overlayReducer = MessageOverlayReducer(),
            resetWatcher = CheckoutResetWatcher(),
            onCheckoutReset = { resetCount++ }
        )

        // Guest is paired -> widget is compact
        coordinator.onPairingChanged(pairedCount = 1)
        assertEquals(PairingWidgetState.COMPACT_TOP_RIGHT, coordinator.pairingWidgetState)

        // Initial epoch 1
        coordinator.onResetEpochPolled(resetEpoch = 1)
        assertEquals(0, resetCount)

        // Epoch increments to 2 -> triggers reset once & restores full QR widget
        coordinator.onResetEpochPolled(resetEpoch = 2)
        assertEquals(1, resetCount)
        assertEquals(PairingWidgetState.FULL_QR_WIDGET, coordinator.pairingWidgetState)
    }
}
