package com.openflux.client.ui

import android.app.Activity
import android.content.BroadcastReceiver
import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.net.VpnService
import android.os.Build
import android.os.Bundle
import android.widget.Toast
import androidx.activity.result.contract.ActivityResultContracts
import androidx.appcompat.app.AppCompatActivity
import androidx.core.content.ContextCompat
import com.openflux.client.R
import com.openflux.client.core.OpenFluxCore
import com.openflux.client.core.PingHelper
import com.openflux.client.data.AppPreferences
import com.openflux.client.databinding.ActivityMainBinding
import android.Manifest
import android.content.pm.PackageManager
import androidx.lifecycle.lifecycleScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import com.openflux.client.service.OpenFluxProxyService
import com.openflux.client.service.OpenFluxVpnService

class MainActivity : AppCompatActivity() {

    private lateinit var binding: ActivityMainBinding
    private lateinit var prefs: AppPreferences

    private val notificationPermissionLauncher = registerForActivityResult(
        ActivityResultContracts.RequestPermission()
    ) { isGranted ->
        if (!isGranted) {
            Toast.makeText(this, "Уведомления отключены", Toast.LENGTH_SHORT).show()
        }
    }

    private val vpnLauncher = registerForActivityResult(
        ActivityResultContracts.StartActivityForResult()
    ) { result ->
        if (result.resultCode == Activity.RESULT_OK) {
            startVpnService()
        } else {
            Toast.makeText(this, "Требуется разрешение для создания VPN-туннеля", Toast.LENGTH_SHORT).show()
            updateUiState(OpenFluxProxyService.STATE_DISCONNECTED)
        }
    }

    private val statusReceiver = object : BroadcastReceiver() {
        override fun onReceive(context: Context?, intent: Intent?) {
            val state = intent?.getStringExtra(OpenFluxProxyService.EXTRA_STATE)
            if (state != null) {
                updateUiState(state)
            }
        }
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        binding = ActivityMainBinding.inflate(layoutInflater)
        setContentView(binding.root)
        binding.root.applySystemWindowInsetsPadding()

        prefs = AppPreferences(this)

        requestNotificationPermission()
        setupModeSwitch()
        setupListeners()
    }

