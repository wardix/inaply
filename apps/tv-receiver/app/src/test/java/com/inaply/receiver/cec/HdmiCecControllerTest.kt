package com.inaply.receiver.cec

import org.junit.Assert.assertEquals
import org.junit.Test

class HdmiCecControllerTest {

    @Test
    fun HdmiCecControllerTest_DebouncesActiveSourceRequests() {
        var nowMs = 1000L
        var sentCount = 0
        val controller = HdmiCecDebounceController(
            debounceWindowMs = 2000L,
            nowMs = { nowMs },
            onRequestActiveSource = { sentCount++ }
        )

        // First request -> sent
        controller.requestActiveSource()
        assertEquals(1, sentCount)

        // Within debounce window -> ignored
        nowMs = 1500L
        controller.requestActiveSource()
        assertEquals(1, sentCount)

        // Past debounce window -> sent
        nowMs = 3500L
        controller.requestActiveSource()
        assertEquals(2, sentCount)
    }
}
