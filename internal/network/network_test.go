package network

import "testing"

func TestParseLinuxRoutes(t *testing.T) {
	input := "default via 10.0.0.1 dev eth0 proto dhcp src 10.0.0.20 metric 100\ndefault via 10.1.0.1 dev wlan0 metric 600\n"
	got, err := parseLinuxRoutes(input)
	if err != nil {
		t.Fatal(err)
	}
	if got.Interface != "eth0" || got.Gateway != "10.0.0.1" || len(got.DefaultRoutes) != 2 {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestParseDarwinRoute(t *testing.T) {
	got, err := parseDarwinRoute("route to: default\ngateway: 192.168.1.1\ninterface: en0\n")
	if err != nil || got.Interface != "en0" || got.Gateway != "192.168.1.1" {
		t.Fatalf("unexpected result: %#v, %v", got, err)
	}
}

func TestParseWindowsRoute(t *testing.T) {
	got, err := parseWindowsRoute("Wi-Fi\r\n192.168.0.1\r\n")
	if err != nil || got.Interface != "Wi-Fi" || got.Gateway != "192.168.0.1" {
		t.Fatalf("unexpected result: %#v, %v", got, err)
	}
}
