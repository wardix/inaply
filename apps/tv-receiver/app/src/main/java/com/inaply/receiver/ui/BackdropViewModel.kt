package com.inaply.receiver.ui

import com.inaply.receiver.network.TVBackdropState

data class BackdropUiState(
    val hotelName: String = "",
    val roomNumber: String = "",
    val greetingTitle: String = "",
    val welcomeSubtitle: String = "",
    val wifiSsid: String = "",
    val pairingPin: String = "",
    val pairingUrl: String = "",
    val activePairedDevices: Int = 0,
    val occupied: Boolean = false
)

/**
 * State holder for the Kiosk Welcome Screen / Backdrop UI.
 */
class BackdropViewModel {
    var uiState: BackdropUiState = BackdropUiState()
        private set

    fun onGatewayState(
        state: TVBackdropState,
        occupied: Boolean,
        guestName: String
    ) {
        val greeting = when {
            occupied && guestName.isNotBlank() -> "Welcome, $guestName"
            occupied && state.welcomeTitle.isNotBlank() -> state.welcomeTitle
            state.hotelName.isNotBlank() -> "Welcome to ${state.hotelName}"
            else -> "Welcome"
        }

        uiState = BackdropUiState(
            hotelName = state.hotelName,
            roomNumber = state.roomNumber,
            greetingTitle = greeting,
            welcomeSubtitle = state.welcomeSubtitle,
            wifiSsid = state.wifiSsid,
            pairingPin = state.pairingPin,
            pairingUrl = state.pairingUrl,
            activePairedDevices = state.activePairedDevices,
            occupied = occupied
        )
    }
}
