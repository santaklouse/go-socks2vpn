package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/santaklouse/go-socks2vpn/client"
	"github.com/santaklouse/go-socks2vpn/internal/deeplink"
	"github.com/santaklouse/go-socks2vpn/internal/elevated"
	proxyconfig "github.com/santaklouse/go-socks2vpn/internal/proxy"
)

func main() {
	importedSettings, hasDeepLink, deepLinkErr := settingsFromArgs(os.Args[1:])
	a := app.NewWithID("com.santaklouse.gosocks2vpn")
	w := a.NewWindow("go-socks2vpn")
	if !elevated.IsAdministrator() {
		showPrivilegeAlert(a, w, runtime.GOOS)
		return
	}
	w.Resize(fyne.NewSize(520, 520))

	prefs := a.Preferences()
	protocol := widget.NewSelect([]string{"SOCKS5", "SOCKS4"}, nil)
	protocol.SetSelected(prefs.StringWithFallback("protocol", "SOCKS5"))
	host := widget.NewEntry()
	host.SetPlaceHolder("proxy.example.com or 2001:db8::1")
	host.SetText(prefs.StringWithFallback("host", ""))
	port := widget.NewEntry()
	port.SetText(prefs.StringWithFallback("port", "1080"))
	username := widget.NewEntry()
	username.SetPlaceHolder("optional")
	username.SetText(prefs.StringWithFallback("username", ""))
	password := widget.NewPasswordEntry()
	password.SetPlaceHolder("not saved")
	protocol.OnChanged = func(value string) {
		if value == "SOCKS4" {
			password.SetText("")
			password.Disable()
		} else {
			password.Enable()
		}
	}
	protocol.OnChanged(protocol.Selected)
	if deepLinkErr == nil && hasDeepLink {
		protocol.SetSelected(strings.ToUpper(importedSettings.Scheme))
		host.SetText(importedSettings.Host)
		port.SetText(strconv.Itoa(importedSettings.Port))
		username.SetText(importedSettings.Username)
		password.SetText(importedSettings.Password)
	}

	initialStatus := "Disconnected"
	if deepLinkErr != nil {
		initialStatus = "Invalid configuration link: " + deepLinkErr.Error()
	} else if hasDeepLink {
		initialStatus = "Configuration imported from link. Review it and press Connect."
	}
	status := widget.NewLabel(initialStatus)
	status.TextStyle = fyne.TextStyle{Bold: true}
	logs := widget.NewMultiLineEntry()
	logs.Disable()
	logs.SetMinRowsVisible(10)

	var mu sync.Mutex
	var cancel context.CancelFunc
	var runDone <-chan struct{}
	var connectButton, disconnectButton *widget.Button
	setConnected := func(running bool, message string) {
		fyne.Do(func() {
			status.SetText(message)
			if running {
				connectButton.Disable()
				disconnectButton.Enable()
			} else {
				connectButton.Enable()
				disconnectButton.Disable()
			}
		})
	}
	logger := log.New(&guiWriter{entry: logs}, "", log.LstdFlags)

	disconnectButton = widget.NewButton("Disconnect", func() {
		mu.Lock()
		if cancel != nil {
			cancel()
		}
		mu.Unlock()
	})
	disconnectButton.Importance = widget.DangerImportance
	disconnectButton.Disable()

	connectButton = widget.NewButton("Connect", func() {
		proxyURL, err := makeProxyURL(protocol.Selected, host.Text, port.Text, username.Text, password.Text)
		if err != nil {
			setConnected(false, "Error: "+err.Error())
			return
		}
		prefs.SetString("host", strings.TrimSpace(host.Text))
		prefs.SetString("port", strings.TrimSpace(port.Text))
		prefs.SetString("username", username.Text)
		prefs.SetString("protocol", protocol.Selected)
		logs.SetText("")
		ctx, stop := context.WithCancel(context.Background())
		done := make(chan struct{})
		mu.Lock()
		cancel = stop
		runDone = done
		mu.Unlock()
		setConnected(true, "Connecting…")
		go func() {
			defer close(done)
			err := client.Run(ctx, client.Options{Proxy: proxyURL, DNS: "8.8.8.8", Log: logger})
			mu.Lock()
			cancel = nil
			runDone = nil
			mu.Unlock()
			if err != nil {
				logger.Printf("Error: %v", err)
				setConnected(false, "Connection failed")
				return
			}
			setConnected(false, "Disconnected")
		}()
	})
	connectButton.Importance = widget.HighImportance

	form := widget.NewForm(
		widget.NewFormItem("Protocol", protocol),
		widget.NewFormItem("Server", host),
		widget.NewFormItem("Port", port),
		widget.NewFormItem("Username", username),
		widget.NewFormItem("Password", password),
	)
	buttons := container.New(layout.NewGridLayout(2), connectButton, disconnectButton)
	help := widget.NewLabel("Run the application with administrator/root privileges to change system routes.")
	help.Wrapping = fyne.TextWrapWord
	w.SetContent(container.NewBorder(
		container.NewVBox(widget.NewLabelWithStyle("SOCKS4/SOCKS5 → system VPN", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), form, buttons, status, help),
		nil, nil, nil,
		container.NewScroll(logs),
	))
	w.SetCloseIntercept(func() {
		mu.Lock()
		stop := cancel
		done := runDone
		mu.Unlock()
		if stop == nil {
			w.SetCloseIntercept(nil)
			w.Close()
			return
		}
		stop()
		setConnected(true, "Disconnecting…")
		go func() {
			if done != nil {
				<-done
			}
			fyne.Do(func() {
				w.SetCloseIntercept(nil)
				w.Close()
			})
		}()
	})
	w.ShowAndRun()
}

