package com.openflux.client.ui

import android.os.Bundle
import android.view.View
import android.widget.Toast
import androidx.appcompat.app.AppCompatActivity
import com.openflux.client.R
import com.openflux.client.data.AppPreferences
import com.openflux.client.databinding.ActivitySettingsBinding

class SettingsActivity : AppCompatActivity() {

    private lateinit var binding: ActivitySettingsBinding
    private lateinit var prefs: AppPreferences

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
        if (prefs.transportType == "oneme") {
            binding.rbOneMe.isChecked = true
            binding.cardYandexConfig.visibility = View.GONE
            binding.cardMaxConfig.visibility = View.VISIBLE
        } else {
            binding.rbYandex.isChecked = true
            binding.cardYandexConfig.visibility = View.VISIBLE
            binding.cardMaxConfig.visibility = View.GONE
        }

        binding.etYandexUrl.setText(prefs.yandexDocUrl)
        binding.etMaxToken.setText(prefs.maxToken)
        binding.etMaxUid.setText(prefs.maxUid)
        binding.etSecretKey.setText(prefs.secretKey)
        binding.etPort.setText(prefs.socksPort.toString())
        binding.switchListenAll.isChecked = prefs.listenAll
        binding.switchDebug.isChecked = prefs.debugLogging
    }

    private fun setupListeners() {
        binding.rgTransport.setOnCheckedChangeListener { _, checkedId ->
            if (checkedId == R.id.rbOneMe) {
                binding.cardYandexConfig.visibility = View.GONE
                binding.cardMaxConfig.visibility = View.VISIBLE
            } else {
                binding.cardYandexConfig.visibility = View.VISIBLE
                binding.cardMaxConfig.visibility = View.GONE
            }
        }

        binding.btnSaveSettings.setOnClickListener {
            val portText = binding.etPort.text?.toString()?.trim() ?: "1080"
            val port = portText.toIntOrNull()
            if (port == null || port < 1 || port > 65535) {
                Toast.makeText(this, "Некорректный порт (допустимо 1-65535)", Toast.LENGTH_SHORT).show()
                return@setOnClickListener
            }

            val transport = if (binding.rbOneMe.isChecked) "oneme" else "yandex"
            val yandexUrl = binding.etYandexUrl.text?.toString()?.trim() ?: ""

            if (transport == "yandex" && yandexUrl.isEmpty()) {
                Toast.makeText(this, "Укажите URL документа Yandex Docs", Toast.LENGTH_SHORT).show()
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
