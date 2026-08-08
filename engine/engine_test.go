package engine

import "testing"

func TestParseProxy(t *testing.T) {
	for _, input := range []string{
		"socks5://alice:p%40ss@[2001:db8::10]:1080",
		"socks5h://127.0.0.1:1080",
		"socks4://userid@192.168.192.100:9050",
		"socks4a://192.168.192.100:9050",
	} {
		if _, err := parseProxy(input); err != nil {
			t.Fatalf("parseProxy(%q): %v", input, err)
		}
	}
	for _, input := range []string{"", "http://127.0.0.1:8080", "socks5://127.0.0.1", "socks4://user:password@127.0.0.1:9050"} {
		if _, err := parseProxy(input); err == nil {
			t.Errorf("parseProxy(%q) unexpectedly succeeded", input)
		}
	}
}

func TestOpenDeviceValidation(t *testing.T) {
	if _, err := openDevice("fd://not-a-number", 1500); err == nil {
		t.Fatal("invalid fd unexpectedly accepted")
	}
	if _, err := openDevice("", 1500); err == nil {
		t.Fatal("empty device unexpectedly accepted")
	}
}

func TestStoppedEngineStatisticsAreZero(t *testing.T) {
	if got := New().Statistics(); got != (Statistics{}) {
		t.Fatalf("new engine statistics = %#v", got)
	}
}
