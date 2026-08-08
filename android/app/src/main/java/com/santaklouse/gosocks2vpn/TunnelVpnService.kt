package com.santaklouse.gosocks2vpn

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Intent
import android.content.pm.ServiceInfo
import android.net.Uri
import android.net.VpnService
import android.os.Build
import mobile.Mobile
import java.net.InetSocketAddress
import java.util.concurrent.Executors
import java.util.concurrent.atomic.AtomicBoolean

class TunnelVpnService : VpnService() {
    private val executor = Executors.newSingleThreadExecutor()
    private val running = AtomicBoolean(false)

    override fun onCreate() {
        super.onCreate()
        createNotificationChannel()
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        when (intent?.action) {
            ACTION_START -> {
                showForegroundNotification("Connecting…")
                val request = StartRequest.from(intent)
                executor.execute { startTunnel(request) }
            }
            ACTION_STOP -> executor.execute { stopTunnel("Disconnected") }
        }
        return Service.START_NOT_STICKY
    }

    override fun onRevoke() {
        executor.execute { stopTunnel("VPN permission was revoked") }
        super.onRevoke()
    }

    override fun onDestroy() {
        if (running.get()) {
            runCatching { Mobile.stop() }
            running.set(false)
        }
        executor.shutdownNow()
        super.onDestroy()
    }

    private fun startTunnel(request: StartRequest) {
        try {
            if (running.get()) Mobile.stop()
            running.set(false)

            val proxyUrl = request.proxyUrl()
            preflight(request.host, request.port)
            val descriptor = Builder()
                .setSession("go-socks2vpn")
                .setMtu(MTU)
                .addAddress("198.18.0.1", 32)
                .addRoute("0.0.0.0", 0)
                .addAddress("fd00:198:18::1", 128)
                .addRoute("::", 0)
                .addDnsServer("1.1.1.1")
                .addDisallowedApplication(packageName)
                .setBlocking(false)
                .establish() ?: error("Android did not create the VPN interface")

            // Ownership is transferred to the Go engine. Engine.Stop closes this fd.
			val detachedFd = descriptor.detachFd()
            Mobile.start(detachedFd.toLong(), proxyUrl, MTU.toLong())
            running.set(true)
            showForegroundNotification("Connected to ${request.host}:${request.port}")
            broadcastStatus(true, "Connected")
        } catch (error: Throwable) {
            running.set(false)
            broadcastStatus(false, "Error: ${error.message ?: error.javaClass.simpleName}")
            stopForeground(STOP_FOREGROUND_REMOVE)
            stopSelf()
        }
    }

    private fun stopTunnel(message: String) {
        if (running.getAndSet(false)) runCatching { Mobile.stop() }
        broadcastStatus(false, message)
        stopForeground(STOP_FOREGROUND_REMOVE)
        stopSelf()
    }

    private fun preflight(host: String, port: Int) {
        java.net.Socket().use { socket ->
            socket.connect(InetSocketAddress(host, port), 4_000)
        }
    }

    private fun showForegroundNotification(message: String) {
        val openApp = PendingIntent.getActivity(
            this,
            0,
            Intent(this, MainActivity::class.java),
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
        val builder = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
			Notification.Builder(this, CHANNEL_ID)
		} else {
			@Suppress("DEPRECATION")
			Notification.Builder(this)
		}
		val notification = builder
            .setSmallIcon(com.santaklouse.gosocks2vpn.R.drawable.ic_launcher)
            .setContentTitle("go-socks2vpn")
            .setContentText(message)
            .setContentIntent(openApp)
            .setOngoing(true)
            .build()
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.UPSIDE_DOWN_CAKE) {
            startForeground(NOTIFICATION_ID, notification, ServiceInfo.FOREGROUND_SERVICE_TYPE_SPECIAL_USE)
        } else {
            startForeground(NOTIFICATION_ID, notification)
        }
    }

    private fun createNotificationChannel() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(CHANNEL_ID, "VPN connection", NotificationManager.IMPORTANCE_LOW)
            getSystemService(NotificationManager::class.java).createNotificationChannel(channel)
        }
    }

    private fun broadcastStatus(connected: Boolean, message: String) {
        sendBroadcast(Intent(ACTION_STATUS).apply {
            setPackage(packageName)
            putExtra(EXTRA_CONNECTED, connected)
            putExtra(EXTRA_MESSAGE, message)
        })
    }

    private data class StartRequest(
        val scheme: String,
        val host: String,
        val port: Int,
        val username: String,
        val password: String,
    ) {
        fun proxyUrl(): String {
            val hostPart = if (host.contains(':')) "[$host]" else host
            val authentication = if (username.isEmpty()) {
                ""
            } else if (scheme == "socks4") {
                "${Uri.encode(username)}@"
            } else {
                "${Uri.encode(username)}:${Uri.encode(password)}@"
            }
            return "$scheme://$authentication$hostPart:$port"
        }

        companion object {
            fun from(intent: Intent) = StartRequest(
                scheme = intent.getStringExtra(EXTRA_SCHEME).orEmpty().ifEmpty { "socks5" },
                host = intent.getStringExtra(EXTRA_HOST).orEmpty(),
                port = intent.getIntExtra(EXTRA_PORT, 0),
                username = intent.getStringExtra(EXTRA_USERNAME).orEmpty(),
                password = intent.getStringExtra(EXTRA_PASSWORD).orEmpty(),
            )
        }
    }

    companion object {
        const val ACTION_START = "com.santaklouse.gosocks2vpn.START"
        const val ACTION_STOP = "com.santaklouse.gosocks2vpn.STOP"
        const val ACTION_STATUS = "com.santaklouse.gosocks2vpn.STATUS"
        const val EXTRA_HOST = "host"
        const val EXTRA_SCHEME = "scheme"
        const val EXTRA_PORT = "port"
        const val EXTRA_USERNAME = "username"
        const val EXTRA_PASSWORD = "password"
        const val EXTRA_CONNECTED = "connected"
        const val EXTRA_MESSAGE = "message"
        private const val CHANNEL_ID = "vpn"
        private const val NOTIFICATION_ID = 1001
        private const val MTU = 1500
    }
}
