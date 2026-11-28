package com.inaply.receiver.mirror

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class MirrorSessionManagerTest {

    @Test
    fun MirrorSessionManagerTest_TerminatesMirroringOnCheckoutResetEpoch() {
        var stopCalled = false
        val manager = MirrorSessionManager(initialEpoch = 1L) {
            stopCalled = true
        }

        assertTrue(manager.startSession(sessionId = "airplay-101", protocol = "airplay"))
        assertTrue(manager.isMirroringActive())

        // Same epoch -> keeps mirroring
        manager.onResetEpoch(1L)
        assertTrue(manager.isMirroringActive())
        assertFalse(stopCalled)

        // Checkout increments resetEpoch -> terminates mirroring immediately
        manager.onResetEpoch(2L)
        assertFalse(manager.isMirroringActive())
        assertTrue(stopCalled)
    }
}
