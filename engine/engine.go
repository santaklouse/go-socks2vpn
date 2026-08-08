// Package engine embeds the upstream xjasonlyu/tun2socks network stack.
package engine

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"gvisor.dev/gvisor/pkg/tcpip/stack"

	"github.com/xjasonlyu/tun2socks/v2/core"
	"github.com/xjasonlyu/tun2socks/v2/core/device"
	"github.com/xjasonlyu/tun2socks/v2/core/device/fdbased"
	"github.com/xjasonlyu/tun2socks/v2/core/device/tun"
	"github.com/xjasonlyu/tun2socks/v2/dialer"
	upstreamlog "github.com/xjasonlyu/tun2socks/v2/log"
	upstreamproxy "github.com/xjasonlyu/tun2socks/v2/proxy"
	"github.com/xjasonlyu/tun2socks/v2/proxy/socks4"
	"github.com/xjasonlyu/tun2socks/v2/proxy/socks5"
	"github.com/xjasonlyu/tun2socks/v2/tunnel"
)

type Config struct {
	Device    string
	Interface string
	ProxyURL  string
	MTU       int
}

type Engine struct {
	mu     sync.Mutex
	device device.Device
	stack  *stack.Stack
}

func New() *Engine {
	return &Engine{}
}

// Start builds the tun2socks stack in-process. Unlike upstream engine.Start,
// it returns initialization errors instead of terminating the whole process.
func (e *Engine) Start(config Config) (result error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.stack != nil || e.device != nil {
		return errors.New("tun2socks engine is already running")
	}
	fd, ownsFD, err := parseDeviceFD(config.Device)
	if err != nil {
		return err
	}
	defer func() {
		if ownsFD {
			_ = closeFD(fd)
		}
	}()
	if config.MTU == 0 {
		config.MTU = 1500
	}
	if config.MTU < 1280 || config.MTU > 9000 {
		return fmt.Errorf("invalid MTU: %d", config.MTU)
	}

	proxy, err := parseProxy(config.ProxyURL)
	if err != nil {
		return err
	}
	dialer.Reset()
	if config.Interface != "" {
		iface, err := net.InterfaceByName(config.Interface)
		if err != nil {
			return fmt.Errorf("cannot bind to interface %s: %w", config.Interface, err)
		}
		dialer.RegisterSockOpt(dialer.WithBindToInterface(iface))
	}
	tunnel.T().SetProxy(proxy)

	dev, err := openDevice(config.Device, uint32(config.MTU))
	if err != nil {
		return err
	}
	// fdbased.Device now owns the descriptor and closes it in Device.Close.
	ownsFD = false
	networkStack, err := core.CreateStack(&core.Config{
		LinkEndpoint:     dev,
		TransportHandler: tunnel.T(),
	})
	if err != nil {
		dev.Close()
		return fmt.Errorf("cannot create tun2socks network stack: %w", err)
	}

	logger, err := upstreamlog.NewLeveled(upstreamlog.SilentLevel)
	if err == nil {
		upstreamlog.SetLogger(logger)
	}
	e.device = dev
	e.stack = networkStack
	return nil
}

func parseDeviceFD(raw string) (int, bool, error) {
	if !strings.HasPrefix(raw, "fd://") {
		return -1, false, nil
	}
	value := strings.TrimPrefix(raw, "fd://")
	fd, err := strconv.Atoi(value)
	if err != nil || fd < 0 {
		return -1, false, fmt.Errorf("invalid TUN file descriptor %q", value)
	}
	return fd, true, nil
}

// Stop closes the embedded stack and its TUN file descriptor.
func (e *Engine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.device != nil {
		e.device.Close()
		e.device = nil
	}
	if e.stack != nil {
		e.stack.Close()
		e.stack.Wait()
		e.stack = nil
	}
	dialer.Reset()
}

func openDevice(raw string, mtu uint32) (device.Device, error) {
	if strings.HasPrefix(raw, "fd://") {
		fd := strings.TrimPrefix(raw, "fd://")
		return fdbased.Open(fd, mtu, 0)
	}
	if raw == "" {
		return nil, errors.New("TUN device name is empty")
	}
	dev, err := tun.Open(raw, mtu)
	if err != nil {
		return nil, fmt.Errorf("cannot open TUN device %s: %w", raw, err)
	}
	return dev, nil
}

func parseProxy(raw string) (upstreamproxy.Proxy, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid SOCKS URL: %w", err)
	}
	scheme := strings.ToLower(u.Scheme)
	switch scheme {
	case "socks4a":
		scheme = "socks4"
	case "socks5h":
		scheme = "socks5"
	}
	if scheme != "socks4" && scheme != "socks5" {
		return nil, fmt.Errorf("unsupported proxy scheme %q", u.Scheme)
	}
	if u.Hostname() == "" || u.Port() == "" {
		return nil, errors.New("SOCKS URL must contain host and port")
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil || port < 1 || port > 65535 {
		return nil, errors.New("SOCKS URL must contain a valid numeric port")
	}
	username, password := "", ""
	if u.User != nil {
		username = u.User.Username()
		password, _ = u.User.Password()
	}
	address := net.JoinHostPort(u.Hostname(), u.Port())
	if scheme == "socks4" {
		if password != "" {
			return nil, errors.New("SOCKS4 supports a user ID but not a password")
		}
		proxy, err := socks4.New(address, username)
		if err != nil {
			return nil, fmt.Errorf("cannot create SOCKS4 proxy: %w", err)
		}
		return proxy, nil
	}
	proxy, err := socks5.New(address, username, password)
	if err != nil {
		return nil, fmt.Errorf("cannot create SOCKS5 proxy: %w", err)
	}
	return proxy, nil
}
