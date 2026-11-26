package com.inaply.receiver.mdm

import android.app.Activity
import android.app.admin.DeviceAdminReceiver
import android.app.admin.DevicePolicyManager
import android.content.ComponentName
import android.content.Context
import android.os.UserManager

/**
 * Device Owner receiver used during factory/staging provisioning
 * (`adb shell dpm set-device-owner com.inaply.receiver/.mdm.HotelDeviceAdminReceiver`)
 * to prevent guests from factory resetting or tampering with Wi-Fi/system settings.
 */
class HotelDeviceAdminReceiver : DeviceAdminReceiver()

class DeviceOwnerManager(private val context: Context) {
    private val dpm = context.getSystemService(Context.DEVICE_POLICY_SERVICE) as DevicePolicyManager
    private val adminComponent = ComponentName(context, HotelDeviceAdminReceiver::class.java)

    fun isDeviceOwner(): Boolean = dpm.isDeviceOwnerApp(context.packageName)

    fun applyHospitalityRestrictions(activity: Activity) {
        if (!isDeviceOwner()) return

        dpm.addUserRestriction(adminComponent, UserManager.DISALLOW_FACTORY_RESET)
        dpm.addUserRestriction(adminComponent, UserManager.DISALLOW_SAFE_BOOT)
        dpm.addUserRestriction(adminComponent, UserManager.DISALLOW_CONFIG_WIFI)
        dpm.addUserRestriction(adminComponent, UserManager.DISALLOW_INSTALL_UNKNOWN_SOURCES)

        dpm.setLockTaskPackages(adminComponent, arrayOf(context.packageName))
        activity.startLockTask()
    }
}
