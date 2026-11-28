package com.inaply.receiver.session

import org.junit.Assert.assertEquals
import org.junit.Test

class CheckoutResetWatcherTest {

    @Test
    fun CheckoutResetWatcherTest_TriggersStopAndCacheClearOnResetEpochIncrease() {
        var resetCount = 0
        val watcher = CheckoutResetWatcher(initialEpoch = 1L) {
            resetCount++
        }

        // Same epoch -> no trigger
        watcher.onResetEpoch(1L)
        assertEquals(0, resetCount)

        // Epoch increases -> triggers exactly once
        watcher.onResetEpoch(2L)
        assertEquals(1, resetCount)

        // Polling again with same new epoch -> no duplicate trigger
        watcher.onResetEpoch(2L)
        assertEquals(1, resetCount)
    }
}
