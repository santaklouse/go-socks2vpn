package proxy

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		input string
		want  Settings
		url   string
	}{
		{"proxy.example:1080", Settings{Scheme: SchemeSOCKS5, Host: "proxy.example", Port: 1080}, "socks5://proxy.example:1080"},
		{"proxy.example:1080:alice:p:a:ss", Settings{Scheme: SchemeSOCKS5, Host: "proxy.example", Port: 1080, Username: "alice", Password: "p:a:ss"}, "socks5://alice:p%3Aa%3Ass@proxy.example:1080"},
		{"[2001:db8::1]:1080:user:pass", Settings{Scheme: SchemeSOCKS5, Host: "2001:db8::1", Port: 1080, Username: "user", Password: "pass"}, "socks5://user:pass@[2001:db8::1]:1080"},
		{"socks5://u:p%40ss@127.0.0.1:9050", Settings{Scheme: SchemeSOCKS5, Host: "127.0.0.1", Port: 9050, Username: "u", Password: "p@ss"}, "socks5://u:p%40ss@127.0.0.1:9050"},
		{"socks4://192.168.192.100:9050", Settings{Scheme: SchemeSOCKS4, Host: "192.168.192.100", Port: 9050}, "socks4://192.168.192.100:9050"},
		{"socks4a://userid@proxy.example:9050", Settings{Scheme: SchemeSOCKS4, Host: "proxy.example", Port: 9050, Username: "userid"}, "socks4://userid@proxy.example:9050"},
	}
	for _, tt := range tests {
		got, err := Parse(tt.input)
		if err != nil {
			t.Fatalf("Parse(%q): %v", tt.input, err)
		}
		if got != tt.want {
			t.Errorf("Parse(%q) = %#v, want %#v", tt.input, got, tt.want)
		}
		if got.URL() != tt.url {
			t.Errorf("URL() = %q, want %q", got.URL(), tt.url)
		}
	}
}

func TestParseRejectsInvalidValues(t *testing.T) {
	for _, input := range []string{"", "host", "host:nope", "host:70000", "http://host:1080", "host:1080::secret", "socks4://user:password@host:1080"} {
		if _, err := Parse(input); err == nil {
			t.Errorf("Parse(%q) unexpectedly succeeded", input)
		}
	}
}

func TestRedactedURL(t *testing.T) {
	s := Settings{Scheme: SchemeSOCKS5, Host: "example.com", Port: 1080, Username: "user", Password: "secret"}
	if got := s.RedactedURL(); got != "socks5://user:%2A%2A%2A@example.com:1080" {
		t.Fatalf("RedactedURL() = %q", got)
	}
}
