package com.santaklouse.gosocks2vpn

import android.Manifest
import android.app.Activity
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.content.pm.PackageManager
import android.graphics.Color
import android.graphics.Typeface
import android.net.VpnService
import android.os.Build
import android.os.Bundle
import android.text.InputType
import android.view.Gravity
import android.view.ViewGroup
import android.widget.ArrayAdapter
import android.widget.Button
import android.widget.EditText
import android.widget.LinearLayout
import android.widget.ProgressBar
import android.widget.ScrollView
import android.widget.Spinner
import android.widget.TextView
import android.widget.Toast

class MainActivity : Activity() {
    private lateinit var protocolInput: Spinner
    private lateinit var hostInput: EditText
    private lateinit var portInput: EditText
    private lateinit var usernameInput: EditText
    private lateinit var passwordInput: EditText
    private lateinit var statusView: TextView
	private lateinit var connectionIndicator: TextView
	private lateinit var trafficTotalsView: TextView
	private lateinit var downloadSpeedView: TextView
	private lateinit var uploadSpeedView: TextView
	private lateinit var downloadMeter: ProgressBar
	private lateinit var uploadMeter: ProgressBar
    private lateinit var connectButton: Button
    private lateinit var disconnectButton: Button
    private var pending: ProxyInput? = null
	private var importedConfiguration = false

    private val statusReceiver = object : BroadcastReceiver() {
        override fun onReceive(context: Context?, intent: Intent?) {
			if (intent?.action == TunnelVpnService.ACTION_STATISTICS) {
				updateStatistics(
					intent.getLongExtra(TunnelVpnService.EXTRA_DOWNLOADED_BYTES, 0),
					intent.getLongExtra(TunnelVpnService.EXTRA_UPLOADED_BYTES, 0),
					intent.getLongExtra(TunnelVpnService.EXTRA_DOWNLOAD_RATE, 0),
					intent.getLongExtra(TunnelVpnService.EXTRA_UPLOAD_RATE, 0),
				)
				return
			}
            val connected = intent?.getBooleanExtra(TunnelVpnService.EXTRA_CONNECTED, false) ?: false
            val message = intent?.getStringExtra(TunnelVpnService.EXTRA_MESSAGE) ?: "Disconnected"
			if (!connected && message == "Disconnected" && importedConfiguration) return
			if (connected && importedConfiguration) {
				updateStatus(true, "Connected; imported configuration will be used after reconnecting.")
				return
			}
            updateStatus(connected, message)
        }
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        buildInterface()
        restoreFields()
        applyDeepLink(intent)
        registerStatusReceiver()
		queryTunnelState()
        requestNotificationPermission()
    }

    override fun onDestroy() {
        unregisterReceiver(statusReceiver)
        super.onDestroy()
    }

    override fun onNewIntent(intent: Intent?) {
        super.onNewIntent(intent)
        setIntent(intent)
        applyDeepLink(intent)
		queryTunnelState()
    }

