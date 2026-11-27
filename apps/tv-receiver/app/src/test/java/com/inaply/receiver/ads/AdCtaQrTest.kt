package com.inaply.receiver.ads

import com.inaply.receiver.network.BackdropAd
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class AdCtaQrTest {

    @Test
    fun AdCtaQrTest_GeneratesQrPayloadOnlyWhenCtaPresent() {
        val withCta = BackdropAd("ad-1", "Spa", "http://gw/assets/spa.mp4", "https://hotel.local/spa", 10)
        val withoutCta = BackdropAd("ad-2", "Pool", "http://gw/assets/pool.mp4", null, 10)
        val blankCta = BackdropAd("ad-3", "Gym", "http://gw/assets/gym.mp4", "   ", 10)

        val qr1 = AdCtaQr.fromAd(withCta)
        assertEquals("https://hotel.local/spa", qr1?.payload)
        assertEquals("Spa", qr1?.label)

        assertNull(AdCtaQr.fromAd(withoutCta))
        assertNull(AdCtaQr.fromAd(blankCta))
    }
}
