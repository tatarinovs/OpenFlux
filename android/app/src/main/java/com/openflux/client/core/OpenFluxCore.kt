package com.openflux.client.core

object OpenFluxCore {

    init {
        try {
            System.loadLibrary("openflux")
            syncTimezone()
        } catch (e: UnsatisfiedLinkError) {
            e.printStackTrace()
        }
    }

    external fun setTimezoneOffset(offsetSeconds: Int)

    fun syncTimezone() {
        try {
            val offsetSeconds = java.util.TimeZone.getDefault().getOffset(System.currentTimeMillis()) / 1000
            setTimezoneOffset(offsetSeconds)
        } catch (e: Throwable) {
            // Ignored if native library not loaded yet or method not found
        }
    }

    external fun startProxy(
        transportType: String,
        url: String,
        maxToken: String,
        maxUid: String,
        secretKey: String,
        port: Int,
        listenAll: Boolean,
        debug: Boolean
    ): Int

    external fun startVpn(
        tunFd: Int,
        transportType: String,
        url: String,
        maxToken: String,
        maxUid: String,
        secretKey: String,
        port: Int,
        debug: Boolean
    ): Int

    external fun stop(): Int

    external fun isRunning(): Boolean

    external fun getStats(): String

    external fun getRecentLogs(): String

    external fun getTrafficStats(): String
}
