package com.openflux.client.core

import java.net.InetSocketAddress
import java.net.Proxy
import java.net.Socket
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

object PingHelper {
    private val TARGETS = listOf(
        Pair("1.1.1.1", 80),
        Pair("1.0.0.1", 80),
        Pair("8.8.8.8", 53)
    )
    private const val TIMEOUT_MS = 2000

    /**
     * Measures latency (RTT in milliseconds) by connecting to an internet target
     * through the local OpenFlux SOCKS5 proxy.
     *
     * Returns RTT in ms, or null if unreachable / timed out.
     */
    suspend fun measurePing(socksPort: Int): Long? = withContext(Dispatchers.IO) {
        for ((host, port) in TARGETS) {
            val rtt = tryConnect(socksPort, host, port)
            if (rtt != null) return@withContext rtt
        }
        null
    }

    private fun tryConnect(socksPort: Int, host: String, port: Int): Long? {
        var socket: Socket? = null
        return try {
            val proxy = Proxy(Proxy.Type.SOCKS, InetSocketAddress("127.0.0.1", socksPort))
            socket = Socket(proxy)
            val startTime = System.currentTimeMillis()
            socket.connect(InetSocketAddress(host, port), TIMEOUT_MS)
            System.currentTimeMillis() - startTime
        } catch (_: Throwable) {
            null
        } finally {
            try {
                socket?.close()
            } catch (_: Throwable) {}
        }
    }
}
