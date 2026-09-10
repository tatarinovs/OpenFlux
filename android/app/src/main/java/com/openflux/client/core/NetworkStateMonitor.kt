package com.openflux.client.core

import android.content.Context
import android.net.ConnectivityManager
import android.net.Network
import android.net.NetworkCapabilities
import android.net.NetworkRequest
import android.os.Build
import android.util.Log

class NetworkStateMonitor(
    private val context: Context,
    private val onNetworkChanged: () -> Unit
) {
    private val connectivityManager =
        context.getSystemService(Context.CONNECTIVITY_SERVICE) as? ConnectivityManager
    private var isRegistered = false
    private var currentNetwork: Network? = null

    private val callback = object : ConnectivityManager.NetworkCallback() {
        override fun onAvailable(network: Network) {
            val prev = currentNetwork
            currentNetwork = network
            if (prev != null && prev != network) {
                Log.i("NetworkStateMonitor", "Network interface changed: $prev -> $network")
                onNetworkChanged()
            }
        }

        override fun onCapabilitiesChanged(network: Network, caps: NetworkCapabilities) {
            val hasInternet = caps.hasCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET) &&
                    caps.hasCapability(NetworkCapabilities.NET_CAPABILITY_VALIDATED)
            if (hasInternet && currentNetwork != null && currentNetwork != network) {
                Log.i("NetworkStateMonitor", "Validated new network available: $network")
                currentNetwork = network
                onNetworkChanged()
            }
        }

        override fun onLost(network: Network) {
            Log.w("NetworkStateMonitor", "Network connection lost: $network")
            if (currentNetwork == network) {
                currentNetwork = null
            }
        }
    }

    fun start() {
        if (isRegistered || connectivityManager == null) return
        try {
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.N) {
                connectivityManager.registerDefaultNetworkCallback(callback)
            } else {
                val request = NetworkRequest.Builder()
                    .addCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET)
                    .build()
                connectivityManager.registerNetworkCallback(request, callback)
            }
            isRegistered = true
            currentNetwork = connectivityManager.activeNetwork
            Log.d("NetworkStateMonitor", "Network monitoring started, active network: $currentNetwork")
        } catch (e: Exception) {
            Log.e("NetworkStateMonitor", "Failed to start network monitoring", e)
        }
    }

    fun stop() {
        if (!isRegistered || connectivityManager == null) return
        try {
            connectivityManager.unregisterNetworkCallback(callback)
            isRegistered = false
            currentNetwork = null
            Log.d("NetworkStateMonitor", "Network monitoring stopped")
        } catch (_: Exception) {}
    }
}
