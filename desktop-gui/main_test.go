package main

import (
	"image/color"
	"strings"
	"testing"

	proxyconfig "github.com/santaklouse/go-socks2vpn/internal/proxy"
)

func TestPrivilegeMessage(t *testing.T) {
	tests := []struct {
		name     string
		goos     string
		expected string
	}{
		{name: "Windows", goos: "windows", expected: "Run as administrator"},
		{name: "macOS", goos: "darwin", expected: "sudo -E socks2vpn-gui"},
		{name: "Linux", goos: "linux", expected: "sudo -E socks2vpn-gui"},
		{name: "Other", goos: "freebsd", expected: "Administrator privileges"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			message := privilegeMessage(test.goos)
			if !strings.Contains(message, test.expected) {
				t.Fatalf("privilegeMessage(%q) = %q, expected text %q", test.goos, message, test.expected)
			}
		})
	}
}

func TestConnectionStateColors(t *testing.T) {
	tests := []struct {
		state connectionState
		want  color.Color
	}{
		{stateDisconnected, disconnectedColor},
		{stateConnecting, connectingColor},
		{stateConnected, connectedColor},
		{stateDisconnecting, connectingColor},
	}
	for _, test := range tests {
		if got := test.state.color(); got != test.want {
			t.Errorf("state %d color = %v, want %v", test.state, got, test.want)
		}
	}
}

func TestSpeedMeterScale(t *testing.T) {
	for value, want := range map[uint64]uint64{
		0:       1 << 10,
		1 << 10: 1 << 10,
		1 << 11: 1 << 13,
		1 << 20: 1 << 22,
	} {
		if got := speedMeterScale(value); got != want {
			t.Errorf("speedMeterScale(%d) = %d, want %d", value, got, want)
		}
	}
}

func TestSettingsFromArgs(t *testing.T) {
	expected := proxyconfig.Settings{
		Scheme: proxyconfig.SchemeSOCKS5,
		Host:   "proxyhost", Port: 1080,
		Username: "proxyuser", Password: "proxypass",
	}
	for _, args := range [][]string{
		{"socks2vpn://socks5-proxyuser:proxypass@proxyhost:1080"},
		{"--deep-link", "socks2vps://socks5-proxyuser:proxypass@proxyhost:1080"},
		{"--deep-link=socks2vpn://socks5-proxyuser:proxypass@proxyhost:1080"},
	} {
		got, found, err := settingsFromArgs(args)
		if err != nil || !found || got != expected {
			t.Fatalf("settingsFromArgs(%q) = %#v, %t, %v", args, got, found, err)
		}
	}
}

func TestSettingsFromArgsIgnoresUnrelatedArguments(t *testing.T) {
	_, found, err := settingsFromArgs([]string{"--verbose"})
	if err != nil || found {
		t.Fatalf("settingsFromArgs() found=%t, err=%v", found, err)
	}
}
