package com.inaply.receiver.network

import java.net.HttpURLConnection
import java.net.URL

data class BackdropAd(
    val id: String,
    val title: String,
    val mediaUrl: String,
    val ctaUrl: String?,
    val durationSec: Int
)

data class BackdropMessage(
    val id: String,
    val title: String,
    val body: String,
    val priority: String,
    val showAsOverlay: Boolean
)

data class TVBackdropState(
    val roomNumber: String,
    val deviceId: String,
    val hotelName: String,
    val welcomeTitle: String,
    val welcomeSubtitle: String,
    val wifiSsid: String,
    val pairingPin: String,
    val pairingUrl: String,
    val activePairedDevices: Int,
    val resetEpoch: Long
)

/**
 * Syncs receiver state with the On-Premise Gateway over hotel LAN.
 * Watches `resetEpoch` to immediately terminate sessions when checkout occurs.
 */
class GatewaySyncClient(
    private val gatewayBaseUrl: String,
    private val deviceId: String
) {
    fun fetchRawStateJson(): String {
        val url = URL("$gatewayBaseUrl/api/v1/tv/state?device_id=$deviceId")
        val conn = (url.openConnection() as HttpURLConnection).apply {
            requestMethod = "GET"
            connectTimeout = 3000
            readTimeout = 3000
        }
        return conn.inputStream.bufferedReader().use { it.readText() }
    }
}
