package com.openflux.client.service

import android.content.Context
import android.content.Intent
import android.net.VpnService
import android.os.ParcelFileDescriptor
import com.openflux.client.core.OpenFluxCore
import com.openflux.client.data.AppPreferences
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.cancel
import kotlinx.coroutines.launch

class OpenFluxVpnService : VpnService() {

    private val serviceScope = CoroutineScope(Dispatchers.IO + Job())
    private lateinit var prefs: AppPreferences
    private var vpnInterface: ParcelFileDescriptor? = null

    override fun onCreate() {
        super.onCreate()
        prefs = AppPreferences(this)
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        val action = intent?.action
        if (action == ACTION_STOP) {
            stopVpn()
            return START_NOT_STICKY
        }

        startVpnForeground()
        return START_STICKY
    }

    private fun startVpnForeground() {
        isRunning = true
        broadcastState(STATE_CONNECTING)

        val stopIntent = Intent(this, OpenFluxVpnService::class.java).apply {
            action = ACTION_STOP
        }
        val notification = NotificationHelper.buildNotification(
            this,
            isVpn = true,
            port = prefs.socksPort,
            stopIntent = stopIntent
        )
        startForeground(NotificationHelper.NOTIFICATION_ID, notification)

        serviceScope.launch {
            try {
                val builder = Builder()
                    .setSession("OpenFlux")
                    .setMtu(1500)
                    .addAddress("10.0.0.2", 24)
                    .addRoute("0.0.0.0", 0)
                    .addDnsServer("1.1.1.1")
                    .addDnsServer("8.8.8.8")

                // Split Tunneling Configuration
                val selected = prefs.selectedPackages
                if (prefs.splitTunnelMode == AppPreferences.MODE_WHITELIST && selected.isNotEmpty()) {
                    for (pkg in selected) {
                        try {
                            builder.addAllowedApplication(pkg)
                        } catch (_: Exception) {}
                    }
                } else {
                    // Blacklist mode: tunnel everything EXCEPT selected apps
                    for (pkg in selected) {
                        try {
                            builder.addDisallowedApplication(pkg)
                        } catch (_: Exception) {}
                    }
                    // Crucial: Disallow OpenFlux itself so its outbound transport doesn't loop
                    try {
                        builder.addDisallowedApplication(packageName)
                    } catch (_: Exception) {}
                }

                val pfd = builder.establish()
                if (pfd == null) {
                    broadcastState(STATE_ERROR)
                    stopSelf()
                    return@launch
                }
                vpnInterface = pfd

                val dupPfd = pfd.dup()
                val tunFd = dupPfd.detachFd()

                val res = kotlinx.coroutines.withTimeoutOrNull(15000L) {
                    OpenFluxCore.startVpn(
                        tunFd = tunFd,
                        transportType = prefs.transportType,
                        url = prefs.yandexDocUrl,
                        maxToken = prefs.maxToken,
                        maxUid = prefs.maxUid,
                        secretKey = prefs.secretKey,
                        port = prefs.socksPort,
                        debug = prefs.debugLogging
                    )
                }

                if (res != null && res == 0) {
                    broadcastState(STATE_CONNECTED)
                    startStatsLoop()
                } else {
                    broadcastState(STATE_ERROR)
                    stopVpn()
                }
            } catch (e: Exception) {
                e.printStackTrace()
                broadcastState(STATE_ERROR)
                stopVpn()
            }
        }
    }

    private var statsJob: Job? = null
    private val isStopping = java.util.concurrent.atomic.AtomicBoolean(false)

    private fun startStatsLoop() {
        statsJob?.cancel()
        statsJob = serviceScope.launch {
            val stopIntent = Intent(this@OpenFluxVpnService, OpenFluxVpnService::class.java).apply {
                action = ACTION_STOP
            }
            while (isRunning && OpenFluxCore.isRunning()) {
                kotlinx.coroutines.delay(1000L)
                try {
                    val stats = OpenFluxCore.getTrafficStats()
                    if (stats.isNotEmpty()) {
                        NotificationHelper.updateNotification(
                            context = this@OpenFluxVpnService,
                            isVpn = true,
                            port = prefs.socksPort,
                            stats = stats,
                            stopIntent = stopIntent
                        )
                    }
                } catch (e: Exception) {
                    e.printStackTrace()
                }
            }
        }
    }

    private fun stopVpn() {
        if (!isStopping.compareAndSet(false, true)) {
            return
        }
        statsJob?.cancel()
        statsJob = null
        isRunning = false

        serviceScope.launch {
            try {
                OpenFluxCore.stop()
            } catch (e: Throwable) {
                android.util.Log.e("OpenFluxVpn", "Error stopping core", e)
            }
            try {
                vpnInterface?.close()
            } catch (e: Throwable) {
                android.util.Log.e("OpenFluxVpn", "Error closing vpnInterface", e)
            } finally {
                vpnInterface = null
            }
            broadcastState(STATE_DISCONNECTED)
            try {
                stopForeground(STOP_FOREGROUND_REMOVE)
            } catch (_: Throwable) {}
            stopSelf()
            isStopping.set(false)
        }
    }

    override fun onDestroy() {
        isRunning = false
        statsJob?.cancel()
        statsJob = null
        serviceScope.cancel()
        try {
            vpnInterface?.close()
        } catch (_: Throwable) {}
        vpnInterface = null
        broadcastState(STATE_DISCONNECTED)
        super.onDestroy()
    }

    private fun broadcastState(state: String) {
        val intent = Intent(ACTION_STATUS_CHANGED).apply {
            putExtra(EXTRA_STATE, state)
            putExtra(EXTRA_MODE, MODE_VPN)
            setPackage(packageName)
        }
        sendBroadcast(intent)
    }

    companion object {
        const val ACTION_START = "com.openflux.client.action.START_VPN"
        const val ACTION_STOP = "com.openflux.client.action.STOP_VPN"
        const val ACTION_STATUS_CHANGED = "com.openflux.client.action.STATUS_CHANGED"

        const val EXTRA_STATE = "extra_state"
        const val EXTRA_MODE = "extra_mode"

        const val STATE_DISCONNECTED = "DISCONNECTED"
        const val STATE_CONNECTING = "CONNECTING"
        const val STATE_CONNECTED = "CONNECTED"
        const val STATE_ERROR = "ERROR"

        const val MODE_VPN = "VPN"

        var isRunning = false
            private set

        fun start(context: Context) {
            val intent = Intent(context, OpenFluxVpnService::class.java).apply {
                action = ACTION_START
            }
            if (android.os.Build.VERSION.SDK_INT >= android.os.Build.VERSION_CODES.O) {
                context.startForegroundService(intent)
            } else {
                context.startService(intent)
            }
        }

        fun stop(context: Context) {
            val intent = Intent(context, OpenFluxVpnService::class.java).apply {
                action = ACTION_STOP
            }
            context.startService(intent)
        }
    }
}
