package com.inaply.receiver.mirror

/**
 * Manages non-DRM AirPlay / WebRTC screen mirroring session lifecycle
 * and terminates active mirroring immediately when checkout `reset_epoch` increases.
 */
class MirrorSessionManager(
    initialEpoch: Long = 0L,
    private val onTerminateMirroring: () -> Unit
) {
    private var lastEpoch: Long = initialEpoch
    private var activeSessionId: String? = null
    private var activeProtocol: String? = null

    fun startSession(sessionId: String, protocol: String): Boolean {
        if (sessionId.isBlank() || protocol.isBlank()) return false
        activeSessionId = sessionId
        activeProtocol = protocol
        return true
    }

    fun isMirroringActive(): Boolean = activeSessionId != null

    fun onResetEpoch(epoch: Long) {
        if (epoch > lastEpoch) {
            lastEpoch = epoch
            if (activeSessionId != null) {
                activeSessionId = null
                activeProtocol = null
                onTerminateMirroring()
            }
        }
    }
}
