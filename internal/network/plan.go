package network

import (
	"context"
	"fmt"
	"strings"

	"github.com/santaklouse/go-socks2vpn/internal/command"
)

type Timing int

const (
	BeforeProcess Timing = iota
	AfterProcess
)

type Plan struct {
	Device        string
	Timing        Timing
	WaitInterface string
	Steps         []Step
}

type Step struct {
	Do   command.Spec
	Undo []command.Spec
}

func BuildPlan(ctx context.Context, goos string, info Info, dns string, runner command.Runner) (Plan, error) {
	switch goos {
	case "linux":
		return linuxPlan(ctx, info, runner)
	case "darwin":
		return darwinPlan(info), nil
	case "windows":
		return windowsPlan(dns), nil
	default:
		return Plan{}, fmt.Errorf("настройка VPN не поддерживается для %s", goos)
	}
}

func linuxPlan(ctx context.Context, info Info, runner command.Runner) (Plan, error) {
	if info.Interface == "" {
		return Plan{}, fmt.Errorf("не указан основной сетевой интерфейс")
	}
	allRP, err := sysctlValue(ctx, runner, "net.ipv4.conf.all.rp_filter")
	if err != nil {
		return Plan{}, err
	}
	interfaceKey := "net.ipv4.conf." + info.Interface + ".rp_filter"
	interfaceRP, err := sysctlValue(ctx, runner, interfaceKey)
	if err != nil {
		return Plan{}, err
	}

	steps := []Step{
		{Do: command.C("ip", "tuntap", "add", "mode", "tun", "dev", "tun0"), Undo: []command.Spec{command.C("ip", "link", "delete", "dev", "tun0")}},
		{Do: command.C("ip", "addr", "add", "198.18.0.1/15", "dev", "tun0"), Undo: []command.Spec{command.C("ip", "addr", "del", "198.18.0.1/15", "dev", "tun0")}},
		{Do: command.C("ip", "link", "set", "dev", "tun0", "up"), Undo: []command.Spec{command.C("ip", "link", "set", "dev", "tun0", "down")}},
	}
	for _, route := range info.DefaultRoutes {
		steps = append(steps, Step{
			Do:   command.C("ip", append([]string{"route", "del"}, route...)...),
			Undo: []command.Spec{command.C("ip", append([]string{"route", "add"}, route...)...)},
		})
	}
	steps = append(steps, Step{
		Do:   command.C("ip", "route", "add", "default", "via", "198.18.0.1", "dev", "tun0", "metric", "1"),
		Undo: []command.Spec{command.C("ip", "route", "del", "default", "via", "198.18.0.1", "dev", "tun0", "metric", "1")},
	})
	primary := []string{"route", "add", "default"}
	primaryDelete := []string{"route", "del", "default"}
	if info.Gateway != "" {
		primary = append(primary, "via", info.Gateway)
		primaryDelete = append(primaryDelete, "via", info.Gateway)
	}
	primary = append(primary, "dev", info.Interface, "metric", "10")
	primaryDelete = append(primaryDelete, "dev", info.Interface, "metric", "10")
	steps = append(steps,
		Step{Do: command.C("ip", primary...), Undo: []command.Spec{command.C("ip", primaryDelete...)}},
		Step{Do: command.C("sysctl", "-w", "net.ipv4.conf.all.rp_filter=0"), Undo: []command.Spec{command.C("sysctl", "-w", "net.ipv4.conf.all.rp_filter="+allRP)}},
		Step{Do: command.C("sysctl", "-w", interfaceKey+"=0"), Undo: []command.Spec{command.C("sysctl", "-w", interfaceKey+"="+interfaceRP)}},
	)
	return Plan{Device: "tun0", Timing: BeforeProcess, Steps: steps}, nil
}

func darwinPlan(info Info) Plan {
	routes := []struct{ network, prefix string }{
		{"1.0.0.0", "8"}, {"2.0.0.0", "7"}, {"4.0.0.0", "6"}, {"8.0.0.0", "5"},
		{"16.0.0.0", "4"}, {"32.0.0.0", "3"}, {"64.0.0.0", "2"}, {"128.0.0.0", "1"}, {"198.18.0.0", "15"},
	}
	steps := []Step{{
		Do:   command.C("ifconfig", "utun123", "198.18.0.1", "198.18.0.1", "up"),
		Undo: []command.Spec{command.C("ifconfig", "utun123", "down")},
	}}
	for _, route := range routes {
		steps = append(steps, Step{
			Do:   command.C("route", "-n", "add", "-net", route.network+"/"+route.prefix, "198.18.0.1"),
			Undo: []command.Spec{command.C("route", "-n", "delete", "-net", route.network+"/"+route.prefix, "198.18.0.1")},
		})
	}
	return Plan{Device: "utun123", Timing: AfterProcess, WaitInterface: "utun123", Steps: steps}
}

func windowsPlan(dns string) Plan {
	if dns == "" {
		dns = "8.8.8.8"
	}
	return Plan{
		Device:        "wintun",
		Timing:        AfterProcess,
		WaitInterface: "wintun",
		Steps: []Step{
			{
				Do:   command.C("netsh", "interface", "ipv4", "set", "address", "name=wintun", "source=static", "addr=192.168.123.1", "mask=255.255.255.0"),
				Undo: []command.Spec{command.C("netsh", "interface", "ipv4", "delete", "address", "name=wintun", "addr=192.168.123.1")},
			},
			{
				Do:   command.C("netsh", "interface", "ipv4", "set", "dnsservers", "name=wintun", "static", "address="+dns, "register=none", "validate=no"),
				Undo: []command.Spec{command.C("netsh", "interface", "ipv4", "set", "dnsservers", "name=wintun", "source=dhcp")},
			},
			{
				Do:   command.C("netsh", "interface", "ipv4", "add", "route", "0.0.0.0/0", "wintun", "192.168.123.1", "metric=1"),
				Undo: []command.Spec{command.C("netsh", "interface", "ipv4", "delete", "route", "0.0.0.0/0", "wintun", "192.168.123.1")},
			},
		},
	}
}

func sysctlValue(ctx context.Context, runner command.Runner, key string) (string, error) {
	out, err := runner.Output(ctx, command.C("sysctl", "-n", key))
	if err != nil {
		return "", fmt.Errorf("не удалось прочитать %s: %w", key, err)
	}
	value := strings.TrimSpace(string(out))
	if value == "" {
		return "", fmt.Errorf("sysctl %s вернул пустое значение", key)
	}
	return value, nil
}
