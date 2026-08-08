package deeplink

import (
	"testing"

	proxyconfig "github.com/santaklouse/go-socks2vpn/internal/proxy"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		link     string
		expected proxyconfig.Settings
	}{
		{
			name: "SOCKS5 credentials",
			link: "socks2vpn://socks5-proxyuser:proxypass@proxyhost:1080",
			expected: proxyconfig.Settings{
				Scheme: proxyconfig.SchemeSOCKS5, Host: "proxyhost", Port: 1080,
				Username: "proxyuser", Password: "proxypass",
			},
		},
		{
			name: "legacy alias and escaped credentials",
			link: "socks2vps://socks5-user%40example:p%40ss%3Aword@[2001:db8::1]:443",
			expected: proxyconfig.Settings{
				Scheme: proxyconfig.SchemeSOCKS5, Host: "2001:db8::1", Port: 443,
				Username: "user@example", Password: "p@ss:word",
			},
		},
		{
			name: "SOCKS4 without credentials",
			link: "socks2vpn://socks4@192.0.2.10:9050",
			expected: proxyconfig.Settings{
				Scheme: proxyconfig.SchemeSOCKS4, Host: "192.0.2.10", Port: 9050,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := Parse(test.link)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.expected {
				t.Fatalf("Parse() = %#v, want %#v", got, test.expected)
			}
		})
	}
}

func TestParseRejectsInvalidLinks(t *testing.T) {
	for _, link := range []string{
		"https://socks5-user:pass@proxyhost:1080",
		"socks2vpn://proxyhost:1080",
		"socks2vpn://socks5:pass@proxyhost:1080",
		"socks2vpn://socks4-user:pass@proxyhost:1080",
		"socks2vpn://socks5-user:pass@proxyhost:not-a-port",
		"socks2vpn://socks5-user:pass@proxyhost:1080/path",
	} {
		if _, err := Parse(link); err == nil {
			t.Fatalf("Parse(%q) unexpectedly succeeded", link)
		}
	}
}
