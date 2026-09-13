package com.openflux.client.data

import android.graphics.Bitmap
import androidx.camera.core.ImageProxy
import com.google.zxing.BarcodeFormat
import com.google.zxing.BinaryBitmap
import com.google.zxing.DecodeHintType
import com.google.zxing.NotFoundException
import com.google.zxing.PlanarYUVLuminanceSource
import com.google.zxing.RGBLuminanceSource
import com.google.zxing.common.GlobalHistogramBinarizer
import com.google.zxing.common.HybridBinarizer
import com.google.zxing.qrcode.QRCodeReader
import java.util.EnumMap

object QRCodeDecoder {

    private val reader = QRCodeReader()

    private val hints = EnumMap<DecodeHintType, Any>(DecodeHintType::class.java).apply {
        put(DecodeHintType.POSSIBLE_FORMATS, listOf(BarcodeFormat.QR_CODE))
        put(DecodeHintType.TRY_HARDER, java.lang.Boolean.TRUE)
        put(DecodeHintType.CHARACTER_SET, "UTF-8")
    }

    /**
     * Decodes a QR code directly from a CameraX ImageProxy (YUV_420_888).
     * Extracts the luminance channel directly without RGB conversion.
     */
    fun decodeImageProxy(imageProxy: ImageProxy): String? {
        return try {
            val yPlane = imageProxy.planes[0]
            val buffer = yPlane.buffer.duplicate()
            val bytes = ByteArray(buffer.remaining())
            buffer.get(bytes)

            val source = PlanarYUVLuminanceSource(
                bytes,
                yPlane.rowStride,
                imageProxy.height,
                0,
                0,
                imageProxy.width,
                imageProxy.height,
                false
            )
            val binaryBitmap = BinaryBitmap(HybridBinarizer(source))
            reader.decode(binaryBitmap, hints)?.text
        } catch (_: Exception) {
            null
        }
    }

    /**
     * Decodes a QR code from a static Bitmap (e.g. chosen from gallery).
     */
    fun decodeBitmap(bitmap: Bitmap): String? {
        return try {
            val width = bitmap.width
            val height = bitmap.height
            val pixels = IntArray(width * height)
            bitmap.getPixels(pixels, 0, width, 0, 0, width, height)

            val source = RGBLuminanceSource(width, height, pixels)
            try {
                reader.decode(BinaryBitmap(HybridBinarizer(source)), hints)?.text
            } catch (_: NotFoundException) {
                // Fallback to global histogram binarizer
                reader.decode(BinaryBitmap(GlobalHistogramBinarizer(source)), hints)?.text
            }
        } catch (_: Exception) {
            null
        }
    }
}
