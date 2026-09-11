package com.openflux.client.ui

import android.content.pm.ApplicationInfo
import android.content.pm.PackageManager
import android.os.Bundle
import android.view.View
import android.widget.Toast
import androidx.appcompat.app.AppCompatActivity
import androidx.core.widget.doAfterTextChanged
import androidx.lifecycle.lifecycleScope
import androidx.recyclerview.widget.LinearLayoutManager
import com.openflux.client.R
import com.openflux.client.data.AppPreferences
import com.openflux.client.databinding.ActivitySplitTunnelBinding
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

class SplitTunnelActivity : AppCompatActivity() {

    private lateinit var binding: ActivitySplitTunnelBinding
    private lateinit var prefs: AppPreferences
    private lateinit var adapter: AppAdapter

    private var allApps: MutableList<AppItem> = mutableListOf()
    private val selectedPackages = mutableSetOf<String>()

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        binding = ActivitySplitTunnelBinding.inflate(layoutInflater)
        setContentView(binding.root)
        binding.root.applySystemWindowInsetsPadding()

        prefs = AppPreferences(this)
        selectedPackages.addAll(prefs.selectedPackages)

        setupMode()
        setupRecyclerView()
        setupListeners()
        loadInstalledApps()
    }

    private fun setupMode() {
        if (prefs.splitTunnelMode == AppPreferences.MODE_WHITELIST) {
            binding.rbWhitelist.isChecked = true
        } else {
            binding.rbBlacklist.isChecked = true
        }

        binding.rgSplitMode.setOnCheckedChangeListener { _, checkedId ->
            prefs.splitTunnelMode = if (checkedId == R.id.rbWhitelist) {
                AppPreferences.MODE_WHITELIST
            } else {
                AppPreferences.MODE_BLACKLIST
            }
        }
    }

    private fun setupRecyclerView() {
        adapter = AppAdapter(emptyList()) { app, isSelected ->
            if (isSelected) {
                selectedPackages.add(app.packageName)
            } else {
                selectedPackages.remove(app.packageName)
            }
        }
        binding.rvApps.layoutManager = LinearLayoutManager(this)
        binding.rvApps.adapter = adapter
    }

    private fun setupListeners() {
        binding.cbShowSystem.isChecked = prefs.showSystemApps
        binding.cbShowSystem.setOnCheckedChangeListener { _, isChecked ->
            prefs.showSystemApps = isChecked
            applyFilter()
        }

        binding.etSearchApp.doAfterTextChanged {
            applyFilter()
        }

        binding.btnSelectAll.setOnClickListener {
            val currentFiltered = getFilteredApps()
            for (app in currentFiltered) {
                app.isSelected = true
                selectedPackages.add(app.packageName)
            }
            applyFilter()
        }

        binding.btnDeselectAll.setOnClickListener {
            val currentFiltered = getFilteredApps()
            for (app in currentFiltered) {
                app.isSelected = false
                selectedPackages.remove(app.packageName)
            }
            applyFilter()
        }

        binding.btnSaveSplit.setOnClickListener {
            prefs.selectedPackages = selectedPackages
            Toast.makeText(this, "Настройки раздельного туннелирования сохранены", Toast.LENGTH_SHORT).show()
            finish()
        }
    }

    private fun loadInstalledApps() {
        binding.progressBar.visibility = View.VISIBLE

        lifecycleScope.launch {
            val loaded = withContext(Dispatchers.IO) {
                val pm = packageManager
                val packages = pm.getInstalledPackages(PackageManager.GET_META_DATA)
                val list = mutableListOf<AppItem>()

                for (pkg in packages) {
                    val appInfo = pkg.applicationInfo ?: continue
                    if (pkg.packageName == packageName) continue // Skip our own app

                    val isSystem = (appInfo.flags and ApplicationInfo.FLAG_SYSTEM) != 0
                    val name = appInfo.loadLabel(pm).toString()
                    val icon = appInfo.loadIcon(pm)
                    val isSelected = selectedPackages.contains(pkg.packageName)

                    list.add(AppItem(name, pkg.packageName, icon, isSystem, isSelected))
                }

                list.sortedWith(
                    compareByDescending<AppItem> { it.isSelected }
                        .thenBy { it.name.lowercase() }
                )
            }

            allApps.clear()
            allApps.addAll(loaded)
            binding.progressBar.visibility = View.GONE
            applyFilter()
        }
    }

    private fun getFilteredApps(): List<AppItem> {
        val query = binding.etSearchApp.text?.toString()?.trim()?.lowercase() ?: ""
        val showSystem = binding.cbShowSystem.isChecked

        return allApps.asSequence()
            .filter { app ->
                val matchQuery = query.isEmpty() ||
                        app.name.lowercase().contains(query) ||
                        app.packageName.lowercase().contains(query)
                val matchSystem = showSystem || !app.isSystem
                matchQuery && matchSystem
            }
            .sortedWith(
                compareByDescending<AppItem> { it.isSelected }
                    .thenBy { it.name.lowercase() }
            )
            .toList()
    }

    private fun applyFilter() {
        adapter.updateData(getFilteredApps())
    }
}
