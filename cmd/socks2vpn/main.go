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
	proxy := flag.String("proxy", "", "SOCKS-прокси: socks4://, socks5:// или host:port[:user:password]")
	cacheDir := flag.String("cache-dir", "", "каталог для Wintun на Windows")
	interfaceName := flag.String("interface", "", "основной сетевой интерфейс (обычно определяется автоматически)")
	gateway := flag.String("gateway", "", "основной IPv4-шлюз (обычно определяется автоматически)")
	dns := flag.String("dns", "8.8.8.8", "DNS-сервер для TUN-интерфейса Windows")
	dryRun := flag.Bool("dry-run", false, "показать план без изменения сети и загрузок")
	checkProxy := flag.Bool("check-proxy", false, "проверить TCP handshake прокси без создания VPN")
	checkTarget := flag.String("check-target", "1.1.1.1:443", "TCP-адрес для --check-proxy")
	showVersion := flag.Bool("version", false, "показать версию")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Использование: %s [параметры]\n\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return 0
	}
	fmt.Println("=============================================")
	fmt.Println("  go-socks2vpn — SOCKS4/SOCKS5 как системный VPN")
	fmt.Println("=============================================")
	if *proxy == "" {
		var err error
		*proxy, err = client.ReadProxy(os.Stdin, os.Stdout)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Ошибка:", err)
			return 2
		}
	}
	if *checkProxy {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := tunengine.CheckProxy(ctx, *proxy, *checkTarget); err != nil {
			fmt.Fprintln(os.Stderr, "Ошибка проверки прокси:", err)
			return 1
		}
		fmt.Printf("Прокси работает: TCP handshake к %s выполнен успешно\n", *checkTarget)
		return 0
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	logger := log.New(os.Stdout, "", log.LstdFlags)
	err := client.Run(ctx, client.Options{
		Proxy:     *proxy,
		CacheDir:  *cacheDir,
		Interface: *interfaceName,
		Gateway:   *gateway,
		DNS:       *dns,
		DryRun:    *dryRun,
		Log:       logger,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Ошибка:", err)
		return 1
	}
	return 0
}
