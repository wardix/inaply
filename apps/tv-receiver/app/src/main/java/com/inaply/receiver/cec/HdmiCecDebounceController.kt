package com.inaply.receiver.cec

/**
 * Debounces HDMI-CEC Active Source requests so rapid triggers do not flood the CEC bus.
 */
class HdmiCecDebounceController(
    private val debounceWindowMs: Long = 2000L,
    private val nowMs: () -> Long = { System.currentTimeMillis() },
    private val onRequestActiveSource: () -> Unit
) {
    private var lastRequestMs: Long = Long.MIN_VALUE

    fun requestActiveSource() {
        val now = nowMs()
        if (lastRequestMs != Long.MIN_VALUE && (now - lastRequestMs) < debounceWindowMs) {
            return
        }
        lastRequestMs = now
        runCatching { onRequestActiveSource() }
    }
}
