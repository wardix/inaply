package com.inaply.receiver.messaging

import com.inaply.receiver.network.BackdropMessage
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class MessageOverlayReducerTest {

    @Test
    fun MessageOverlayReducerTest_ShowsHighestPriorityAndAutoDismissesNormal() {
        val reducer = MessageOverlayReducer(normalDurationSec = 10)
        val normalMsg = BackdropMessage("m1", "Welcome", "Breakfast 7-10 AM", "normal", true)
        val emergencyMsg = BackdropMessage("m2", "Fire Alarm", "Please evacuate calmly", "emergency", true)

        reducer.onMessages(listOf(normalMsg, emergencyMsg))
        val active1 = reducer.currentOverlay()
        assertEquals("m2", active1?.id)
        assertTrue(reducer.isFullScreenEmergency())

        // Normal message auto-dismisses after normalDurationSec
        val reducer2 = MessageOverlayReducer(normalDurationSec = 10)
        reducer2.onMessages(listOf(normalMsg))
        assertEquals("m1", reducer2.currentOverlay()?.id)
        assertFalse(reducer2.isFullScreenEmergency())

        reducer2.advanceSeconds(10)
        assertNull(reducer2.currentOverlay())
    }
}
