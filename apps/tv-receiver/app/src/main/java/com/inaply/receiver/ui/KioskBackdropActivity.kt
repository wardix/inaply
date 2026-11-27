package com.inaply.receiver.ui

import android.os.Bundle
import android.view.KeyEvent
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import com.inaply.receiver.mdm.DeviceOwnerManager
import com.inaply.receiver.mdm.KioskKeyPolicy
import com.inaply.receiver.network.GatewaySyncClient

/**
 * Home Launcher Kiosk Activity for Inaply Android TV Receiver.
 * Displays hotel branding, personalized guest greeting, Wi-Fi info, local time,
 * and paired device count while locking Home/Back/Settings keys when Kiosk mode is active.
 */
class KioskBackdropActivity : ComponentActivity() {

    private val viewModel = BackdropViewModel()
    private val keyPolicy = KioskKeyPolicy(kioskLocked = true)
    private lateinit var deviceOwnerManager: DeviceOwnerManager
    private lateinit var gatewaySyncClient: GatewaySyncClient
    private var uiState by mutableStateOf(BackdropUiState())

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        deviceOwnerManager = DeviceOwnerManager(this)
        gatewaySyncClient = GatewaySyncClient("http://192.168.10.1:8080", "tv-301-main")
        deviceOwnerManager.applyHospitalityRestrictions(this)
        uiState = viewModel.uiState

        setContent {
            MaterialTheme {
                KioskWelcomeScreen(state = uiState)
            }
        }
    }

    override fun dispatchKeyEvent(event: KeyEvent): Boolean {
        if (keyPolicy.shouldBlockKey(event.keyCode)) {
            return true
        }
        return super.dispatchKeyEvent(event)
    }
}

@Composable
fun KioskWelcomeScreen(state: BackdropUiState) {
    val bgColor = runCatching {
        Color(android.graphics.Color.parseColor(state.themeColor))
    }.getOrElse { Color(0xFF1E293B) }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(bgColor)
            .padding(48.dp),
        verticalArrangement = Arrangement.SpaceBetween
    ) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            Text(
                text = state.hotelName.ifBlank { "Inaply Hospitality" },
                style = MaterialTheme.typography.headlineMedium,
                color = Color.White
            )
            Text(
                text = state.localTime,
                style = MaterialTheme.typography.headlineSmall,
                color = Color.White.copy(alpha = 0.85f)
            )
        }

        Column {
            Text(
                text = state.greetingTitle.ifBlank { "Welcome" },
                style = MaterialTheme.typography.displayMedium,
                color = Color.White
            )
            Spacer(modifier = Modifier.height(12.dp))
            Text(
                text = state.welcomeSubtitle.ifBlank { "Connect to Wi-Fi to cast from your device" },
                style = MaterialTheme.typography.titleLarge,
                color = Color.White.copy(alpha = 0.8f)
            )
        }

        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween
        ) {
            Text(
                text = "Wi-Fi: ${state.wifiSsid.ifBlank { "Hotel-Guest" }}  •  Room ${state.roomNumber}",
                style = MaterialTheme.typography.titleMedium,
                color = Color.White.copy(alpha = 0.9f)
            )
            Text(
                text = "Paired Devices: ${state.activePairedDevices}",
                style = MaterialTheme.typography.titleMedium,
                color = Color.White.copy(alpha = 0.9f)
            )
        }
    }
}
