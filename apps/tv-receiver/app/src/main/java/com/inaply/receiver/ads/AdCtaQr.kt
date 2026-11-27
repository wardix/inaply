package com.inaply.receiver.ads

import com.inaply.receiver.network.BackdropAd

data class AdCtaQrState(
    val label: String,
    val payload: String
)

/**
 * Generates QR CTA overlay state only when an ad has a valid non-blank ctaUrl.
 */
object AdCtaQr {
    fun fromAd(ad: BackdropAd): AdCtaQrState? {
        val url = ad.ctaUrl?.trim().orEmpty()
        if (url.isEmpty()) return null
        return AdCtaQrState(
            label = ad.title,
            payload = url
        )
    }
}
