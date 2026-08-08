package network

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/santaklouse/go-socks2vpn/internal/command"
)

type Info struct {
	Interface     string
	Gateway       string
	DefaultRoutes [][]string
}

func Detect(ctx context.Context, goos string, runner command.Runner) (Info, error) {
	switch goos {
	case "linux":
		out, err := runner.Output(ctx, command.C("ip", "-4", "route", "show", "default"))
		if err != nil {
			return Info{}, fmt.Errorf("could not detect the primary Linux route: %w", err)
		}
		return parseLinuxRoutes(string(out))
	case "darwin":
		out, err := runner.Output(ctx, command.C("route", "-n", "get", "default"))
		if err != nil {
			return Info{}, fmt.Errorf("could not detect the primary macOS route: %w", err)
		}
		return parseDarwinRoute(string(out))
	case "windows":
		script := `$r=Get-NetRoute -AddressFamily IPv4 -DestinationPrefix '0.0.0.0/0' | Where-Object {$_.State -eq 'Alive'} | Sort-Object RouteMetric,InterfaceMetric | Select-Object -First 1; if ($null -eq $r) { exit 2 }; $a=Get-NetAdapter -InterfaceIndex $r.InterfaceIndex; [Console]::OutputEncoding=[Text.Encoding]::UTF8; Write-Output $a.Name; Write-Output $r.NextHop`
		out, err := runner.Output(ctx, command.C("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script))
		if err != nil {
			return Info{}, fmt.Errorf("could not detect the primary Windows route: %w", err)
		}
		return parseWindowsRoute(string(out))
	default:
		return Info{}, fmt.Errorf("network detection is not supported on %s", goos)
	}
}

func parseLinuxRoutes(output string) (Info, error) {
	var routes [][]string
	bestMetric := int(^uint(0) >> 1)
	best := Info{}
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || fields[0] != "default" {
			continue
		}
		routes = append(routes, fields)
		iface, gateway, metric := fieldAfter(fields, "dev"), fieldAfter(fields, "via"), 0
		if raw := fieldAfter(fields, "metric"); raw != "" {
			metric, _ = strconv.Atoi(raw)
		}
		if iface != "" && metric < bestMetric {
			bestMetric = metric
			best.Interface, best.Gateway = iface, gateway
		}
	}
	if best.Interface == "" {
		return Info{}, errors.New("default route with a network interface was not found")
	}
	best.DefaultRoutes = routes
	return best, nil
}

func parseDarwinRoute(output string) (Info, error) {
	info := Info{}
	for _, line := range strings.Split(output, "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok {
			continue
		}
		switch strings.TrimSpace(key) {
		case "interface":
			info.Interface = strings.TrimSpace(value)
		case "gateway":
			info.Gateway = strings.TrimSpace(value)
		}
	}
	if info.Interface == "" {
		return Info{}, errors.New("primary macOS route interface was not found")
	}
	return info, nil
}

func parseWindowsRoute(output string) (Info, error) {
	lines := strings.Split(strings.ReplaceAll(strings.TrimSpace(output), "\r", ""), "\n")
	if len(lines) < 2 || strings.TrimSpace(lines[0]) == "" {
		return Info{}, errors.New("PowerShell did not return an interface and gateway")
	}
	return Info{Interface: strings.TrimSpace(lines[0]), Gateway: strings.TrimSpace(lines[1])}, nil
}

func fieldAfter(fields []string, key string) string {
	for i := 0; i+1 < len(fields); i++ {
		if fields[i] == key {
			return fields[i+1]
		}
	}
	return ""
}
