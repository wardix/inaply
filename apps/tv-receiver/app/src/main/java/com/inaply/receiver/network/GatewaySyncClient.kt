package com.inaply.receiver.network

import org.json.JSONObject
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
    val occupied: Boolean = false,
    val guestName: String = "",
    val hotelName: String,
    val welcomeTitle: String,
    val welcomeSubtitle: String,
    val wifiSsid: String,
    val pairingPin: String,
    val pairingUrl: String,
    val activePairedDevices: Int,
    val resetEpoch: Long,
    val mode: String = "room_tv",
    val orientation: String = "landscape",
    val ads: List<BackdropAd> = emptyList(),
    val messages: List<BackdropMessage> = emptyList()
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

    fun fetchState(): TVBackdropState = parseStateJson(fetchRawStateJson())

    companion object {
        fun parseStateJson(json: String): TVBackdropState {
            val obj = JSONObject(json)
            val adsList = mutableListOf<BackdropAd>()
            val adsArr = obj.optJSONArray("ads")
            if (adsArr != null) {
                for (i in 0 until adsArr.length()) {
                    val item = adsArr.getJSONObject(i)
                    adsList.add(
                        BackdropAd(
                            id = item.optString("id", ""),
                            title = item.optString("title", ""),
                            mediaUrl = item.optString("media_url", ""),
                            ctaUrl = if (item.isNull("cta_url")) null else item.optString("cta_url", "").ifBlank { null },
                            durationSec = item.optInt("duration_sec", 15)
                        )
                    )
                }
            }

            val msgList = mutableListOf<BackdropMessage>()
            val msgArr = obj.optJSONArray("messages")
            if (msgArr != null) {
                for (i in 0 until msgArr.length()) {
                    val item = msgArr.getJSONObject(i)
                    msgList.add(
                        BackdropMessage(
                            id = item.optString("id", ""),
                            title = item.optString("title", ""),
                            body = item.optString("body", ""),
                            priority = item.optString("priority", "normal"),
                            showAsOverlay = item.optBoolean("show_as_overlay", false)
                        )
                    )
                }
            }

            return TVBackdropState(
                roomNumber = obj.optString("room_number", ""),
                deviceId = obj.optString("device_id", ""),
                occupied = obj.optBoolean("occupied", false),
                guestName = obj.optString("guest_name", ""),
                hotelName = obj.optString("hotel_name", ""),
                welcomeTitle = obj.optString("welcome_title", ""),
                welcomeSubtitle = obj.optString("welcome_subtitle", ""),
                wifiSsid = obj.optString("wifi_ssid", ""),
                pairingPin = obj.optString("pairing_pin", ""),
                pairingUrl = obj.optString("pairing_url", ""),
                activePairedDevices = obj.optInt("active_paired_devices", 0),
                resetEpoch = obj.optLong("reset_epoch", 0L),
                mode = obj.optString("mode", "room_tv"),
                orientation = obj.optString("orientation", "landscape"),
                ads = adsList,
                messages = msgList
            )
        }
    }
}
