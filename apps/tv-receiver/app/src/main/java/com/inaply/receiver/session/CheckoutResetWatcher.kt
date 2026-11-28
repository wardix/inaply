package com.inaply.receiver.session

/**
 * Watches Gateway `reset_epoch` and triggers playback stop & cache clear exactly once per increase.
 */
class CheckoutResetWatcher(
    initialEpoch: Long = 0L,
    private val onCheckoutReset: () -> Unit
) {
    private var lastEpoch: Long = initialEpoch

    fun onResetEpoch(epoch: Long) {
        if (epoch > lastEpoch) {
            lastEpoch = epoch
            onCheckoutReset()
        }
    }
}