    @Deprecated("VpnService.prepare still uses the activity-result contract")
    override fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?) {
        super.onActivityResult(requestCode, resultCode, data)
        if (requestCode != VPN_PERMISSION_REQUEST) return
        val configuration = pending
        pending = null
        if (resultCode == RESULT_OK && configuration != null) {
            startTunnel(configuration)
        } else {
            updateStatus(false, "VPN permission was not granted")
        }
    }

    private fun buildInterface() {
        val density = resources.displayMetrics.density
        val padding = (20 * density).toInt()
        val root = LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            setPadding(padding, padding, padding, padding)
        }

        root.addView(TextView(this).apply {
            text = "SOCKS4/SOCKS5 → system VPN"
            textSize = 22f
            setTypeface(typeface, Typeface.BOLD)
            setPadding(0, 0, 0, (16 * density).toInt())
        })

        root.addView(TextView(this).apply { text = "Protocol" })
        protocolInput = Spinner(this).apply {
            adapter = ArrayAdapter(
                this@MainActivity,
                android.R.layout.simple_spinner_dropdown_item,
                listOf("SOCKS5", "SOCKS4"),
            )
        }
        root.addView(protocolInput, ViewGroup.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT))

        hostInput = addField(root, "Server", "proxy.example.com or 2001:db8::1")
        portInput = addField(root, "Port", "1080", InputType.TYPE_CLASS_NUMBER)
        usernameInput = addField(root, "Username", "optional")
        passwordInput = addField(
            root,
            "Password",
            "not saved",
            InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_VARIATION_PASSWORD,
        )

        connectButton = Button(this).apply {
            text = "Connect"
            setOnClickListener { requestConnection() }
        }
        disconnectButton = Button(this).apply {
            text = "Disconnect"
            isEnabled = false
            setOnClickListener {
				updateStatus(true, "Disconnecting…")
                startService(Intent(this@MainActivity, TunnelVpnService::class.java).apply {
                    action = TunnelVpnService.ACTION_STOP
                })
            }
        }
        root.addView(LinearLayout(this).apply {
            orientation = LinearLayout.HORIZONTAL
            addView(connectButton, LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f))
            addView(disconnectButton, LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f))
        })

		connectionIndicator = TextView(this).apply {
			text = "●"
			textSize = 22f
			setTextColor(DISCONNECTED_COLOR)
			gravity = Gravity.CENTER_VERTICAL
			setPadding(0, 0, (8 * density).toInt(), 0)
		}
        statusView = TextView(this).apply {
            text = "Disconnected"
            textSize = 17f
			gravity = Gravity.CENTER_VERTICAL
            setTypeface(typeface, Typeface.BOLD)
        }
		root.addView(LinearLayout(this).apply {
			orientation = LinearLayout.HORIZONTAL
			gravity = Gravity.CENTER
			setPadding(0, (18 * density).toInt(), 0, (10 * density).toInt())
			addView(connectionIndicator)
			addView(statusView)
		})

		trafficTotalsView = TextView(this).apply {
			textSize = 15f
			gravity = Gravity.CENTER_HORIZONTAL
		}
		downloadSpeedView = TextView(this).apply { gravity = Gravity.CENTER_HORIZONTAL }
		uploadSpeedView = TextView(this).apply { gravity = Gravity.CENTER_HORIZONTAL }
		downloadMeter = ProgressBar(this, null, android.R.attr.progressBarStyleHorizontal).apply { max = METER_STEPS }
		uploadMeter = ProgressBar(this, null, android.R.attr.progressBarStyleHorizontal).apply { max = METER_STEPS }
		root.addView(trafficTotalsView)
		root.addView(TextView(this).apply {
			text = "Current speed"
			setTypeface(typeface, Typeface.BOLD)
			setPadding(0, (8 * density).toInt(), 0, 0)
		})
		root.addView(LinearLayout(this).apply {
			orientation = LinearLayout.HORIZONTAL
			addView(LinearLayout(this@MainActivity).apply {
				orientation = LinearLayout.VERTICAL
				addView(downloadSpeedView)
				addView(downloadMeter, ViewGroup.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, (8 * density).toInt()))
			}, LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f))
			addView(LinearLayout(this@MainActivity).apply {
				orientation = LinearLayout.VERTICAL
				addView(uploadSpeedView)
				addView(uploadMeter, ViewGroup.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, (8 * density).toInt()))
			}, LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f))
		})
		updateStatistics(0, 0, 0, 0)
        root.addView(TextView(this).apply {
            text = "Android will show a system prompt to create the VPN. The password is kept in memory only until disconnection."
            textSize = 14f
        })
        setContentView(ScrollView(this).apply { addView(root) })
    }

    private fun addField(parent: LinearLayout, label: String, hint: String, type: Int = InputType.TYPE_CLASS_TEXT): EditText {
        parent.addView(TextView(this).apply { text = label })
        return EditText(this).also { field ->
            field.hint = hint
            field.inputType = type
            field.isSingleLine = true
            parent.addView(field, ViewGroup.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT))
        }
    }

    private fun applyDeepLink(source: Intent?) {
        val raw = source?.dataString ?: return
        source.data = null
        val imported = runCatching { ProxyDeepLink.parse(raw) }.getOrElse { error ->
            Toast.makeText(this, error.message ?: "Invalid configuration link", Toast.LENGTH_LONG).show()
            return
        }
        protocolInput.setSelection(if (imported.scheme == "socks4") 1 else 0)
        hostInput.setText(imported.host.removePrefix("[").removeSuffix("]"))
        portInput.setText(imported.port.toString())
        usernameInput.setText(imported.username)
        passwordInput.setText(imported.password)
		importedConfiguration = true
        updateStatus(false, "Configuration imported from link. Review it and tap Connect.")
    }

    private fun requestConnection() {
        val configuration = validateFields() ?: return
        saveNonSecretFields(configuration)
        val permissionIntent = VpnService.prepare(this)
        if (permissionIntent == null) {
            startTunnel(configuration)
        } else {
            pending = configuration
            @Suppress("DEPRECATION")
            startActivityForResult(permissionIntent, VPN_PERMISSION_REQUEST)
        }
    }

    private fun startTunnel(configuration: ProxyInput) {
		importedConfiguration = false
		updateStatistics(0, 0, 0, 0)
		updateStatus(true, "Connecting…")
        val intent = Intent(this, TunnelVpnService::class.java).apply {
            action = TunnelVpnService.ACTION_START
            putExtra(TunnelVpnService.EXTRA_HOST, configuration.host)
            putExtra(TunnelVpnService.EXTRA_SCHEME, configuration.scheme)
            putExtra(TunnelVpnService.EXTRA_PORT, configuration.port)
            putExtra(TunnelVpnService.EXTRA_USERNAME, configuration.username)
            putExtra(TunnelVpnService.EXTRA_PASSWORD, configuration.password)
        }
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) startForegroundService(intent) else startService(intent)
    }

    private fun validateFields(): ProxyInput? {
        val scheme = protocolInput.selectedItem.toString().lowercase()
        val host = hostInput.text.toString().trim().removePrefix("[").removeSuffix("]")
        val port = portInput.text.toString().trim().toIntOrNull()
        val username = usernameInput.text.toString()
        val password = passwordInput.text.toString()
        val error = when {
            host.isEmpty() || host.any { it.isWhitespace() || it == '/' || it == '\\' } -> "Enter a valid server address"
            port == null || port !in 1..65535 -> "Port must be a number from 1 to 65535"
            username.isEmpty() && password.isNotEmpty() -> "A password was provided without a username"
            scheme == "socks4" && password.isNotEmpty() -> "SOCKS4 supports a user ID but not a password"
            else -> null
        }
        if (error != null) {
            Toast.makeText(this, error, Toast.LENGTH_LONG).show()
            return null
        }
        return ProxyInput(scheme, host, port!!, username, password)
    }

    private fun updateStatus(connected: Boolean, message: String) {
        statusView.text = message
		connectionIndicator.setTextColor(
			when {
				!connected -> DISCONNECTED_COLOR
				message.startsWith("Connecting") || message.startsWith("Disconnecting") -> CONNECTING_COLOR
				else -> CONNECTED_COLOR
			},
		)
        connectButton.isEnabled = !connected
        disconnectButton.isEnabled = connected
    }

	private fun updateStatistics(downloaded: Long, uploaded: Long, downloadRate: Long, uploadRate: Long) {
		trafficTotalsView.text = "Session traffic: ↓ Download ${formatBytes(downloaded)}    ↑ Upload ${formatBytes(uploaded)}"
		downloadSpeedView.text = "↓ ${formatBytes(downloadRate)}/s"
		uploadSpeedView.text = "↑ ${formatBytes(uploadRate)}/s"
		val scale = speedMeterScale(maxOf(downloadRate, uploadRate))
		downloadMeter.progress = ((downloadRate.coerceAtLeast(0).toDouble() / scale) * METER_STEPS).toInt().coerceIn(0, METER_STEPS)
		uploadMeter.progress = ((uploadRate.coerceAtLeast(0).toDouble() / scale) * METER_STEPS).toInt().coerceIn(0, METER_STEPS)
	}

	private fun formatBytes(value: Long): String {
		val safe = value.coerceAtLeast(0)
		if (safe < 1024) return "$safe B"
		var divisor = 1024.0
		var unit = 0
		while (safe / divisor >= 1024 && unit < 4) {
			divisor *= 1024
			unit++
		}
		return String.format(java.util.Locale.US, "%.1f %ciB", safe / divisor, "KMGTPE"[unit])
	}

	private fun speedMeterScale(value: Long): Double {
		var scale = 1024.0
		while (scale < value.coerceAtLeast(0) && scale < Long.MAX_VALUE / 8.0) scale *= 8
		return scale
	}

    private fun saveNonSecretFields(input: ProxyInput) {
        getSharedPreferences(PREFERENCES, MODE_PRIVATE).edit()
            .putString("host", input.host)
            .putString("scheme", input.scheme)
            .putInt("port", input.port)
            .putString("username", input.username)
            .apply()
    }

    private fun restoreFields() {
        val preferences = getSharedPreferences(PREFERENCES, MODE_PRIVATE)
        protocolInput.setSelection(if (preferences.getString("scheme", "socks5") == "socks4") 1 else 0)
        hostInput.setText(preferences.getString("host", ""))
        portInput.setText(preferences.getInt("port", 1080).toString())
        usernameInput.setText(preferences.getString("username", ""))
    }

    private fun registerStatusReceiver() {
		val filter = IntentFilter().apply {
			addAction(TunnelVpnService.ACTION_STATUS)
			addAction(TunnelVpnService.ACTION_STATISTICS)
		}
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            registerReceiver(statusReceiver, filter, RECEIVER_NOT_EXPORTED)
        } else {
            @Suppress("DEPRECATION")
            registerReceiver(statusReceiver, filter)
        }
    }

	private fun queryTunnelState() {
		startService(Intent(this, TunnelVpnService::class.java).apply {
			action = TunnelVpnService.ACTION_QUERY
		})
	}

    private fun requestNotificationPermission() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU &&
            checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) != PackageManager.PERMISSION_GRANTED
        ) {
            requestPermissions(arrayOf(Manifest.permission.POST_NOTIFICATIONS), NOTIFICATION_PERMISSION_REQUEST)
        }
    }

    private data class ProxyInput(
        val scheme: String,
        val host: String,
        val port: Int,
        val username: String,
        val password: String,
    )

    companion object {
        private const val VPN_PERMISSION_REQUEST = 100
        private const val NOTIFICATION_PERMISSION_REQUEST = 101
        private const val PREFERENCES = "proxy"
		private const val METER_STEPS = 1000
		private val DISCONNECTED_COLOR = Color.rgb(217, 50, 50)
		private val CONNECTING_COLOR = Color.rgb(240, 162, 2)
		private val CONNECTED_COLOR = Color.rgb(32, 177, 90)
    }
}
