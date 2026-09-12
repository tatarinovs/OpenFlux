package com.openflux.client.ui

import android.os.Bundle
import android.view.View
import android.widget.ArrayAdapter
import android.widget.Toast
import androidx.appcompat.app.AppCompatActivity
import com.openflux.client.R
import com.openflux.client.data.AppPreferences
import com.openflux.client.databinding.ActivitySettingsBinding

class SettingsActivity : AppCompatActivity() {

    private lateinit var binding: ActivitySettingsBinding
    private lateinit var prefs: AppPreferences

    data class TransportOption(val id: String, val displayName: String)

    private val transportOptions = listOf(
        TransportOption("yandex", "Yandex Docs"),
        TransportOption("vyandex", "Yandex Volga"),
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
        binding.etMaxToken.setText(prefs.maxToken)
        binding.etMaxUid.setText(prefs.maxUid)
        binding.etSecretKey.setText(prefs.secretKey)
        binding.etPort.setText(prefs.socksPort.toString())
        binding.switchListenAll.isChecked = prefs.listenAll
        binding.switchDebug.isChecked = prefs.debugLogging
    }

    private fun updateCardsVisibility(transportId: String) {
        if (transportId == "oneme") {
            binding.cardYandexConfig.visibility = View.GONE
            binding.cardMaxConfig.visibility = View.VISIBLE
        } else {
            binding.cardYandexConfig.visibility = View.VISIBLE
            binding.cardMaxConfig.visibility = View.GONE
        }
    }

    private fun setupListeners() {
        binding.actvTransport.setOnItemClickListener { _, _, position, _ ->
            val selected = transportOptions[position]
            updateCardsVisibility(selected.id)
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

            if ((transport == "yandex" || transport == "vyandex") && yandexUrl.isEmpty()) {
                Toast.makeText(this, "Укажите URL документа Yandex Docs", Toast.LENGTH_SHORT).show()
                return@setOnClickListener
            }

            if (transport == "oneme" && (binding.etMaxToken.text?.toString()?.trim() ?: "").isEmpty()) {
                Toast.makeText(this, "Укажите токен MAX", Toast.LENGTH_SHORT).show()
                return@setOnClickListener
            }

            prefs.transportType = transport
            prefs.yandexDocUrl = yandexUrl
            prefs.maxToken = binding.etMaxToken.text?.toString()?.trim() ?: ""
            prefs.maxUid = binding.etMaxUid.text?.toString()?.trim() ?: ""
            prefs.secretKey = binding.etSecretKey.text?.toString()?.trim() ?: ""
            prefs.socksPort = port
            prefs.listenAll = binding.switchListenAll.isChecked
            prefs.debugLogging = binding.switchDebug.isChecked

            Toast.makeText(this, "Настройки сохранены", Toast.LENGTH_SHORT).show()
            finish()
        }
    }
}
