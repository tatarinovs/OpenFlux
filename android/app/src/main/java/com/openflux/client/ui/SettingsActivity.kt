package com.openflux.client.ui

import android.content.Intent
import android.os.Bundle
import android.view.View
import android.widget.ArrayAdapter
import android.widget.ImageView
import android.widget.Toast
import androidx.appcompat.app.AppCompatActivity
import com.google.android.material.dialog.MaterialAlertDialogBuilder
import com.openflux.client.R
import com.openflux.client.data.AppPreferences
import com.openflux.client.data.QrConfigHelper
import com.openflux.client.databinding.ActivitySettingsBinding
import androidx.activity.result.contract.ActivityResultContracts

class SettingsActivity : AppCompatActivity() {

    private lateinit var binding: ActivitySettingsBinding
    private lateinit var prefs: AppPreferences

    private val qrScanner = registerForActivityResult(
        ActivityResultContracts.StartActivityForResult()
    ) { result ->
        if (result.resultCode == RESULT_OK) {
            val raw = result.data?.getStringExtra(ScannerActivity.EXTRA_SCAN_RESULT)
            if (!raw.isNullOrEmpty()) {
                applyScannedConfig(raw)
            }
        }
    }

    data class TransportOption(val id: String, val displayName: String)

    private val transportOptions = listOf(
        TransportOption("yandex", "Yandex Docs"),
        TransportOption("vyandex", "Yandex Volga"),
        TransportOption("cupsonline", "Cups.online (Live Coding)"),
        TransportOption("oneme", "MAX Messenger")
    )

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        binding = ActivitySettingsBinding.inflate(layoutInflater)
        setContentView(binding.root)
        binding.root.applySystemWindowInsetsPadding()

