package com.openflux.client.ui

import android.Manifest
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.graphics.BitmapFactory
import android.net.Uri
import android.os.Build
import android.os.Bundle
import android.os.VibrationEffect
import android.os.Vibrator
import android.os.VibratorManager
import android.widget.Toast
import androidx.activity.result.contract.ActivityResultContracts
import androidx.appcompat.app.AppCompatActivity
import androidx.camera.core.Camera
import androidx.camera.core.CameraSelector
import androidx.camera.core.ImageAnalysis
import androidx.camera.core.Preview
import androidx.camera.core.resolutionselector.ResolutionSelector
import androidx.camera.core.resolutionselector.ResolutionStrategy
import androidx.camera.lifecycle.ProcessCameraProvider
import androidx.core.content.ContextCompat
import androidx.core.view.WindowCompat
import androidx.core.view.WindowInsetsCompat
import androidx.core.view.WindowInsetsControllerCompat
import androidx.lifecycle.lifecycleScope
import com.openflux.client.data.QRCodeDecoder
import com.openflux.client.databinding.ActivityScannerBinding
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import java.util.concurrent.ExecutorService
import java.util.concurrent.Executors
import java.util.concurrent.atomic.AtomicBoolean

class ScannerActivity : AppCompatActivity() {

    private lateinit var binding: ActivityScannerBinding
    private var cameraExecutor: ExecutorService? = null
    private var camera: Camera? = null
    private var isTorchOn = false
    private val isResultFound = AtomicBoolean(false)

    private val permissionLauncher = registerForActivityResult(
        ActivityResultContracts.RequestPermission()
    ) { isGranted ->
        if (isGranted) {
            startCamera()
        } else {
            Toast.makeText(this, "Требуется доступ к камере для сканирования QR-кода", Toast.LENGTH_LONG).show()
            finish()
        }
    }

    private val galleryLauncher = registerForActivityResult(
        ActivityResultContracts.GetContent()
    ) { uri: Uri? ->
        if (uri != null) {
            decodeGalleryUri(uri)
        }
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        binding = ActivityScannerBinding.inflate(layoutInflater)
        setContentView(binding.root)

        // Make fullscreen edge-to-edge with transparent system bars
        WindowCompat.setDecorFitsSystemWindows(window, false)
        window.statusBarColor = android.graphics.Color.TRANSPARENT
        window.navigationBarColor = android.graphics.Color.TRANSPARENT
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            window.isStatusBarContrastEnforced = false
            window.isNavigationBarContrastEnforced = false
        }
        val insetsController = WindowCompat.getInsetsController(window, window.decorView)
        insetsController.isAppearanceLightStatusBars = false
        insetsController.isAppearanceLightNavigationBars = false

        // Dynamically apply window insets to controls
        androidx.core.view.ViewCompat.setOnApplyWindowInsetsListener(binding.root) { _, windowInsets ->
            val statusBars = windowInsets.getInsets(
                WindowInsetsCompat.Type.statusBars() or WindowInsetsCompat.Type.displayCutout()
            )
            val navBars = windowInsets.getInsets(WindowInsetsCompat.Type.navigationBars())
            val systemBars = windowInsets.getInsets(WindowInsetsCompat.Type.systemBars())

            binding.topBar.setPadding(
                (20 * resources.displayMetrics.density).toInt() + systemBars.left,
                statusBars.top + (12 * resources.displayMetrics.density).toInt(),
                (20 * resources.displayMetrics.density).toInt() + systemBars.right,
                0
            )

            val galleryParams = binding.btnGallery.layoutParams as? android.view.ViewGroup.MarginLayoutParams
            galleryParams?.let {
                it.leftMargin = (24 * resources.displayMetrics.density).toInt() + systemBars.left
                it.bottomMargin = navBars.bottom + (24 * resources.displayMetrics.density).toInt()
                binding.btnGallery.layoutParams = it
            }

            windowInsets
        }

        cameraExecutor = Executors.newSingleThreadExecutor()

        setupListeners()

