package com.openflux.client.service

import android.app.PendingIntent
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.net.VpnService
import android.os.Build
import android.service.quicksettings.Tile
import android.service.quicksettings.TileService
import androidx.core.content.ContextCompat
import com.openflux.client.R
import com.openflux.client.data.AppPreferences
import com.openflux.client.ui.MainActivity

class OpenFluxTileService : TileService() {

    private var isReceiverRegistered = false

    private val statusReceiver = object : BroadcastReceiver() {
        override fun onReceive(context: Context?, intent: Intent?) {
            updateTileState()
        }
    }

    override fun onStartListening() {
        super.onStartListening()
        registerStatusReceiver()
        updateTileState()
    }

    override fun onStopListening() {
        super.onStopListening()
        unregisterStatusReceiver()
    }

    override fun onClick() {
        super.onClick()
        val isRunning = OpenFluxVpnService.isRunning || OpenFluxProxyService.isRunning
        val prefs = AppPreferences(this)

        if (isRunning) {
            if (OpenFluxVpnService.isRunning) {
                OpenFluxVpnService.stop(this)
            }
            if (OpenFluxProxyService.isRunning) {
                OpenFluxProxyService.stop(this)
            }
            updateTileState()
        } else {
            if (prefs.isVpnMode) {
                val prepareIntent = VpnService.prepare(this)
                if (prepareIntent != null) {
                    launchMainActivity()
                } else {
                    OpenFluxVpnService.start(this)
                }
            } else {
                OpenFluxProxyService.start(this)
            }
            updateTileState()
        }
    }

    private fun launchMainActivity() {
        val intent = Intent(this, MainActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP
        }
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.UPSIDE_DOWN_CAKE) {
            val pendingIntent = PendingIntent.getActivity(
                this,
                0,
                intent,
                PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
            )
            startActivityAndCollapse(pendingIntent)
        } else {
            @Suppress("DEPRECATION")
            startActivityAndCollapse(intent)
        }
    }

    private fun updateTileState() {
        val tile = qsTile ?: return
        val isRunning = OpenFluxVpnService.isRunning || OpenFluxProxyService.isRunning

        tile.state = if (isRunning) Tile.STATE_ACTIVE else Tile.STATE_INACTIVE
        tile.label = getString(R.string.qs_tile_label)

        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            val subtitle = if (isRunning) {
                val prefs = AppPreferences(this)
                if (prefs.isVpnMode) "VPN" else "SOCKS5"
            } else {
                getString(R.string.qs_tile_inactive)
            }
            tile.subtitle = subtitle
        }

        tile.updateTile()
    }

    private fun registerStatusReceiver() {
        if (!isReceiverRegistered) {
            val filter = IntentFilter().apply {
                addAction(OpenFluxProxyService.ACTION_STATUS_CHANGED)
                addAction(OpenFluxVpnService.ACTION_STATUS_CHANGED)
            }
            ContextCompat.registerReceiver(
                this,
                statusReceiver,
                filter,
                ContextCompat.RECEIVER_NOT_EXPORTED
            )
            isReceiverRegistered = true
        }
    }

    private fun unregisterStatusReceiver() {
        if (isReceiverRegistered) {
            try {
                unregisterReceiver(statusReceiver)
            } catch (_: Exception) {}
            isReceiverRegistered = false
        }
    }
}
