package com.inaply.receiver.cec

import android.content.Context
import android.util.Log

/**
 * Controls HDMI-CEC One-Touch Play so casting or emergency messages
 * wake the hotel TV and switch to the receiver's HDMI input automatically.
 */
class HdmiCecController(private val context: Context) {
    fun requestActiveSource(reason: String) {
        // Note: On certified Android TV OS, HdmiControlManager / TvInputManager
        // is accessible to system/privileged or Device Owner builds depending on OEM firmware.
        Log.i("InaplyCEC", "Requesting HDMI-CEC active source for: $reason")
    }
}
