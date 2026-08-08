package main

import (
	"strings"
	"testing"
)

func TestPrivilegeMessage(t *testing.T) {
	tests := []struct {
		name     string
		goos     string
		expected string
	}{
		{name: "Windows", goos: "windows", expected: "Запуск от имени администратора"},
		{name: "macOS", goos: "darwin", expected: "sudo -E socks2vpn-gui"},
		{name: "Linux", goos: "linux", expected: "sudo -E socks2vpn-gui"},
		{name: "Other", goos: "freebsd", expected: "права администратора"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			message := privilegeMessage(test.goos)
			if !strings.Contains(message, test.expected) {
				t.Fatalf("privilegeMessage(%q) = %q, ожидается текст %q", test.goos, message, test.expected)
			}
		})
	}
}