    private fun requestNotificationPermission() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            if (ContextCompat.checkSelfPermission(this, Manifest.permission.POST_NOTIFICATIONS) != PackageManager.PERMISSION_GRANTED) {
                notificationPermissionLauncher.launch(Manifest.permission.POST_NOTIFICATIONS)
            }
        }
    }

    override fun onResume() {
        super.onResume()
        OpenFluxCore.syncTimezone()
        updateProxyInfo()
        updateModeDetails()

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

        val running = OpenFluxVpnService.isRunning || OpenFluxProxyService.isRunning
        if (running) {
            updateUiState(OpenFluxProxyService.STATE_CONNECTED)
        } else {
            updateUiState(OpenFluxProxyService.STATE_DISCONNECTED)
        }
    }

    private var uiStatsJob: kotlinx.coroutines.Job? = null

    override fun onPause() {
        super.onPause()
        uiStatsJob?.cancel()
        uiStatsJob = null
        try {
            unregisterReceiver(statusReceiver)
        } catch (_: Exception) {}
    }

    private fun setupModeSwitch() {
        val isProxyOnly = !prefs.isVpnMode
        val checkedId = if (isProxyOnly) R.id.btnModeProxy else R.id.btnModeVpn
        binding.toggleGroupMode.check(checkedId)
        updateModeDetails()

        binding.toggleGroupMode.addOnButtonCheckedListener { _, buttonId, isChecked ->
            if (isChecked) {
                prefs.isVpnMode = (buttonId == R.id.btnModeVpn)
                updateModeDetails()
                updateProxyInfo()
            }
        }
    }

    private fun getTransportDisplayName(): String {
        return when (prefs.transportType) {
            "vyandex" -> "Yandex Volga"
            "cupsonline" -> "Cups.online"
            "oneme" -> "MAX Messenger"
            else -> "Yandex Docs"
        }
    }

    private fun updateModeDetails() {
        val transName = getTransportDisplayName()
        val modeName = if (prefs.isVpnMode) "VPN" else "Прокси"
        binding.tvStatsDetails.text = "Режим: $modeName | Транспорт: $transName"
        binding.tvStatsDetails.setTextColor(ContextCompat.getColor(this, R.color.text_secondary))
    }

    private fun setupListeners() {
        binding.btnSettings.setOnClickListener {
            startActivity(Intent(this, SettingsActivity::class.java))
        }

        binding.btnSplitTunnel.setOnClickListener {
            startActivity(Intent(this, SplitTunnelActivity::class.java))
        }

        binding.btnLogs.setOnClickListener {
            startActivity(Intent(this, LogActivity::class.java))
        }

        binding.btnCopyProxy.setOnClickListener {
            val addr = binding.tvProxyAddress.text.toString()
            val clipboard = getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
            clipboard.setPrimaryClip(ClipData.newPlainText("SOCKS5 Proxy", addr))
            Toast.makeText(this, "Адрес скопирован: $addr", Toast.LENGTH_SHORT).show()
        }

        binding.btnConnect.setOnClickListener {
            val isRunning = OpenFluxVpnService.isRunning || OpenFluxProxyService.isRunning
            if (isRunning) {
                disconnect()
            } else {
                connect()
            }
        }
    }

    private fun connect() {
        // Validation
        val isYandexFamily = prefs.transportType == "yandex" || prefs.transportType == "vyandex"
        if (isYandexFamily && prefs.yandexDocUrl.isEmpty()) {
            Toast.makeText(this, "Пожалуйста, настройте URL документа в Настройках", Toast.LENGTH_LONG).show()
            startActivity(Intent(this, SettingsActivity::class.java))
            return
        }
        if (prefs.transportType == "cupsonline" && prefs.cupsRooms.isEmpty()) {
            Toast.makeText(this, "Пожалуйста, настройте комнаты Cups.online в Настройках", Toast.LENGTH_LONG).show()
            startActivity(Intent(this, SettingsActivity::class.java))
            return
        }
        if (prefs.transportType == "oneme" && prefs.maxToken.isEmpty()) {
            Toast.makeText(this, "Пожалуйста, настройте токен MAX в Настройках", Toast.LENGTH_LONG).show()
            startActivity(Intent(this, SettingsActivity::class.java))
            return
        }

        updateUiState(OpenFluxProxyService.STATE_CONNECTING)

        if (prefs.isVpnMode) {
            val prepareIntent = VpnService.prepare(this)
            if (prepareIntent != null) {
                vpnLauncher.launch(prepareIntent)
            } else {
                startVpnService()
            }
        } else {
            OpenFluxProxyService.start(this)
        }
    }

    private fun startVpnService() {
        OpenFluxVpnService.start(this)
    }

    private fun disconnect() {
        binding.btnConnect.isEnabled = false
        lifecycleScope.launch(Dispatchers.IO) {
            if (OpenFluxVpnService.isRunning) {
                OpenFluxVpnService.stop(this@MainActivity)
            } else if (OpenFluxProxyService.isRunning) {
                OpenFluxProxyService.stop(this@MainActivity)
            } else {
                OpenFluxCore.stop()
            }
            withContext(Dispatchers.Main) {
                binding.btnConnect.isEnabled = true
                updateUiState(OpenFluxProxyService.STATE_DISCONNECTED)
            }
        }
    }

    private fun startUiStatsLoop() {
        uiStatsJob?.cancel()
        uiStatsJob = lifecycleScope.launch(Dispatchers.IO) {
            var counter = 0
            var lastPing: Long? = null
            var isPinging = false

            while (lifecycle.currentState.isAtLeast(androidx.lifecycle.Lifecycle.State.RESUMED) &&
                (OpenFluxVpnService.isRunning || OpenFluxProxyService.isRunning)) {
                
                if (counter % 3 == 0 && !isPinging) {
                    isPinging = true
                    lifecycleScope.launch(Dispatchers.IO) {
                        try {
                            lastPing = PingHelper.measurePing(prefs.socksPort)
                        } finally {
                            isPinging = false
                        }
                    }
                }
                counter++

                val stats = OpenFluxCore.getStats()
                val currentPing = lastPing
                val pingStr = if (currentPing != null) {
                    val indicator = when {
                        currentPing <= 250 -> "🟢"
                        currentPing <= 500 -> "🟡"
                        else -> "🔴"
                    }
                    "$indicator $currentPing мс"
                } else if (isPinging && counter <= 3) {
                    "измерение..."
                } else {
                    "--"
                }

                val transName = getTransportDisplayName()
                val fullStats = if (stats.isNotEmpty() && stats != "Stopped") {
                    if (stats.contains("Время: ")) {
                        val lastIdx = stats.lastIndexOf("Время: ")
                        val mainPart = stats.substring(0, lastIdx).trimEnd()
                        val uptimeStr = stats.substring(lastIdx)
                        "Подключено ($transName)\n$mainPart\nПинг: $pingStr | $uptimeStr"
                    } else {
                        "Подключено ($transName)\n$stats\nПинг: $pingStr"
                    }
                } else {
                    "Подключено ($transName)\nПинг: $pingStr"
                }

                withContext(Dispatchers.Main) {
                    binding.tvStatsDetails.text = fullStats
                    binding.tvStatsDetails.setTextColor(ContextCompat.getColor(this@MainActivity, R.color.text_secondary))
                }
                kotlinx.coroutines.delay(1000L)
            }
        }
    }

    private fun updateUiState(state: String) {
        val isRunning = state == OpenFluxProxyService.STATE_CONNECTED || state == OpenFluxProxyService.STATE_CONNECTING
        binding.btnModeVpn.isEnabled = !isRunning
        binding.btnModeProxy.isEnabled = !isRunning

        when (state) {
            OpenFluxProxyService.STATE_CONNECTED -> {
                binding.btnConnect.backgroundTintList = ContextCompat.getColorStateList(this, R.color.status_connected)
                startUiStatsLoop()
            }
            OpenFluxProxyService.STATE_CONNECTING -> {
                uiStatsJob?.cancel()
                uiStatsJob = null
                binding.btnConnect.backgroundTintList = ContextCompat.getColorStateList(this, R.color.status_connecting)
                val transName = getTransportDisplayName()
                binding.tvStatsDetails.text = "Подключение к $transName..."
                binding.tvStatsDetails.setTextColor(ContextCompat.getColor(this, R.color.status_connecting))
            }
            OpenFluxProxyService.STATE_ERROR -> {
                uiStatsJob?.cancel()
                uiStatsJob = null
                binding.btnConnect.backgroundTintList = ContextCompat.getColorStateList(this, R.color.status_disconnected)
                binding.tvStatsDetails.text = "Ошибка подключения"
                binding.tvStatsDetails.setTextColor(ContextCompat.getColor(this, R.color.status_disconnected))
            }
            else -> {
                uiStatsJob?.cancel()
                uiStatsJob = null
                binding.btnConnect.backgroundTintList = ContextCompat.getColorStateList(this, R.color.primary)
                updateModeDetails()
            }
        }
    }

    private fun updateProxyInfo() {
        val host = if (prefs.listenAll) "0.0.0.0" else "127.0.0.1"
        binding.tvProxyAddress.text = "$host:${prefs.socksPort}"
    }
}
