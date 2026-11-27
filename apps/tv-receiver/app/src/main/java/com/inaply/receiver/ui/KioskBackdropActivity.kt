package com.inaply.receiver.ui

import android.os.Bundle
import android.view.KeyEvent
import androidx.activity.ComponentActivity
import com.inaply.receiver.mdm.KioskKeyPolicy

/**
 * Home Launcher Kiosk Activity for Inaply Android TV Receiver.
 * Displays hotel branding, personalized guest greeting, Wi-Fi info, and pairing status
 * while locking Home/Back/Settings keys when Kiosk mode is active.
 */
class KioskBackdropActivity : ComponentActivity() {

    private val viewModel = BackdropViewModel()
    private val keyPolicy = KioskKeyPolicy(kioskLocked = true)

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        // Compose for TV / Leanback content renders from viewModel.uiState
    }

    override fun dispatchKeyEvent(event: KeyEvent): Boolean {
        if (keyPolicy.shouldBlockKey(event.keyCode)) {
            return true
        }
        return super.dispatchKeyEvent(event)
    }
}
