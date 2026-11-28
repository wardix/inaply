package com.inaply.receiver.messaging

import com.inaply.receiver.network.BackdropMessage
import org.junit.Assert.assertEquals
import org.junit.Test

class MessageDeduplicatorTest {

    @Test
    fun MessageDeduplicatorTest_DoesNotReplayDismissedMessageID() {
        val dedup = MessageDeduplicator()
        val msg = BackdropMessage("m-101", "Spa", "20% off today", "normal", true)

        val firstPoll = dedup.filterNewOverlays(listOf(msg))
        assertEquals(1, firstPoll.size)

        dedup.markDismissed("m-101")
        val secondPoll = dedup.filterNewOverlays(listOf(msg))
        assertEquals(0, secondPoll.size)
    }
}
