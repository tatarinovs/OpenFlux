package com.openflux.client.ui

import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.os.Bundle
import android.widget.Toast
import androidx.appcompat.app.AppCompatActivity
import androidx.lifecycle.lifecycleScope
import com.openflux.client.R
import com.openflux.client.core.OpenFluxCore
import com.openflux.client.databinding.ActivityLogBinding
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch

class LogActivity : AppCompatActivity() {

    private lateinit var binding: ActivityLogBinding

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        binding = ActivityLogBinding.inflate(layoutInflater)
        setContentView(binding.root)
        binding.root.applySystemWindowInsetsPadding()

        setupListeners()
        OpenFluxCore.syncTimezone()
        startLogPolling()
    }

    private fun setupListeners() {
        binding.btnCopyLogs.setOnClickListener {
            val logs = binding.tvLogs.text.toString()
            val clipboard = getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
            val clip = ClipData.newPlainText("OpenFlux Logs", logs)
            clipboard.setPrimaryClip(clip)
            Toast.makeText(this, R.string.logs_copied, Toast.LENGTH_SHORT).show()
        }

        binding.btnClearLogs.setOnClickListener {
            binding.tvLogs.text = ""
            try {
                OpenFluxCore.clearLogs()
            } catch (_: Throwable) {}
            try {
                java.io.File(filesDir, "crash.log").delete()
            } catch (_: Exception) {}
        }
    }

    private fun startLogPolling() {
        lifecycleScope.launch {
            while (isActive) {
                val crashFile = java.io.File(filesDir, "crash.log")
                val crashText = if (crashFile.exists()) {
                    "=== КРАШ ПРИЛОЖЕНИЯ ===\n${crashFile.readText()}\n========================\n\n"
                } else ""
                val logs = OpenFluxCore.getRecentLogs()
                val fullLogs = crashText + logs
                if (fullLogs.isNotEmpty() && fullLogs != binding.tvLogs.text.toString()) {
                    binding.tvLogs.text = fullLogs
                    binding.scrollLogs.post {
                        binding.scrollLogs.fullScroll(android.view.View.FOCUS_DOWN)
                    }
                }
                delay(1000)
            }
        }
    }
}
