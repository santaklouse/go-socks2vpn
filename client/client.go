// Package client manages the desktop proxy tunnel lifecycle.
package client

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"

	tunengine "github.com/santaklouse/go-socks2vpn/engine"
	"github.com/santaklouse/go-socks2vpn/internal/assets"
	"github.com/santaklouse/go-socks2vpn/internal/command"
	"github.com/santaklouse/go-socks2vpn/internal/elevated"
	"github.com/santaklouse/go-socks2vpn/internal/network"
	"github.com/santaklouse/go-socks2vpn/internal/platform"
	proxyconfig "github.com/santaklouse/go-socks2vpn/internal/proxy"
	"github.com/santaklouse/go-socks2vpn/internal/wintun"
)

type Logger interface {
	Printf(format string, args ...any)
}

type Options struct {
	Proxy     string
	CacheDir  string
	Interface string
	Gateway   string
	DNS       string
	DryRun    bool
	Log       Logger
}

// Run validates the configuration, prepares the host network, runs tun2socks,
// and restores every successfully changed setting before returning.
func Run(ctx context.Context, options Options) error {
	logger := options.Log
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}
	proxySettings, err := proxyconfig.Parse(options.Proxy)
	if err != nil {
		return err
	}
	info := platform.Detect()
	if info.OS != "darwin" && info.OS != "linux" && info.OS != "windows" {
		return fmt.Errorf("the desktop client does not support %s; on Android, use the application from the android directory", info.OS)
	}
	logger.Printf("Platform: %s", info.Description())
	logger.Printf("SOCKS proxy: %s", proxySettings.RedactedURL())
	if proxySettings.Scheme == proxyconfig.SchemeSOCKS4 {
		logger.Printf("Warning: SOCKS4 only carries TCP to IPv4 destinations; UDP, DNS, and IPv6 destinations are not supported")
	}

	if !options.DryRun && !elevated.IsAdministrator() {
		if info.OS == "windows" {
			return errors.New("run the application as administrator to configure the VPN")
		}
		return errors.New("run the application with sudo to configure the VPN")
	}

	runner := command.ExecRunner{}
	networkInfo, detectErr := network.Detect(ctx, info.OS, runner)
	if detectErr != nil {
		if !options.DryRun && options.Interface == "" {
			return detectErr
		}
		logger.Printf("Warning: %v", detectErr)
		networkInfo = fallbackNetwork(info.OS)
	}
	if options.Interface != "" {
		networkInfo.Interface = options.Interface
	}
	if options.Gateway != "" {
		networkInfo.Gateway = options.Gateway
	}
	if networkInfo.Interface == "" {
		return errors.New("could not detect the primary network interface; specify --interface")
	}
	logger.Printf("Primary interface: %s", networkInfo.Interface)
	if networkInfo.Gateway != "" {
		logger.Printf("Primary gateway: %s", networkInfo.Gateway)
	}

	plan, err := network.BuildPlan(ctx, info.OS, networkInfo, options.DNS, runner)
	if err != nil {
		return err
	}
	logger.Printf("Embedded tun2socks: device=%s, proxy=%s, interface=%s", plan.Device, proxySettings.RedactedURL(), networkInfo.Interface)

	if options.DryRun {
		logger.Printf("Network configuration plan:")
		for _, step := range plan.Steps {
			logger.Printf("  %s", step.Do.String())
		}
		return nil
	}
	if err := prepareNativeDependencies(ctx, options, info, logger); err != nil {
		return err
	}

	session := &networkSession{runner: runner, log: logger}
	if plan.Timing == network.BeforeProcess {
		if err := session.apply(ctx, plan); err != nil {
			return err
		}
	}

	embedded := tunengine.New()
	if err := embedded.Start(tunengine.Config{
		Device:    plan.Device,
		Interface: networkInfo.Interface,
		ProxyURL:  proxySettings.URL(),
		MTU:       1500,
	}); err != nil {
		rollbackWithTimeout(session)
		return fmt.Errorf("could not start the embedded tun2socks engine: %w", err)
	}

	if plan.Timing == network.AfterProcess {
		if err := waitForInterface(ctx, plan.WaitInterface); err != nil {
			embedded.Stop()
			return err
		}
		if err := session.apply(ctx, plan); err != nil {
			embedded.Stop()
			return err
		}
	}

	logger.Printf("VPN connected. Press Ctrl+C to disconnect.")
	<-ctx.Done()
	logger.Printf("Disconnect signal received")

	rollbackWithTimeout(session)
	embedded.Stop()
	logger.Printf("Network configuration restored")
	return nil
}

func prepareNativeDependencies(ctx context.Context, options Options, info platform.Info, logger Logger) error {
	if info.OS != "windows" {
		return nil
	}
	dir := options.CacheDir
	if dir == "" {
		var err error
		dir, err = assets.DefaultCacheDir()
		if err != nil {
			return err
		}
	}
	path, err := assets.New(logger).EnsureWintun(ctx, info, dir)
	if err != nil {
		return err
	}
	return wintun.Load(path)
}

type networkSession struct {
	runner  command.Runner
	log     Logger
	applied []network.Step
}

func (s *networkSession) apply(ctx context.Context, plan network.Plan) error {
	for _, step := range plan.Steps {
		s.log.Printf("Applying: %s", step.Do.String())
		if _, err := s.runner.Output(ctx, step.Do); err != nil {
			rollbackWithTimeout(s)
			return fmt.Errorf("could not configure the network: %w", err)
		}
		s.applied = append(s.applied, step)
	}
	return nil
}

func (s *networkSession) rollback(ctx context.Context) {
	for i := len(s.applied) - 1; i >= 0; i-- {
		for _, undo := range s.applied[i].Undo {
			s.log.Printf("Rolling back: %s", undo.String())
			if _, err := s.runner.Output(ctx, undo); err != nil {
				s.log.Printf("Rollback warning: %v", err)
			}
		}
	}
	s.applied = nil
}

func rollbackWithTimeout(session *networkSession) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	session.rollback(ctx)
}

func waitForInterface(ctx context.Context, name string) error {
	deadline := time.NewTimer(12 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		if _, err := net.InterfaceByName(name); err == nil {
			return nil
		}
		select {
		case <-ticker.C:
		case <-deadline.C:
			return fmt.Errorf("TUN interface %s did not appear within 12 seconds", name)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func fallbackNetwork(goos string) network.Info {
	switch goos {
	case "darwin":
		return network.Info{Interface: "en0"}
	case "windows":
		return network.Info{Interface: "Ethernet"}
	default:
		return network.Info{Interface: "eth0"}
	}
}

// ReadProxy reads one complete proxy specification from a CLI stream.
func ReadProxy(reader io.Reader, writer io.Writer) (string, error) {
	fmt.Fprintln(writer, "Enter a SOCKS proxy (socks4://host:port, socks5:// URL, or host:port:user:password):")
	line, err := bufio.NewReader(reader).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return "", errors.New("proxy is not specified")
	}
	return line, nil
}
