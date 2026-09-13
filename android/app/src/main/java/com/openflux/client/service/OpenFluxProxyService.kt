package com.openflux.client.service

import android.app.Service
import android.content.Context
import android.content.Intent
import android.os.IBinder
import com.openflux.client.core.OpenFluxCore
import com.openflux.client.data.AppPreferences
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.cancel
import kotlinx.coroutines.launch

class OpenFluxProxyService : Service() {

    private val serviceScope = CoroutineScope(Dispatchers.IO + Job())
    private lateinit var prefs: AppPreferences

    override fun onCreate() {
        super.onCreate()
        prefs = AppPreferences(this)
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        val action = intent?.action
        if (action == ACTION_STOP) {
            stopService()
            return START_NOT_STICKY
        }

        startProxyForeground()
        return START_STICKY
    }

    private var statsJob: Job? = null

    private fun startProxyForeground() {
        isRunning = true
        broadcastState(STATE_CONNECTING)

        val stopIntent = Intent(this, OpenFluxProxyService::class.java).apply {
            action = ACTION_STOP
        }
        val notification = NotificationHelper.buildNotification(
            this,
            isVpn = false,
            port = prefs.socksPort,
            stopIntent = stopIntent
        )
        startForeground(NotificationHelper.NOTIFICATION_ID, notification)

        serviceScope.launch {
            OpenFluxCore.syncTimezone()
            val targetUrl = if (prefs.transportType == "cupsonline") prefs.cupsRooms else prefs.yandexDocUrl
            val res = kotlinx.coroutines.withTimeoutOrNull(15000L) {
                OpenFluxCore.startProxy(
                    transportType = prefs.transportType,
                    url = targetUrl,
                    maxToken = prefs.maxToken,
                    maxUid = prefs.maxUid,
                    secretKey = prefs.secretKey,
                    port = prefs.socksPort,
                    listenAll = prefs.listenAll,
                    debug = prefs.debugLogging
                )
            }

            if (res != null && res == 0) {
                broadcastState(STATE_CONNECTED)
                startStatsLoop()
                startNetworkMonitor()
            } else {
                isRunning = false
                OpenFluxCore.stop()
                broadcastState(STATE_ERROR)
                stopForeground(STOP_FOREGROUND_REMOVE)
                stopSelf()
            }
        }
    }

    private var networkMonitor: com.openflux.client.core.NetworkStateMonitor? = null

    private fun startNetworkMonitor() {
        networkMonitor?.stop()
        networkMonitor = com.openflux.client.core.NetworkStateMonitor(this) {
            if (isRunning) {
                serviceScope.launch {
                    broadcastState(STATE_CONNECTING)
                    OpenFluxCore.stop()
                    kotlinx.coroutines.delay(600L)
                    OpenFluxCore.syncTimezone()
                    val targetUrl = if (prefs.transportType == "cupsonline") prefs.cupsRooms else prefs.yandexDocUrl
                    val res = OpenFluxCore.startProxy(
                        transportType = prefs.transportType,
                        url = targetUrl,
                        maxToken = prefs.maxToken,
                        maxUid = prefs.maxUid,
                        secretKey = prefs.secretKey,
                        port = prefs.socksPort,
                        listenAll = prefs.listenAll,
                        debug = prefs.debugLogging
                    )
                    if (res == 0) {
                        broadcastState(STATE_CONNECTED)
                    } else {
                        broadcastState(STATE_ERROR)
                    }
                }
            }
        }
        networkMonitor?.start()
    }

    private fun startStatsLoop() {
        statsJob?.cancel()
        statsJob = serviceScope.launch {
            val stopIntent = Intent(this@OpenFluxProxyService, OpenFluxProxyService::class.java).apply {
                action = ACTION_STOP
            }
            while (isRunning && OpenFluxCore.isRunning()) {
                kotlinx.coroutines.delay(1000L)
                try {
                    val stats = OpenFluxCore.getTrafficStats()
                    if (stats.isNotEmpty()) {
                        NotificationHelper.updateNotification(
                            context = this@OpenFluxProxyService,
                            isVpn = false,
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

    private fun stopService() {
        networkMonitor?.stop()
        networkMonitor = null
        statsJob?.cancel()
        statsJob = null
        serviceScope.launch {
            OpenFluxCore.stop()
            broadcastState(STATE_DISCONNECTED)
            stopForeground(STOP_FOREGROUND_REMOVE)
            stopSelf()
        }
    }

    override fun onDestroy() {
        isRunning = false
        networkMonitor?.stop()
        networkMonitor = null
        statsJob?.cancel()
        statsJob = null
        serviceScope.cancel()
        OpenFluxCore.stop()
        broadcastState(STATE_DISCONNECTED)
        super.onDestroy()
    }

    override fun onBind(intent: Intent?): IBinder? = null

    private fun broadcastState(state: String) {
        val intent = Intent(ACTION_STATUS_CHANGED).apply {
            putExtra(EXTRA_STATE, state)
            putExtra(EXTRA_MODE, MODE_PROXY)
            setPackage(packageName)
        }
        sendBroadcast(intent)
    }

    companion object {
        const val ACTION_START = "com.openflux.client.action.START_PROXY"
        const val ACTION_STOP = "com.openflux.client.action.STOP_PROXY"
        const val ACTION_STATUS_CHANGED = "com.openflux.client.action.STATUS_CHANGED"

        const val EXTRA_STATE = "extra_state"
        const val EXTRA_MODE = "extra_mode"

        const val STATE_DISCONNECTED = "DISCONNECTED"
        const val STATE_CONNECTING = "CONNECTING"
        const val STATE_CONNECTED = "CONNECTED"
        const val STATE_ERROR = "ERROR"

        const val MODE_PROXY = "PROXY"

        var isRunning = false
            private set

        fun start(context: Context) {
            val intent = Intent(context, OpenFluxProxyService::class.java).apply {
                action = ACTION_START
            }
            if (android.os.Build.VERSION.SDK_INT >= android.os.Build.VERSION_CODES.O) {
                context.startForegroundService(intent)
            } else {
                context.startService(intent)
            }
        }

        fun stop(context: Context) {
            val intent = Intent(context, OpenFluxProxyService::class.java).apply {
                action = ACTION_STOP
            }
            context.startService(intent)
        }
    }
}
