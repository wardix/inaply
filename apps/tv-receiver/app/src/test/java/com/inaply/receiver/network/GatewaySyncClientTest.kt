package com.inaply.receiver.network

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class GatewaySyncClientTest {

    @Test
    fun `parseStateJson parses TVBackdropState with ads and messages`() {
        val json = """
            {
              "room_number": "301",
              "device_id": "tv-301",
              "occupied": true,
              "guest_name": "Alice",
              "hotel_name": "Grand Inaply",
              "welcome_title": "Welcome Alice",
              "welcome_subtitle": "Enjoy your stay",
              "wifi_ssid": "Inaply-Guest",
              "pairing_pin": "654321",
              "pairing_url": "http://192.168.10.1/pair?room=301",
              "active_paired_devices": 1,
              "reset_epoch": 3,
              "ads": [
                {
                  "id": "ad-1",
                  "title": "Spa Promo",
                  "media_url": "http://192.168.10.1/assets/spa.mp4",
                  "cta_url": "https://hotel.example.com/spa",
                  "duration_sec": 15
                }
              ],
              "messages": [
                {
                  "id": "msg-1",
                  "title": "Fire Drill",
                  "body": "Test at 10:00",
                  "priority": "emergency",
                  "show_as_overlay": true
                }
              ]
            }
        """.trimIndent()

        val state = GatewaySyncClient.parseStateJson(json)
        assertEquals("301", state.roomNumber)
        assertEquals("tv-301", state.deviceId)
        assertTrue(state.occupied)
        assertEquals("Alice", state.guestName)
        assertEquals("654321", state.pairingPin)
        assertEquals(1, state.activePairedDevices)
        assertEquals(3L, state.resetEpoch)

        assertEquals(1, state.ads.size)
        assertEquals("ad-1", state.ads[0].id)
        assertEquals("https://hotel.example.com/spa", state.ads[0].ctaUrl)
        assertEquals(15, state.ads[0].durationSec)

        assertEquals(1, state.messages.size)
        assertEquals("msg-1", state.messages[0].id)
        assertEquals("emergency", state.messages[0].priority)
        assertTrue(state.messages[0].showAsOverlay)
    }

    @Test
    fun `parseStateJson handles missing optional ads and messages arrays`() {
        val json = """
            {
              "room_number": "302",
              "device_id": "tv-302",
              "hotel_name": "Grand Inaply",
              "welcome_title": "Welcome",
              "welcome_subtitle": "",
              "wifi_ssid": "Inaply-Guest",
              "pairing_pin": "111222",
              "pairing_url": "http://192.168.10.1/pair",
              "active_paired_devices": 0,
              "reset_epoch": 1
            }
        """.trimIndent()

        val state = GatewaySyncClient.parseStateJson(json)
        assertEquals("302", state.roomNumber)
        assertFalse(state.occupied)
        assertEquals("room_tv", state.mode)
        assertEquals("landscape", state.orientation)
        assertTrue(state.ads.isEmpty())
        assertTrue(state.messages.isEmpty())
    }

    @Test
    fun `parseStateJson parses signage mode and portrait orientation`() {
        val json = """
            {
              "room_number": "",
              "device_id": "tv-lift-1",
              "mode": "signage",
              "orientation": "portrait",
              "hotel_name": "Grand Inaply",
              "welcome_title": "Spa Promo",
              "welcome_subtitle": "",
              "wifi_ssid": "",
              "pairing_pin": "",
              "pairing_url": "",
              "active_paired_devices": 0,
              "reset_epoch": 0
            }
        """.trimIndent()

        val state = GatewaySyncClient.parseStateJson(json)
        assertEquals("signage", state.mode)
        assertEquals("portrait", state.orientation)
    }
}
