package com.santaklouse.gosocks2vpn

import org.junit.Assert.assertEquals
import org.junit.Assert.assertThrows
import org.junit.Test

class ProxyDeepLinkTest {
    @Test
    fun parsesSocks5Credentials() {
        assertEquals(
            ProxyDeepLink("socks5", "proxyhost", 1080, "proxyuser", "proxypass"),
            ProxyDeepLink.parse("socks2vpn://socks5-proxyuser:proxypass@proxyhost:1080"),
        )
    }

    @Test
    fun parsesAliasEscapingAndIpv6() {
        assertEquals(
            ProxyDeepLink("socks5", "[2001:db8::1]", 443, "user@example", "p@ss:word"),
            ProxyDeepLink.parse("socks2vps://socks5-user%40example:p%40ss%3Aword@[2001:db8::1]:443"),
        )
    }

    @Test
    fun rejectsPasswordForSocks4() {
        assertThrows(IllegalArgumentException::class.java) {
            ProxyDeepLink.parse("socks2vpn://socks4-user:secret@proxyhost:1080")
        }
    }
}