func settingsFromArgs(args []string) (proxyconfig.Settings, bool, error) {
	for index := 0; index < len(args); index++ {
		argument := args[index]
		if argument == "--deep-link" {
			if index+1 >= len(args) {
				return proxyconfig.Settings{}, true, fmt.Errorf("--deep-link requires a URL")
			}
			settings, err := deeplink.Parse(args[index+1])
			return settings, true, err
		}
		if strings.HasPrefix(argument, "--deep-link=") {
			settings, err := deeplink.Parse(strings.TrimPrefix(argument, "--deep-link="))
			return settings, true, err
		}
		lower := strings.ToLower(argument)
		if strings.HasPrefix(lower, deeplink.Scheme+"://") || strings.HasPrefix(lower, deeplink.LegacyScheme+"://") {
			settings, err := deeplink.Parse(argument)
			return settings, true, err
		}
	}
	return proxyconfig.Settings{}, false, nil
}

func showPrivilegeAlert(a fyne.App, w fyne.Window, goos string) {
	message := privilegeMessage(goos)
	label := widget.NewLabel(message)
	label.Wrapping = fyne.TextWrapWord
	w.Resize(fyne.NewSize(480, 180))
	w.SetTitle("Administrator privileges required")
	w.SetContent(container.NewCenter(widget.NewLabel("Startup stopped: insufficient privileges.")))
	w.SetCloseIntercept(a.Quit)
	w.Show()

	closeButton := widget.NewButton("Close", a.Quit)
	alert := widget.NewModalPopUp(container.NewVBox(label, closeButton), w.Canvas())
	alert.Show()
	a.Run()
}

func privilegeMessage(goos string) string {
	switch goos {
	case "windows":
		return "The GUI was not started. Open go-socks2vpn using Run as administrator."
	case "darwin", "linux":
		return "The GUI was not started. Run the application from a terminal with: sudo -E socks2vpn-gui"
	default:
		return "The GUI was not started. Administrator privileges are required to change system routes."
	}
}

func makeProxyURL(protocol, hostText, portText, username, password string) (string, error) {
	host := strings.TrimSpace(hostText)
	if host == "" || strings.ContainsAny(host, " /\\\t\r\n") {
		return "", fmt.Errorf("enter a valid server address")
	}
	port, err := strconv.Atoi(strings.TrimSpace(portText))
	if err != nil || port < 1 || port > 65535 {
		return "", fmt.Errorf("port must be a number from 1 to 65535")
	}
	if username == "" && password != "" {
		return "", fmt.Errorf("a password was provided without a username")
	}
	scheme := proxyconfig.SchemeSOCKS5
	if protocol == "SOCKS4" {
		scheme = proxyconfig.SchemeSOCKS4
	}
	settings, err := proxyconfig.NewForScheme(scheme, strings.Trim(host, "[]"), port, username, password)
	if err != nil {
		return "", err
	}
	return settings.URL(), nil
}

type guiWriter struct {
	mu    sync.Mutex
	entry *widget.Entry
}

func (w *guiWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	text := string(data)
	fyne.Do(func() {
		const maxLogBytes = 64 << 10
		current := w.entry.Text + text
		if len(current) > maxLogBytes {
			current = current[len(current)-maxLogBytes:]
		}
		w.entry.SetText(current)
		w.entry.CursorRow = len(strings.Split(current, "\n")) - 1
		w.entry.Refresh()
	})
	return len(data), nil
}
