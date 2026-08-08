package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/santaklouse/go-socks2vpn/client"
	tunengine "github.com/santaklouse/go-socks2vpn/engine"
)

var version = "dev"

func main() {
	os.Exit(run())
}

func run() int {
	proxy := flag.String("proxy", "", "SOCKS proxy: socks4://, socks5://, or host:port[:user:password]")
	cacheDir := flag.String("cache-dir", "", "directory for Wintun on Windows")
	interfaceName := flag.String("interface", "", "primary network interface (usually detected automatically)")
	gateway := flag.String("gateway", "", "primary IPv4 gateway (usually detected automatically)")
	dns := flag.String("dns", "8.8.8.8", "DNS server for the Windows TUN interface")
	dryRun := flag.Bool("dry-run", false, "show the plan without changing the network or downloading files")
	showStats := flag.Bool("stats", false, "print session traffic totals and current transfer rates once per second")
	checkProxy := flag.Bool("check-proxy", false, "test the proxy TCP handshake without creating a VPN")
	checkTarget := flag.String("check-target", "1.1.1.1:443", "TCP address used by --check-proxy")
	showVersion := flag.Bool("version", false, "show the version")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options]\n\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return 0
	}
	fmt.Println("=============================================")
	fmt.Println("  go-socks2vpn — SOCKS4/SOCKS5 as a system VPN")
	fmt.Println("=============================================")
	if *proxy == "" {
		var err error
		*proxy, err = client.ReadProxy(os.Stdin, os.Stdout)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			return 2
		}
	}
	if *checkProxy {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := tunengine.CheckProxy(ctx, *proxy, *checkTarget); err != nil {
			fmt.Fprintln(os.Stderr, "Proxy check failed:", err)
			return 1
		}
		fmt.Printf("Proxy is working: TCP handshake to %s succeeded\n", *checkTarget)
		return 0
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	logger := log.New(os.Stdout, "", log.LstdFlags)
	var statistics func(client.Statistics)
	if *showStats {
		statistics = func(value client.Statistics) {
			logger.Printf(
				"Traffic: download=%s (%s), upload=%s (%s)",
				client.FormatBytes(value.DownloadedBytes),
				client.FormatRate(value.DownloadBytesPerSecond),
				client.FormatBytes(value.UploadedBytes),
				client.FormatRate(value.UploadBytesPerSecond),
			)
		}
	}
	err := client.Run(ctx, client.Options{
		Proxy:      *proxy,
		CacheDir:   *cacheDir,
		Interface:  *interfaceName,
		Gateway:    *gateway,
		DNS:        *dns,
		DryRun:     *dryRun,
		Log:        logger,
		Statistics: statistics,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return 1
	}
	return 0
}
