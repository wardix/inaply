package com.inaply.receiver.ui

import android.content.pm.ActivityInfo
import android.os.Bundle
import android.view.KeyEvent
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import com.inaply.receiver.ads.AdCtaQr
import com.inaply.receiver.ads.AdPlaylistScheduler
import com.inaply.receiver.ads.MediaCacheManager
import com.inaply.receiver.cec.HdmiCecController
import com.inaply.receiver.cec.HdmiCecDebounceController
import com.inaply.receiver.mdm.DeviceOwnerManager
import com.inaply.receiver.mdm.KioskKeyPolicy
import com.inaply.receiver.messaging.MessageDeduplicator
import com.inaply.receiver.messaging.MessageOverlayReducer
import com.inaply.receiver.messaging.PairingWidgetState
import com.inaply.receiver.mirror.MirrorSessionManager
import com.inaply.receiver.network.BackdropAd
import com.inaply.receiver.network.BackdropMessage
import com.inaply.receiver.network.GatewaySyncClient
import java.io.File
import java.net.URL
import java.util.concurrent.Executors
import java.util.concurrent.ScheduledExecutorService
import java.util.concurrent.TimeUnit

/**
 * Home Launcher Kiosk Activity for Inaply Android TV Receiver.
 * Coordinates GatewaySyncClient polling, BackdropViewModel, AdPlaylistScheduler,
 * MediaCacheManager, BackdropCoordinator, MirrorSessionManager, and HDMI-CEC.
 */
class KioskBackdropActivity : ComponentActivity() {

    private val viewModel = BackdropViewModel()
    private val keyPolicy = KioskKeyPolicy(kioskLocked = true)
    private val overlayReducer = MessageOverlayReducer()
    private val deduplicator = MessageDeduplicator()
    private val adScheduler = AdPlaylistScheduler()

    private lateinit var mirrorSessionManager: MirrorSessionManager
    private lateinit var cecController: HdmiCecController
    private lateinit var cecDebounce: HdmiCecDebounceController

    private lateinit var deviceOwnerManager: DeviceOwnerManager
    private lateinit var gatewaySyncClient: GatewaySyncClient
    private lateinit var mediaCacheManager: MediaCacheManager
    private lateinit var coordinator: BackdropCoordinator
    private var executor: ScheduledExecutorService? = null

    private var uiState by mutableStateOf(BackdropUiState())
    private var currentAd by mutableStateOf<BackdropAd?>(null)
    private var currentOverlay by mutableStateOf<BackdropMessage?>(null)
    private var isEmergencyOverlay by mutableStateOf(false)
    private var pairingWidgetState by mutableStateOf(
        PairingWidgetState.fromGateway(0, "", "")
    )

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        deviceOwnerManager = DeviceOwnerManager(this)
        gatewaySyncClient = GatewaySyncClient("http://192.168.10.1:8080", "tv-301-main")
        mediaCacheManager = MediaCacheManager(File(cacheDir, "ads")) { url, dest ->
            URL(url).openStream().use { input ->
                dest.outputStream().use { output -> input.copyTo(output) }
            }
        }
        mirrorSessionManager = MirrorSessionManager(initialEpoch = 0L) {
            // Active mirroring session terminated on checkout reset
        }
        cecController = HdmiCecController(this)
        cecDebounce = HdmiCecDebounceController(debounceWindowMs = 5_000L) {
            cecController.requestActiveSource("emergency_overlay")
        }
        coordinator = BackdropCoordinator(
            deduplicator = deduplicator,
            overlayReducer = overlayReducer
        )

        deviceOwnerManager.applyHospitalityRestrictions(this)
        uiState = viewModel.uiState
        startSyncLoop()

