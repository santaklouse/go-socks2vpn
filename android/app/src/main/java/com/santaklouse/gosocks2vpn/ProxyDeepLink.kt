package com.santaklouse.gosocks2vpn

import java.net.URI
import java.util.Locale

data class ProxyDeepLink(
    val scheme: String,
    val host: String,
    val port: Int,
    val username: String,
    val password: String,
) {
    companion object {
        fun parse(raw: String): ProxyDeepLink {
            val uri = runCatching { URI(raw.trim()) }
                .getOrElse { throw IllegalArgumentException("Invalid go-socks2vpn link") }
            val outerScheme = uri.scheme?.lowercase(Locale.ROOT)
            require(outerScheme == "socks2vpn" || outerScheme == "socks2vps") {
                "Unsupported configuration link scheme"
            }
            require(uri.rawPath.isNullOrEmpty() || uri.rawPath == "/") {
                "Configuration link must not contain a path"
            }
            require(uri.rawQuery == null && uri.rawFragment == null) {
                "Configuration link must not contain a query or fragment"
            }

            val userInfo = uri.userInfo
                ?: throw IllegalArgumentException("Configuration link must specify SOCKS4 or SOCKS5")
            val separator = userInfo.indexOf(':')
            val identity = if (separator < 0) userInfo else userInfo.substring(0, separator)
            val password = if (separator < 0) "" else userInfo.substring(separator + 1)
            val lowerIdentity = identity.lowercase(Locale.ROOT)
            val scheme: String
            val username: String
            when {
                lowerIdentity == "socks5" -> {
                    scheme = "socks5"
                    username = ""
                }
                lowerIdentity.startsWith("socks5-") -> {
                    scheme = "socks5"
                    username = identity.substring("socks5-".length)
                }
                lowerIdentity == "socks4" -> {
                    scheme = "socks4"
                    username = ""
                }
                lowerIdentity.startsWith("socks4-") -> {
                    scheme = "socks4"
                    username = identity.substring("socks4-".length)
                }
                else -> throw IllegalArgumentException("Proxy credentials must start with SOCKS4 or SOCKS5")
            }

            require(scheme != "socks4" || separator < 0) { "SOCKS4 links must not contain a password" }
            require(username.isNotEmpty() || password.isEmpty()) { "A password requires a proxy username" }
            val host = uri.host ?: throw IllegalArgumentException("Configuration link must contain a proxy host")
            require(host.none { it.isWhitespace() || it == '/' || it == '\\' }) {
                "Configuration link contains an invalid proxy host"
            }
            require(uri.port in 1..65535) { "Configuration link must contain a numeric proxy port" }
            return ProxyDeepLink(scheme, host, uri.port, username, password)
        }
    }
}