        prefs = AppPreferences(this)
        loadSettings()
        setupListeners()
    }

    private fun loadSettings() {
        val names = transportOptions.map { it.displayName }
        val adapter = ArrayAdapter(this, android.R.layout.simple_dropdown_item_1line, names)
        binding.actvTransport.setAdapter(adapter)

        val currentOpt = transportOptions.find { it.id == prefs.transportType } ?: transportOptions[0]
        binding.actvTransport.setText(currentOpt.displayName, false)
        updateCardsVisibility(currentOpt.id)

        binding.etYandexUrl.setText(prefs.yandexDocUrl)
        binding.etCupsRooms.setText(prefs.cupsRooms)
        binding.etMaxToken.setText(prefs.maxToken)
        binding.etMaxUid.setText(prefs.maxUid)
        binding.etSecretKey.setText(prefs.secretKey)
        binding.etPort.setText(prefs.socksPort.toString())
        binding.switchListenAll.isChecked = prefs.listenAll
        binding.switchDebug.isChecked = prefs.debugLogging
    }

    private fun updateCardsVisibility(transportId: String) {
        binding.cardYandexConfig.visibility =
            if (transportId == "yandex" || transportId == "vyandex") View.VISIBLE else View.GONE
        binding.cardCupsConfig.visibility =
            if (transportId == "cupsonline") View.VISIBLE else View.GONE
        binding.cardMaxConfig.visibility =
            if (transportId == "oneme") View.VISIBLE else View.GONE
    }

    private fun setupListeners() {
        binding.actvTransport.setOnItemClickListener { _, _, position, _ ->
            val selected = transportOptions[position]
            updateCardsVisibility(selected.id)
        }

        binding.btnScanQr.setOnClickListener {
            val intent = Intent(this, ScannerActivity::class.java)
            qrScanner.launch(intent)
        }

        binding.btnShareQr.setOnClickListener {
            showShareQrDialog()
        }

        binding.btnSaveSettings.setOnClickListener {
            val portText = binding.etPort.text?.toString()?.trim() ?: "1080"
            val port = portText.toIntOrNull()
            if (port == null || port < 1 || port > 65535) {
                Toast.makeText(this, "Некорректный порт (допустимо 1-65535)", Toast.LENGTH_SHORT).show()
                return@setOnClickListener
            }

            val selectedText = binding.actvTransport.text?.toString() ?: ""
            val selectedOption = transportOptions.find { it.displayName == selectedText } ?: transportOptions[0]
            val transport = selectedOption.id
            val yandexUrl = binding.etYandexUrl.text?.toString()?.trim() ?: ""
            val cupsRooms = binding.etCupsRooms.text?.toString()?.trim() ?: ""
            val secretKey = binding.etSecretKey.text?.toString()?.trim() ?: ""

            if ((transport == "yandex" || transport == "vyandex") && yandexUrl.isEmpty()) {
                Toast.makeText(this, "Укажите URL документа Yandex Docs", Toast.LENGTH_SHORT).show()
                return@setOnClickListener
            }

            if (transport == "cupsonline" && cupsRooms.isEmpty()) {
                Toast.makeText(this, "Укажите base64 строку комнат Cups.online", Toast.LENGTH_SHORT).show()
                return@setOnClickListener
            }

            if (transport == "oneme" && (binding.etMaxToken.text?.toString()?.trim() ?: "").isEmpty()) {
                Toast.makeText(this, "Укажите токен MAX", Toast.LENGTH_SHORT).show()
                return@setOnClickListener
            }

            if (secretKey.isNotEmpty() && secretKey != "none" && secretKey != "off" && secretKey.length < 16) {
                Toast.makeText(this, "Секретный ключ E2E должен содержать не менее 16 символов", Toast.LENGTH_SHORT).show()
                return@setOnClickListener
            }

            prefs.transportType = transport
            prefs.yandexDocUrl = yandexUrl
            prefs.cupsRooms = cupsRooms
            prefs.maxToken = binding.etMaxToken.text?.toString()?.trim() ?: ""
            prefs.maxUid = binding.etMaxUid.text?.toString()?.trim() ?: ""
            prefs.secretKey = secretKey
            prefs.socksPort = port
            prefs.listenAll = binding.switchListenAll.isChecked
            prefs.debugLogging = binding.switchDebug.isChecked

            Toast.makeText(this, "Настройки сохранены", Toast.LENGTH_SHORT).show()
            finish()
        }
    }

    private fun applyScannedConfig(raw: String) {
        val cfg = QrConfigHelper.parse(raw)
        if (cfg == null) {
            Toast.makeText(this, "Не удалось распознать конфигурацию QR-кода", Toast.LENGTH_LONG).show()
            return
        }

        val opt = transportOptions.find { it.id.equals(cfg.transport, ignoreCase = true) }
            ?: transportOptions[0]
        binding.actvTransport.setText(opt.displayName, false)
        updateCardsVisibility(opt.id)

        if (opt.id == "cupsonline") {
            binding.etCupsRooms.setText(cfg.target)
        } else if (opt.id == "yandex" || opt.id == "vyandex") {
            binding.etYandexUrl.setText(cfg.target)
        }

        if (cfg.secretKey != null) {
            binding.etSecretKey.setText(cfg.secretKey)
        }
        if (cfg.socksPort != null && cfg.socksPort > 0) {
            binding.etPort.setText(cfg.socksPort.toString())
        }
        if (cfg.maxToken != null) {
            binding.etMaxToken.setText(cfg.maxToken)
        }
        if (cfg.maxUid != null) {
            binding.etMaxUid.setText(cfg.maxUid)
        }

        Toast.makeText(this, "Конфигурация импортирована из QR-кода", Toast.LENGTH_SHORT).show()
    }

    private fun showShareQrDialog() {
        val selectedText = binding.actvTransport.text?.toString() ?: ""
        val opt = transportOptions.find { it.displayName == selectedText } ?: transportOptions[0]
        val transport = opt.id
        val target = if (transport == "cupsonline") {
            binding.etCupsRooms.text?.toString()?.trim() ?: ""
        } else {
            binding.etYandexUrl.text?.toString()?.trim() ?: ""
        }
        val secretKey = binding.etSecretKey.text?.toString()?.trim() ?: ""
        val port = binding.etPort.text?.toString()?.trim()?.toIntOrNull() ?: 1080

        if (target.isEmpty() && transport != "oneme") {
            Toast.makeText(this, "Заполните целевой URL или комнаты перед созданием QR-кода", Toast.LENGTH_SHORT).show()
            return
        }

        val jsonStr = QrConfigHelper.buildJson(transport, target, secretKey, port)
        val bitmap = try {
            QrConfigHelper.generateQrBitmap(jsonStr, 600)
        } catch (e: Exception) {
            Toast.makeText(this, "Ошибка создания QR: ${e.message}", Toast.LENGTH_SHORT).show()
            return
        }

        val imageView = ImageView(this).apply {
            setImageBitmap(bitmap)
            adjustViewBounds = true
            setPadding(32, 24, 32, 24)
        }

        MaterialAlertDialogBuilder(this)
            .setTitle("QR-код конфигурации (${opt.displayName})")
            .setView(imageView)
            .setPositiveButton("Поделиться текстом") { _, _ ->
                val sendIntent = Intent().apply {
                    action = Intent.ACTION_SEND
                    putExtra(Intent.EXTRA_TEXT, jsonStr)
                    type = "text/plain"
                }
                startActivity(Intent.createChooser(sendIntent, "Поделиться конфигурацией OpenFlux"))
            }
            .setNegativeButton("Закрыть", null)
            .show()
    }
}
