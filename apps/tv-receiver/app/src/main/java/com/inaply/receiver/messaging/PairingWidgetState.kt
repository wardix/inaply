package com.inaply.receiver.messaging

/**
 * Controls visibility of the fallback QR & 6-digit PIN pairing widget.
 * Hides the full QR widget when at least one device is paired and restores it on checkout.
 */
data class PairingWidgetState(
    val showFullQrWidget: Boolean,
    val pairingPin: String,
    val pairingUrl: String
) {
    companion object {
        fun fromGateway(
            activePairedDevices: Int,
            pairingPin: String,
            pairingUrl: String
        ): PairingWidgetState {
            return PairingWidgetState(
                showFullQrWidget = activePairedDevices <= 0,
                pairingPin = pairingPin,
                pairingUrl = pairingUrl
            )
        }
    }
}
