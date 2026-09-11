package com.openflux.client.data

import android.content.Context
import android.content.SharedPreferences
import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import android.util.Base64
import java.nio.ByteBuffer
import java.nio.charset.StandardCharsets
import java.security.KeyStore
import javax.crypto.Cipher
import javax.crypto.KeyGenerator
import javax.crypto.SecretKey
import javax.crypto.spec.GCMParameterSpec

/**
 * Хранилище конфиденциальных данных (URL документов, приватные ключи, токены),
 * зашифрованное аппаратным ключом Android KeyStore (AES-256-GCM).
 *
 * Защищает секреты от извлечения даже при наличии root-прав или снятии ADB backup.
 */
class SecureSettings(context: Context) {

    private val prefs: SharedPreferences =
        context.getSharedPreferences(STORE_NAME, Context.MODE_PRIVATE)

    fun getString(name: String, fallback: String): String {
        val encoded = prefs.getString(name, null) ?: return fallback
        return try {
            val packed = Base64.decode(encoded, Base64.NO_WRAP)
            val buffer = ByteBuffer.wrap(packed)
            val ivLength = buffer.get().toInt() and 0xff
            if (ivLength < 12 || ivLength > 16 || buffer.remaining() <= ivLength) {
                return fallback
            }
            val iv = ByteArray(ivLength)
            buffer.get(iv)
            val ciphertext = ByteArray(buffer.remaining())
            buffer.get(ciphertext)

            val cipher = Cipher.getInstance(TRANSFORMATION)
            cipher.init(Cipher.DECRYPT_MODE, getOrCreateKey(), GCMParameterSpec(128, iv))
            cipher.updateAAD(name.toByteArray(StandardCharsets.UTF_8))
            val plaintext = cipher.doFinal(ciphertext)
            String(plaintext, StandardCharsets.UTF_8)
        } catch (_: Exception) {
            fallback
        }
    }

    fun putString(name: String, value: String): Boolean {
        return try {
            val cipher = Cipher.getInstance(TRANSFORMATION)
            cipher.init(Cipher.ENCRYPT_MODE, getOrCreateKey())
            cipher.updateAAD(name.toByteArray(StandardCharsets.UTF_8))
            val ciphertext = cipher.doFinal(value.toByteArray(StandardCharsets.UTF_8))
            val iv = cipher.iv

            val packed = ByteBuffer.allocate(1 + iv.size + ciphertext.size)
            packed.put(iv.size.toByte())
            packed.put(iv)
            packed.put(ciphertext)

            prefs.edit()
                .putString(name, Base64.encodeToString(packed.array(), Base64.NO_WRAP))
                .commit()
        } catch (_: Exception) {
            false
        }
    }

    private fun getOrCreateKey(): SecretKey {
        val keyStore = KeyStore.getInstance(ANDROID_KEY_STORE)
        keyStore.load(null)

        if (keyStore.containsAlias(KEY_ALIAS)) {
            val entry = keyStore.getEntry(KEY_ALIAS, null) as? KeyStore.SecretKeyEntry
            if (entry != null) {
                return entry.secretKey
            }
        }

        val generator = KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES, ANDROID_KEY_STORE)
        val spec = KeyGenParameterSpec.Builder(
            KEY_ALIAS,
            KeyProperties.PURPOSE_ENCRYPT or KeyProperties.PURPOSE_DECRYPT
        )
            .setBlockModes(KeyProperties.BLOCK_MODE_GCM)
            .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
            .setKeySize(256)
            .build()

        generator.init(spec)
        return generator.generateKey()
    }

    companion object {
        private const val STORE_NAME = "openflux_secure_prefs"
        private const val KEY_ALIAS = "com.openflux.client.settings.v1"
        private const val ANDROID_KEY_STORE = "AndroidKeyStore"
        private const val TRANSFORMATION = "AES/GCM/NoPadding"
    }
}
