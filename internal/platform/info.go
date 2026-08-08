package platform

import (
	"os"
	"runtime"
	"strings"

	"golang.org/x/sys/cpu"
)

// Info describes the platform running the desktop client.
type Info struct {
	OS         string
	Arch       string
	AMD64V3    bool
	ARMVariant string
	Musl       bool
}

// Detect returns information about the currently running process.
func Detect() Info {
	i := Info{OS: runtime.GOOS, Arch: runtime.GOARCH}
	if i.Arch == "amd64" {
		i.AMD64V3 = cpu.X86.HasAVX2
	}
	if i.OS == "linux" {
		i.Musl = detectMusl()
		if i.Arch == "arm" {
			i.ARMVariant = detectARMVariant()
		}
	}
	return i
}

func (i Info) Description() string {
	desc := i.OS + "/" + i.Arch
	if i.AMD64V3 {
		desc += " (AVX2)"
	}
	if i.ARMVariant != "" {
		desc += " (" + i.ARMVariant + ")"
	}
	if i.Musl {
		desc += " (musl)"
	}
	return desc
}

func detectMusl() bool {
	if data, err := os.ReadFile("/etc/os-release"); err == nil && strings.Contains(strings.ToLower(string(data)), "alpine") {
		return true
	}
	for _, path := range []string{
		"/lib/ld-musl-x86_64.so.1",
		"/lib/ld-musl-aarch64.so.1",
		"/lib/ld-musl-armhf.so.1",
		"/lib/ld-musl-riscv64.so.1",
	} {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

func detectARMVariant() string {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return "armv7"
	}
	lower := strings.ToLower(string(data))
	for _, variant := range []string{"armv7", "armv6", "armv5"} {
		if strings.Contains(lower, variant) {
			return variant
		}
	}
	return "armv7"
}