        setContent {
            MaterialTheme {
                KioskWelcomeScreen(
                    state = uiState,
                    currentAd = currentAd,
                    currentOverlay = currentOverlay,
                    isEmergency = isEmergencyOverlay,
                    pairingWidget = pairingWidgetState
                )
            }
        }
    }

    private fun startSyncLoop() {
        val pool = Executors.newSingleThreadScheduledExecutor()
        executor = pool

        pool.scheduleAtFixedRate({
            try {
                val state = gatewaySyncClient.fetchState()
                viewModel.onGatewayState(state)
                adScheduler.updatePlaylist(state.ads)
                val keepFiles = state.ads.map { "${it.id}.mp4" }.toSet()
                state.ads.forEach { ad ->
                    runCatching { mediaCacheManager.ensureCached("${ad.id}.mp4", ad.mediaUrl) }
                }
                mediaCacheManager.pruneExcept(keepFiles)

                coordinator.onMessagesPolled(state.messages)
                coordinator.onPairingChanged(
                    activePairedDevices = state.activePairedDevices,
                    pairingPin = state.pairingPin,
                    pairingUrl = state.pairingUrl
                )
                coordinator.onResetEpochPolled(state.resetEpoch)
                mirrorSessionManager.onResetEpoch(state.resetEpoch)

                if (overlayReducer.isFullScreenEmergency()) {
                    cecDebounce.requestActiveSource()
                }

                runOnUiThread {
                    uiState = viewModel.uiState
                    val targetOrientation = if (uiState.orientation == "portrait") {
                        ActivityInfo.SCREEN_ORIENTATION_PORTRAIT
                    } else {
                        ActivityInfo.SCREEN_ORIENTATION_LANDSCAPE
                    }
                    if (requestedOrientation != targetOrientation) {
                        requestedOrientation = targetOrientation
                    }
                    currentAd = adScheduler.currentAd()
                    currentOverlay = coordinator.currentOverlay
                    isEmergencyOverlay = overlayReducer.isFullScreenEmergency()
                    pairingWidgetState = coordinator.pairingWidgetState
                }
            } catch (_: Exception) {
                adScheduler.onGatewayUnreachable()
            }
        }, 0L, 5L, TimeUnit.SECONDS)

        pool.scheduleAtFixedRate({
            adScheduler.advanceSeconds(1)
            coordinator.tickOverlay(1)
            runOnUiThread {
                currentAd = adScheduler.currentAd()
                currentOverlay = coordinator.currentOverlay
                isEmergencyOverlay = overlayReducer.isFullScreenEmergency()
            }
        }, 1L, 1L, TimeUnit.SECONDS)
    }

    override fun onDestroy() {
        executor?.shutdownNow()
        executor = null
        super.onDestroy()
    }

    override fun dispatchKeyEvent(event: KeyEvent): Boolean {
        if (keyPolicy.shouldBlockKey(event.keyCode)) {
            return true
        }
        return super.dispatchKeyEvent(event)
    }
}

@Composable
fun KioskWelcomeScreen(
    state: BackdropUiState,
    currentAd: BackdropAd? = null,
    currentOverlay: BackdropMessage? = null,
    isEmergency: Boolean = false,
    pairingWidget: PairingWidgetState = PairingWidgetState.fromGateway(0, "", "")
) {
    val bgColor = runCatching {
        Color(android.graphics.Color.parseColor(state.themeColor))
    }.getOrElse { Color(0xFF1E293B) }

    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(bgColor)
    ) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(48.dp),
            verticalArrangement = Arrangement.SpaceBetween
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Column {
                    if (state.logoUrl.isNotBlank()) {
                        Text(
                            text = "[LOGO: ${state.logoUrl}]",
                            style = MaterialTheme.typography.labelMedium,
                            color = Color.White.copy(alpha = 0.7f)
                        )
                    }
                    Text(
                        text = state.hotelName.ifBlank { "Inaply Hospitality" },
                        style = MaterialTheme.typography.headlineMedium,
                        color = Color.White
                    )
                }
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

                if (currentAd != null) {
                    Spacer(modifier = Modifier.height(20.dp))
                    val ctaQr = AdCtaQr.fromAd(currentAd)
                    Surface(color = Color.Black.copy(alpha = 0.35f)) {
                        Column(modifier = Modifier.padding(16.dp)) {
                            Text(
                                text = "Featured: ${currentAd.title}",
                                style = MaterialTheme.typography.titleMedium,
                                color = Color.White
                            )
                            if (ctaQr != null) {
                                Text(
                                    text = "Scan QR: ${ctaQr.payload}",
                                    style = MaterialTheme.typography.bodyMedium,
                                    color = Color.White.copy(alpha = 0.85f)
                                )
                            }
                        }
                    }
                }
            }

            if (state.mode != "signage") {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.Bottom
                ) {
                    Text(
                        text = "Wi-Fi: ${state.wifiSsid.ifBlank { "Hotel-Guest" }}  •  Room ${state.roomNumber}",
                        style = MaterialTheme.typography.titleMedium,
                        color = Color.White.copy(alpha = 0.9f)
                    )
                    Column(horizontalAlignment = Alignment.End) {
                        if (pairingWidget.showFullQrWidget && pairingWidget.pairingPin.isNotBlank()) {
                            Text(
                                text = "Pair PIN: ${pairingWidget.pairingPin}  •  ${pairingWidget.pairingUrl}",
                                style = MaterialTheme.typography.titleMedium,
                                color = Color.White
                            )
                        }
                        Text(
                            text = "Paired Devices: ${state.activePairedDevices}",
                            style = MaterialTheme.typography.titleMedium,
                            color = Color.White.copy(alpha = 0.9f)
                        )
                    }
                }
            }
        }

        if (currentOverlay != null) {
            val overlayBg = if (isEmergency) Color(0xFFB91C1C) else Color.Black.copy(alpha = 0.85f)
            Box(
                modifier = Modifier
                    .fillMaxSize()
                    .background(overlayBg)
                    .padding(48.dp),
                contentAlignment = Alignment.Center
            ) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Text(
                        text = currentOverlay.title,
                        style = MaterialTheme.typography.displaySmall,
                        color = Color.White
                    )
                    Spacer(modifier = Modifier.height(16.dp))
                    Text(
                        text = currentOverlay.body,
                        style = MaterialTheme.typography.headlineSmall,
                        color = Color.White.copy(alpha = 0.9f)
                    )
                }
            }
        }
    }
}
