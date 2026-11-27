package com.inaply.receiver.mdm

import android.view.KeyEvent

/**
 * Enforces hardware/remote key restrictions when Device Owner Kiosk mode is active.
 */
class KioskKeyPolicy(private val kioskLocked: Boolean) {

    fun shouldBlockKey(keyCode: Int): Boolean {
        if (!kioskLocked) return false
        return when (keyCode) {
            KeyEvent.KEYCODE_HOME,
            KeyEvent.KEYCODE_BACK,
            KeyEvent.KEYCODE_SETTINGS -> true
            else -> false
        }
    }
}
