package com.inaply.receiver.ads

import com.inaply.receiver.network.BackdropAd
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class AdPlaylistSchedulerTest {

    @Test
    fun AdPlaylistSchedulerTest_RotatesByDurationAndFallsBackOffline() {
        val scheduler = AdPlaylistScheduler()
        val ads = listOf(
            BackdropAd("ad-1", "Spa Promo", "http://192.168.10.1/assets/spa.mp4", "https://hotel.local/spa", 10),
            BackdropAd("ad-2", "Dinner", "http://192.168.10.1/assets/dinner.mp4", null, 15)
        )

        scheduler.updatePlaylist(ads)
        assertEquals("ad-1", scheduler.currentAd()?.id)

        // Advance 9s -> still ad-1
        scheduler.advanceSeconds(9)
        assertEquals("ad-1", scheduler.currentAd()?.id)

        // Advance 1s (total 10s) -> rotates to ad-2
        scheduler.advanceSeconds(1)
        assertEquals("ad-2", scheduler.currentAd()?.id)

        // Gateway unreachable -> keep playing cached playlist
        scheduler.onGatewayUnreachable()
        assertEquals("ad-2", scheduler.currentAd()?.id)

        // Empty playlist -> null
        scheduler.updatePlaylist(emptyList())
        assertNull(scheduler.currentAd())
    }
}
