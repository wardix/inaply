package com.inaply.receiver.mdm

import android.view.KeyEvent
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class KioskKeyPolicyTest {

    @Test
    fun KioskKeyPolicyTest_BlocksHomeAndSettingsWhenLocked() {
        val lockedPolicy = KioskKeyPolicy(kioskLocked = true)
        assertTrue(lockedPolicy.shouldBlockKey(KeyEvent.KEYCODE_HOME))
        assertTrue(lockedPolicy.shouldBlockKey(KeyEvent.KEYCODE_SETTINGS))
        assertTrue(lockedPolicy.shouldBlockKey(KeyEvent.KEYCODE_BACK))
        assertFalse(lockedPolicy.shouldBlockKey(KeyEvent.KEYCODE_DPAD_CENTER))

        val unlockedPolicy = KioskKeyPolicy(kioskLocked = false)
        assertFalse(unlockedPolicy.shouldBlockKey(KeyEvent.KEYCODE_HOME))
        assertFalse(unlockedPolicy.shouldBlockKey(KeyEvent.KEYCODE_SETTINGS))
    }
}