        if (ContextCompat.checkSelfPermission(this, Manifest.permission.CAMERA) == PackageManager.PERMISSION_GRANTED) {
            startCamera()
        } else {
            permissionLauncher.launch(Manifest.permission.CAMERA)
        }
    }

    private fun setupListeners() {
        binding.btnClose.setOnClickListener {
            finish()
        }

        binding.btnTorch.setOnClickListener {
            toggleTorch()
        }

        binding.btnGallery.setOnClickListener {
            galleryLauncher.launch("image/*")
        }
    }

    private fun toggleTorch() {
        val cam = camera ?: return
        if (cam.cameraInfo.hasFlashUnit()) {
            isTorchOn = !isTorchOn
            cam.cameraControl.enableTorch(isTorchOn)
            binding.btnTorch.setColorFilter(
                if (isTorchOn) 0xFFFFD700.toInt() else 0xFFFFFFFF.toInt()
            )
        } else {
            Toast.makeText(this, "Вспышка недоступна на этом устройстве", Toast.LENGTH_SHORT).show()
        }
    }

    private fun startCamera() {
        val cameraProviderFuture = ProcessCameraProvider.getInstance(this)

        cameraProviderFuture.addListener({
            val cameraProvider: ProcessCameraProvider = cameraProviderFuture.get()

            val preview = Preview.Builder().build().also {
                it.surfaceProvider = binding.previewView.surfaceProvider
            }

            val resolutionSelector = ResolutionSelector.Builder()
                .setResolutionStrategy(
                    ResolutionStrategy(
                        android.util.Size(1280, 720),
                        ResolutionStrategy.FALLBACK_RULE_CLOSEST_HIGHER_THEN_LOWER
                    )
                )
                .build()

            val imageAnalyzer = ImageAnalysis.Builder()
                .setResolutionSelector(resolutionSelector)
                .setBackpressureStrategy(ImageAnalysis.STRATEGY_KEEP_ONLY_LATEST)
                .build()
                .also { analysis ->
                    cameraExecutor?.let { executor ->
                        analysis.setAnalyzer(executor) { imageProxy ->
                            if (!isResultFound.get()) {
                                val text = QRCodeDecoder.decodeImageProxy(imageProxy)
                                if (!text.isNullOrEmpty() && isResultFound.compareAndSet(false, true)) {
                                    runOnUiThread {
                                        onQrDetected(text)
                                    }
                                }
                            }
                            imageProxy.close()
                        }
                    }
                }

            val cameraSelector = CameraSelector.DEFAULT_BACK_CAMERA

            try {
                cameraProvider.unbindAll()
                camera = cameraProvider.bindToLifecycle(
                    this,
                    cameraSelector,
                    preview,
                    imageAnalyzer
                )
            } catch (exc: Exception) {
                Toast.makeText(this, "Не удалось запустить камеру: ${exc.message}", Toast.LENGTH_SHORT).show()
            }
        }, ContextCompat.getMainExecutor(this))
    }

    private fun decodeGalleryUri(uri: Uri) {
        lifecycleScope.launch(Dispatchers.IO) {
            try {
                contentResolver.openInputStream(uri)?.use { stream ->
                    val bitmap = BitmapFactory.decodeStream(stream)
                    if (bitmap != null) {
                        val text = QRCodeDecoder.decodeBitmap(bitmap)
                        withContext(Dispatchers.Main) {
                            if (!text.isNullOrEmpty() && isResultFound.compareAndSet(false, true)) {
                                onQrDetected(text)
                            } else {
                                Toast.makeText(
                                    this@ScannerActivity,
                                    "QR-код на изображении не распознан",
                                    Toast.LENGTH_LONG
                                ).show()
                            }
                        }
                    }
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    Toast.makeText(this@ScannerActivity, "Ошибка чтения изображения: ${e.message}", Toast.LENGTH_SHORT).show()
                }
            }
        }
    }

    private fun onQrDetected(rawContent: String) {
        triggerHaptic()
        val resultIntent = Intent().apply {
            putExtra(EXTRA_SCAN_RESULT, rawContent)
        }
        setResult(RESULT_OK, resultIntent)
        finish()
    }

    private fun triggerHaptic() {
        try {
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
                val vibratorManager = getSystemService(Context.VIBRATOR_MANAGER_SERVICE) as? VibratorManager
                vibratorManager?.defaultVibrator?.vibrate(
                    VibrationEffect.createOneShot(50, VibrationEffect.DEFAULT_AMPLITUDE)
                )
            } else {
                @Suppress("DEPRECATION")
                val vibrator = getSystemService(Context.VIBRATOR_SERVICE) as? Vibrator
                vibrator?.vibrate(50)
            }
        } catch (_: Exception) {}
    }

    override fun onDestroy() {
        super.onDestroy()
        cameraExecutor?.shutdown()
    }

    companion object {
        const val EXTRA_SCAN_RESULT = "extra_scan_result"
    }
}
