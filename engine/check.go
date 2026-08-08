package engine

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"strconv"

	"github.com/xjasonlyu/tun2socks/v2/dialer"
	"github.com/xjasonlyu/tun2socks/v2/metadata"
)

// CheckProxy performs the same TCP proxy handshake used by the TUN engine.
func CheckProxy(ctx context.Context, proxyURL, destination string) error {
	proxy, err := parseProxy(proxyURL)
	if err != nil {
		return err
	}
	host, portText, err := net.SplitHostPort(destination)
	if err != nil {
		return fmt.Errorf("invalid check destination %q: %w", destination, err)
	}
	port, err := strconv.ParseUint(portText, 10, 16)
	if err != nil || port == 0 {
		return fmt.Errorf("invalid check destination port %q", portText)
	}
	ip, err := resolveDestination(ctx, host)
	if err != nil {
		return err
	}

	dialer.Reset()
	connection, err := proxy.DialContext(ctx, &metadata.Metadata{
		Network: metadata.TCP,
		DstIP:   ip,
		DstPort: uint16(port),
	})
	if err != nil {
		return fmt.Errorf("proxy handshake to %s failed: %w", destination, err)
	}
	return connection.Close()
}

func resolveDestination(ctx context.Context, host string) (netip.Addr, error) {
	if ip, err := netip.ParseAddr(host); err == nil {
		return ip.Unmap(), nil
	}
	addresses, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("cannot resolve check destination %s: %w", host, err)
	}
	for _, address := range addresses {
		if address.Is4() || address.Is4In6() {
			return address.Unmap(), nil
		}
	}
	if len(addresses) > 0 {
		return addresses[0], nil
	}
	return netip.Addr{}, fmt.Errorf("check destination %s has no IP addresses", host)
}
