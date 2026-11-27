package com.inaply.receiver.ui

import com.inaply.receiver.network.TVBackdropState
import org.junit.Assert.assertEquals
import org.junit.Test

class BackdropViewModelTest {

    @Test
    fun BackdropViewModelTest_DisplaysGuestGreetingWhenOccupied() {
        val vm = BackdropViewModel()

        // Occupied room -> personal guest greeting
        vm.onGatewayState(
            TVBackdropState(
                roomNumber = "301",
                deviceId = "tv-301-main",
                occupied = true,
                guestName = "Budi",
                hotelName = "Grand Inaply Hotel",
                welcomeTitle = "Welcome, Budi",
                welcomeSubtitle = "Enjoy your stay",
                wifiSsid = "Inaply-Guest",
                pairingPin = "482910",
                pairingUrl = "http://192.168.10.1/pair?pin=482910",
                activePairedDevices = 1,
                resetEpoch = 1L
            ),
            localTime = "19:30",
            logoUrl = "https://cdn.inaply.local/logo.png",
            themeColor = "#0F172A"
        )
        assertEquals("Welcome, Budi", vm.uiState.greetingTitle)
        assertEquals("19:30", vm.uiState.localTime)
        assertEquals("#0F172A", vm.uiState.themeColor)

        // Vacant room -> fallback to property title
        vm.onGatewayState(
            TVBackdropState(
                roomNumber = "301",
                deviceId = "tv-301-main",
                occupied = false,
                guestName = "",
                hotelName = "Grand Inaply Hotel",
                welcomeTitle = "",
                welcomeSubtitle = "Connect to Wi-Fi to cast",
                wifiSsid = "Inaply-Guest",
                pairingPin = "112233",
                pairingUrl = "http://192.168.10.1/pair?pin=112233",
                activePairedDevices = 0,
                resetEpoch = 2L
            )
        )
        assertEquals("Welcome to Grand Inaply Hotel", vm.uiState.greetingTitle)
    }

    @Test
    fun BackdropViewModelTest_UpdatesPairedCountFromGatewayState() {
        val vm = BackdropViewModel()
        vm.onGatewayState(
            TVBackdropState(
                roomNumber = "302",
                deviceId = "tv-302-main",
                occupied = true,
                guestName = "Siti",
                hotelName = "Grand Inaply Hotel",
                welcomeTitle = "Welcome, Siti",
                welcomeSubtitle = "Enjoy your stay",
                wifiSsid = "Inaply-Guest",
                pairingPin = "998877",
                pairingUrl = "http://192.168.10.1/pair?pin=998877",
                activePairedDevices = 3,
                resetEpoch = 1L
            )
        )

        assertEquals(3, vm.uiState.activePairedDevices)
        assertEquals("Inaply-Guest", vm.uiState.wifiSsid)
    }
}
