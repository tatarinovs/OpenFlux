package com.openflux.client.data

import android.content.Context
import android.content.SharedPreferences
import com.openflux.client.BuildConfig

class AppPreferences(context: Context) {

    private val prefs: SharedPreferences =
        context.getSharedPreferences("openflux_prefs", Context.MODE_PRIVATE)

    var isVpnMode: Boolean
        get() = prefs.getBoolean(KEY_IS_VPN_MODE, false) // Default to Proxy mode as requested
        set(value) = prefs.edit().putBoolean(KEY_IS_VPN_MODE, value).apply()

    var transportType: String
        get() = prefs.getString(KEY_TRANSPORT_TYPE, "yandex") ?: "yandex"
        set(value) = prefs.edit().putString(KEY_TRANSPORT_TYPE, value).apply()

    var yandexDocUrl: String
        get() = prefs.getString(KEY_YANDEX_DOC_URL, "") ?: ""
        set(value) = prefs.edit().putString(KEY_YANDEX_DOC_URL, value).apply()

    var maxToken: String
        get() = prefs.getString(KEY_MAX_TOKEN, "") ?: ""
        set(value) = prefs.edit().putString(KEY_MAX_TOKEN, value).apply()

    var maxUid: String
        get() = prefs.getString(KEY_MAX_UID, "") ?: ""
        set(value) = prefs.edit().putString(KEY_MAX_UID, value).apply()

    var socksPort: Int
        get() = prefs.getInt(KEY_SOCKS_PORT, 1080)
        set(value) = prefs.edit().putInt(KEY_SOCKS_PORT, value).apply()

    var listenAll: Boolean
        get() = prefs.getBoolean(KEY_LISTEN_ALL, false)
        set(value) = prefs.edit().putBoolean(KEY_LISTEN_ALL, value).apply()

    var debugLogging: Boolean
        get() = prefs.getBoolean(KEY_DEBUG_LOGGING, false)
        set(value) = prefs.edit().putBoolean(KEY_DEBUG_LOGGING, value).apply()

    var splitTunnelMode: String
        get() = prefs.getString(KEY_SPLIT_MODE, MODE_WHITELIST) ?: MODE_WHITELIST
        set(value) = prefs.edit().putString(KEY_SPLIT_MODE, value).apply()

    var selectedPackages: Set<String>
        get() = prefs.getStringSet(KEY_SELECTED_PACKAGES, emptySet()) ?: emptySet()
        set(value) = prefs.edit().putStringSet(KEY_SELECTED_PACKAGES, value).apply()

    var showSystemApps: Boolean
        get() = prefs.getBoolean(KEY_SHOW_SYSTEM_APPS, false)
        set(value) = prefs.edit().putBoolean(KEY_SHOW_SYSTEM_APPS, value).apply()

    var secretKey: String
        get() = prefs.getString(KEY_SECRET_KEY, DEFAULT_SECRET_KEY) ?: DEFAULT_SECRET_KEY
        set(value) = prefs.edit().putString(KEY_SECRET_KEY, value).apply()

    companion object {
        val DEFAULT_SECRET_KEY: String = BuildConfig.DEFAULT_SECRET_KEY

        const val MODE_WHITELIST = "WHITELIST"
        const val MODE_BLACKLIST = "BLACKLIST"

        private const val KEY_IS_VPN_MODE = "is_vpn_mode"
        private const val KEY_TRANSPORT_TYPE = "transport_type"
        private const val KEY_YANDEX_DOC_URL = "yandex_doc_url"
        private const val KEY_MAX_TOKEN = "max_token"
        private const val KEY_MAX_UID = "max_uid"
        private const val KEY_SECRET_KEY = "secret_key"
        private const val KEY_SOCKS_PORT = "socks_port"
        private const val KEY_LISTEN_ALL = "listen_all"
        private const val KEY_DEBUG_LOGGING = "debug_logging"
        private const val KEY_SPLIT_MODE = "split_mode"
        private const val KEY_SELECTED_PACKAGES = "selected_packages"
        private const val KEY_SHOW_SYSTEM_APPS = "show_system_apps"
    }
}
