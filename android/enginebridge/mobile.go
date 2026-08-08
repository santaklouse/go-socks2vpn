// Package mobile exposes the shared embedded tun2socks engine to Android via gomobile.
package mobile

import (
	"errors"
	"strconv"
	"sync"

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
	}); err != nil {
		return err
	}
	current = instance
	return nil
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
