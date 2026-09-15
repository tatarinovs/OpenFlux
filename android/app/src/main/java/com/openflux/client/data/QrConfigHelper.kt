package com.openflux.client.data

import android.graphics.Bitmap
import android.graphics.Color
import com.google.zxing.BarcodeFormat
import com.google.zxing.EncodeHintType
import com.google.zxing.qrcode.QRCodeWriter
import org.json.JSONObject
import java.util.EnumMap

data class QrConfigData(
    val transport: String,
    val target: String,
    val secretKey: String? = null,
    val socksPort: Int? = null,
    val maxToken: String? = null,
    val maxUid: String? = null
)

object QrConfigHelper {

    fun buildJson(
        transport: String,
        target: String,
        secretKey: String,
        socksPort: Int
    ): String {
        val json = JSONObject()
        json.put("app", "openflux")
        json.put("version", 1)
        json.put("transport", transport)
        json.put("target", target)
        if (secretKey.isNotEmpty()) {
            json.put("secret_key", secretKey)
        }
        json.put("socks_port", socksPort)
        return json.toString(2)
    }

    fun parse(raw: String): QrConfigData? {
        val trimmed = raw.trim()
        return try {
            val json = JSONObject(trimmed)

            // 1. Standard OpenFlux schema
            if (json.has("transport") || json.has("target")) {
                val transport = json.optString("transport", "cupsonline")
                val target = json.optString("target", "")
                val secretKey = if (json.has("secret_key")) json.optString("secret_key") else null
                val port = if (json.has("socks_port")) json.optInt("socks_port", 1080) else null
                val maxToken = if (json.has("max_token")) json.optString("max_token") else null
                val maxUid = if (json.has("max_uid")) json.optString("max_uid") else null
                return QrConfigData(
                    transport = transport,
                    target = target,
                    secretKey = secretKey,
                    socksPort = port,
                    maxToken = maxToken,
                    maxUid = maxUid
                )
            }

            // 2. Upstream OpenFlux Tunnel schema
            if (json.has("transportType") && json.has("transportConnPayload")) {
                val tType = json.getString("transportType").lowercase()
                val payloadArr = json.getJSONArray("transportConnPayload")
                val payload = mutableListOf<String>()
                for (i in 0 until payloadArr.length()) {
                    payload.add(payloadArr.getString(i))
                }

                fun getArg(name: String): String {
                    val idx = payload.indexOf(name)
                    return if (idx >= 0 && idx + 1 < payload.size) payload[idx + 1] else ""
                }

                if (tType == "yandex" || tType == "vyandex" || tType == "mailru") {
                    val url = getArg("--url")
                    return QrConfigData(
                        transport = tType,
                        target = url
                    )
                } else if (tType == "max") {
                    val token = getArg("--maxToken")
                    val uid = getArg("--maxUid")
                    return QrConfigData(
                        transport = "oneme",
                        target = "",
                        maxToken = token,
                        maxUid = uid
                    )
                }
            }

            null
        } catch (e: Exception) {
            null
        }
    }

    fun generateQrBitmap(content: String, sizePx: Int = 512): Bitmap {
        val hints = EnumMap<EncodeHintType, Any>(EncodeHintType::class.java)
        hints[EncodeHintType.CHARACTER_SET] = "UTF-8"
        hints[EncodeHintType.MARGIN] = 1

        val writer = QRCodeWriter()
        val bitMatrix = writer.encode(content, BarcodeFormat.QR_CODE, sizePx, sizePx, hints)
        val width = bitMatrix.width
        val height = bitMatrix.height
        val bmp = Bitmap.createBitmap(width, height, Bitmap.Config.RGB_565)

        for (x in 0 until width) {
            for (y in 0 until height) {
                bmp.setPixel(x, y, if (bitMatrix.get(x, y)) Color.BLACK else Color.WHITE)
            }
        }
        return bmp
    }
}
