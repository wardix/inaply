package com.inaply.receiver.messaging

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class PairingWidgetStateTest {

    @Test
    fun PairingWidgetStateTest_HidesQRWhenPairedAndRestoresOnCheckout() {
        val stateBefore = PairingWidgetState.fromGateway(
            activePairedDevices = 0,
            pairingPin = "482910",
            pairingUrl = "http://192.168.10.1/pair?pin=482910"
        )
        assertTrue(stateBefore.showFullQrWidget)

        val statePaired = PairingWidgetState.fromGateway(
            activePairedDevices = 2,
            pairingPin = "482910",
            pairingUrl = "http://192.168.10.1/pair?pin=482910"
        )
        assertFalse(statePaired.showFullQrWidget)

        // After checkout -> activePairedDevices drops to 0 -> QR widget restored
        val stateAfterCheckout = PairingWidgetState.fromGateway(
            activePairedDevices = 0,
            pairingPin = "991122",
            pairingUrl = "http://192.168.10.1/pair?pin=991122"
        )
        assertTrue(stateAfterCheckout.showFullQrWidget)
    }
}
