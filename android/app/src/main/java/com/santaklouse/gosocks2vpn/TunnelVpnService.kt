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
import android.os.SystemClock
import android.util.Log
import mobile.Mobile
import java.util.concurrent.Executors
import java.util.concurrent.ScheduledFuture
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicBoolean

class TunnelVpnService : VpnService() {
    private val executor = Executors.newSingleThreadScheduledExecutor()
    private val running = AtomicBoolean(false)
    private var statisticsTask: ScheduledFuture<*>? = null
	private var lastEngineWarning = ""

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
			ACTION_QUERY -> executor.execute {
				if (running.get()) {
					broadcastStatus(true, "Connected")
					broadcastStatistics(Mobile.downloadedBytes(), Mobile.uploadedBytes(), 0, 0)
				} else {
					broadcastStatus(false, "Disconnected")
					broadcastStatistics(0, 0, 0, 0)
					stopSelfResult(startId)
				}
			}
        }
        return Service.START_NOT_STICKY
    }

    override fun onRevoke() {
        executor.execute { stopTunnel("VPN permission was revoked") }
        super.onRevoke()
    }

    override fun onDestroy() {
        stopStatistics()
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
            stopStatistics()
            running.set(false)
			broadcastStatistics(0, 0, 0, 0)

            val proxyUrl = request.proxyUrl()
			Mobile.checkProxy(proxyUrl)
            val descriptor = Builder()
                .setSession("go-socks2vpn")
                .setMtu(MTU)
                .addAddress("198.18.0.1", 32)
                .addRoute("0.0.0.0", 0)
                .addDnsServer("1.1.1.1")
                .addDisallowedApplication(packageName)
                .setBlocking(false)
                .establish() ?: error("Android did not create the VPN interface")

            // Ownership is transferred to the Go engine. Engine.Stop closes this fd.
			val detachedFd = descriptor.detachFd()
            Mobile.start(detachedFd.toLong(), proxyUrl, MTU.toLong())
            running.set(true)
            startStatistics()
            showForegroundNotification("Connected to ${request.host}:${request.port}")
            broadcastStatus(true, "Connected")
        } catch (error: Throwable) {
            stopStatistics()
            running.set(false)
            broadcastStatus(false, "Error: ${error.message ?: error.javaClass.simpleName}")
            stopForeground(STOP_FOREGROUND_REMOVE)
            stopSelf()
        }
    }

    private fun stopTunnel(message: String) {
		val wasRunning = running.getAndSet(false)
		if (wasRunning) {
			val uploaded = runCatching { Mobile.uploadedBytes() }.getOrDefault(0)
			val downloaded = runCatching { Mobile.downloadedBytes() }.getOrDefault(0)
			stopStatistics()
			broadcastStatistics(downloaded, uploaded, 0, 0)
			runCatching { Mobile.stop() }
		} else {
			stopStatistics()
		}
        broadcastStatus(false, message)
        stopForeground(STOP_FOREGROUND_REMOVE)
        stopSelf()
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

	private fun startStatistics() {
		stopStatistics()
		lastEngineWarning = ""
		var previousDownloaded = Mobile.downloadedBytes()
		var previousUploaded = Mobile.uploadedBytes()
		var previousAt = SystemClock.elapsedRealtimeNanos()
		broadcastStatistics(previousDownloaded, previousUploaded, 0, 0)
		statisticsTask = executor.scheduleAtFixedRate({
			if (!running.get()) return@scheduleAtFixedRate
			val now = SystemClock.elapsedRealtimeNanos()
			val downloaded = Mobile.downloadedBytes()
			val uploaded = Mobile.uploadedBytes()
			val engineWarning = Mobile.lastWarning()
			if (engineWarning.isNotEmpty() && engineWarning != lastEngineWarning) {
				Log.w(LOG_TAG, engineWarning)
				lastEngineWarning = engineWarning
			}
			val elapsedSeconds = (now - previousAt).coerceAtLeast(1).toDouble() / 1_000_000_000.0
			val downloadRate = ((downloaded - previousDownloaded).coerceAtLeast(0) / elapsedSeconds).toLong()
			val uploadRate = ((uploaded - previousUploaded).coerceAtLeast(0) / elapsedSeconds).toLong()
			broadcastStatistics(downloaded, uploaded, downloadRate, uploadRate)
			previousDownloaded = downloaded
			previousUploaded = uploaded
			previousAt = now
		}, 1, 1, TimeUnit.SECONDS)
	}

	private fun stopStatistics() {
		statisticsTask?.cancel(false)
		statisticsTask = null
	}

	private fun broadcastStatistics(downloaded: Long, uploaded: Long, downloadRate: Long, uploadRate: Long) {
		sendBroadcast(Intent(ACTION_STATISTICS).apply {
			setPackage(packageName)
			putExtra(EXTRA_DOWNLOADED_BYTES, downloaded)
			putExtra(EXTRA_UPLOADED_BYTES, uploaded)
			putExtra(EXTRA_DOWNLOAD_RATE, downloadRate)
			putExtra(EXTRA_UPLOAD_RATE, uploadRate)
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
		const val ACTION_QUERY = "com.santaklouse.gosocks2vpn.QUERY"
        const val ACTION_STATUS = "com.santaklouse.gosocks2vpn.STATUS"
		const val ACTION_STATISTICS = "com.santaklouse.gosocks2vpn.STATISTICS"
        const val EXTRA_HOST = "host"
        const val EXTRA_SCHEME = "scheme"
        const val EXTRA_PORT = "port"
        const val EXTRA_USERNAME = "username"
        const val EXTRA_PASSWORD = "password"
        const val EXTRA_CONNECTED = "connected"
        const val EXTRA_MESSAGE = "message"
		const val EXTRA_DOWNLOADED_BYTES = "downloadedBytes"
		const val EXTRA_UPLOADED_BYTES = "uploadedBytes"
		const val EXTRA_DOWNLOAD_RATE = "downloadRate"
		const val EXTRA_UPLOAD_RATE = "uploadRate"
        private const val CHANNEL_ID = "vpn"
        private const val NOTIFICATION_ID = 1001
        private const val MTU = 1500
		private const val LOG_TAG = "go-socks2vpn"
    }
}
