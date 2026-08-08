// Package mobile exposes the shared embedded tun2socks engine to Android via gomobile.
package mobile

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	tunengine "github.com/santaklouse/go-socks2vpn/engine"
	"golang.org/x/sys/unix"
)

var (
	mu      sync.Mutex
	current *tunengine.Engine
)

// Start attaches tun2socks to a file descriptor created by Android VpnService.
func Start(fd int, proxyURL string, mtu int) error {
	mu.Lock()
	defer mu.Unlock()
	if current != nil {
		_ = unix.Close(fd)
		return errors.New("tun2socks engine is already active")
	}
	if fd < 0 {
		return errors.New("invalid VPN file descriptor")
	}
	instance := tunengine.New()
	if err := instance.Start(tunengine.Config{
		Device:   "fd://" + strconv.Itoa(fd),
		ProxyURL: proxyURL,
		MTU:      mtu,
		DNSOverHTTPS: &tunengine.DNSOverHTTPSConfig{
			URL:     "https://cloudflare-dns.com/dns-query",
			Address: "1.1.1.1:443",
		},
	}); err != nil {
		return err
	}
	current = instance
	return nil
}

// CheckProxy performs an authenticated SOCKS handshake before Android installs
// the full-tunnel route.
func CheckProxy(proxyURL string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	if err := tunengine.CheckProxy(ctx, proxyURL, "1.1.1.1:443"); err != nil {
		return err
	}
	return tunengine.CheckDNSOverHTTPS(ctx, proxyURL, tunengine.DNSOverHTTPSConfig{
		URL:     "https://cloudflare-dns.com/dns-query",
		Address: "1.1.1.1:443",
	})
}

// Stop releases the VPN file descriptor and all network-stack resources.
func Stop() {
	mu.Lock()
	defer mu.Unlock()
	if current == nil {
		return
	}
	current.Stop()
	current = nil
}

// UploadedBytes returns the number of IP bytes sent by applications through
// the VPN during the current session.
func UploadedBytes() int64 {
	mu.Lock()
	defer mu.Unlock()
	if current == nil {
		return 0
	}
	return int64(current.Statistics().UploadedBytes)
}

// DownloadedBytes returns the number of IP bytes delivered from the VPN to
// applications during the current session.
func DownloadedBytes() int64 {
	mu.Lock()
	defer mu.Unlock()
	if current == nil {
		return 0
	}
	return int64(current.Statistics().DownloadedBytes)
}
