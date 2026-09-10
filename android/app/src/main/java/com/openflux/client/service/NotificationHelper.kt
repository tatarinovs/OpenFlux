package com.openflux.client.service

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.os.Build
import androidx.core.app.NotificationCompat
import com.openflux.client.R
import com.openflux.client.ui.MainActivity

object NotificationHelper {

    const val CHANNEL_ID = "openflux_channel"
    const val NOTIFICATION_ID = 1001

    fun createNotificationChannel(context: Context) {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val name = context.getString(R.string.notif_channel_name)
            val channel = NotificationChannel(
                CHANNEL_ID,
                name,
                NotificationManager.IMPORTANCE_LOW
            ).apply {
                description = "OpenFlux Background Service"
                setShowBadge(false)
            }
            val manager = context.getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
            manager.createNotificationChannel(channel)
        }
    }

    fun buildNotification(
        context: Context,
        isVpn: Boolean,
        port: Int,
        stopIntent: Intent,
        stats: String? = null
    ): Notification {
        createNotificationChannel(context)

        val mainIntent = Intent(context, MainActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_SINGLE_TOP
        }
        val mainPendingIntent = PendingIntent.getActivity(
            context,
            0,
            mainIntent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )

        val stopPendingIntent = PendingIntent.getService(
            context,
            1,
            stopIntent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )

        val title = if (isVpn) "OpenFlux: VPN Туннель" else "OpenFlux: SOCKS5 Прокси"
        val statusText = if (isVpn) {
            context.getString(R.string.notif_vpn_running, port)
        } else {
            context.getString(R.string.notif_proxy_running, port)
        }

        val content = if (!stats.isNullOrBlank()) {
            "$statusText\n$stats"
        } else {
            statusText
        }

        val builder = NotificationCompat.Builder(context, CHANNEL_ID)
            .setSmallIcon(R.drawable.ic_vpn)
            .setContentTitle(title)
            .setContentText(if (!stats.isNullOrBlank()) stats else statusText)
            .setStyle(NotificationCompat.BigTextStyle().bigText(content))
            .setContentIntent(mainPendingIntent)
            .addAction(0, context.getString(R.string.btn_disconnect), stopPendingIntent)
            .setOngoing(true)
            .setOnlyAlertOnce(true)
            .setPriority(NotificationCompat.PRIORITY_LOW)

        if (!stats.isNullOrBlank()) {
            val sub = if (stats.contains("/s")) {
                val parts = stats.split("  ")
                if (parts.size == 2) {
                    val up = parts[0].substringBefore(" (")
                    val down = parts[1].substringBefore(" (")
                    "$up  $down"
                } else stats
            } else stats
            builder.setSubText(sub)
        }

        return builder.build()
    }

    fun updateNotification(
        context: Context,
        isVpn: Boolean,
        port: Int,
        stats: String?,
        stopIntent: Intent
    ) {
        val notification = buildNotification(context, isVpn, port, stopIntent, stats)
        val manager = context.getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
        manager.notify(NOTIFICATION_ID, notification)
    }
}
