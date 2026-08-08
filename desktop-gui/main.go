package main

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
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
	w.Resize(fyne.NewSize(620, 650))

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
	connectionLamp := canvas.NewCircle(disconnectedColor)
	connectionLamp.StrokeColor = color.NRGBA{R: 0x55, G: 0x55, B: 0x55, A: 0xff}
	connectionLamp.StrokeWidth = 1
	statusRow := container.NewHBox(
		container.NewGridWrap(fyne.NewSize(16, 16), connectionLamp),
		status,
	)
	logs := widget.NewTextGrid()
	logs.Scroll = fyne.ScrollBoth
	logBackground := canvas.NewRectangle(logBackgroundColor)
	logBackground.CornerRadius = 5
	logView := container.NewStack(logBackground, logs)
	logWriter := newGUIWriter(logs)
	traffic := newTrafficPanel()

	var mu sync.Mutex
	var cancel context.CancelFunc
	var runDone <-chan struct{}
	var connectButton, disconnectButton *widget.Button
	setConnectionState := func(state connectionState, message string) {
		fyne.Do(func() {
			status.SetText(message)
			connectionLamp.FillColor = state.color()
			connectionLamp.Refresh()
			switch state {
			case stateConnecting, stateConnected:
				connectButton.Disable()
				disconnectButton.Enable()
			case stateDisconnecting:
				connectButton.Disable()
				disconnectButton.Disable()
			default:
				connectButton.Enable()
				disconnectButton.Disable()
			}
		})
	}
	logger := log.New(logWriter, "", log.LstdFlags)

	disconnectButton = widget.NewButton("Disconnect", func() {
		setConnectionState(stateDisconnecting, "Disconnecting…")
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
			setConnectionState(stateDisconnected, "Error: "+err.Error())
			return
		}
		prefs.SetString("host", strings.TrimSpace(host.Text))
		prefs.SetString("port", strings.TrimSpace(port.Text))
		prefs.SetString("username", username.Text)
		prefs.SetString("protocol", protocol.Selected)
		logWriter.Reset()
		traffic.Reset()
		ctx, stop := context.WithCancel(context.Background())
		done := make(chan struct{})
		mu.Lock()
		cancel = stop
		runDone = done
		mu.Unlock()
		setConnectionState(stateConnecting, "Connecting…")
		go func() {
			defer close(done)
			err := client.Run(ctx, client.Options{
				Proxy:      proxyURL,
				DNS:        "8.8.8.8",
				Log:        logger,
				Statistics: traffic.Update,
				OnConnected: func() {
					setConnectionState(stateConnected, "Connected")
				},
			})
			mu.Lock()
			cancel = nil
			runDone = nil
			mu.Unlock()
			if err != nil {
				if errors.Is(err, context.Canceled) {
					setConnectionState(stateDisconnected, "Disconnected")
					return
				}
				logger.Printf("Error: %s", redactSensitiveError(err))
				setConnectionState(stateDisconnected, "Connection failed")
				return
			}
			setConnectionState(stateDisconnected, "Disconnected")
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
		container.NewVBox(
			widget.NewLabelWithStyle("SOCKS4/SOCKS5 → system VPN", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			form,
			buttons,
			statusRow,
			traffic.View,
			help,
		),
		nil, nil, nil,
		logView,
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
		setConnectionState(stateDisconnecting, "Disconnecting…")
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

func redactSensitiveError(err error) string {
	if err == nil {
		return ""
	}
	message := err.Error()
	parts := strings.Fields(message)
	if len(parts) == 0 {
		return message
	}

	trimPunctuation := func(s string) (core string, suffix string) {
		core = s
		for len(core) > 0 {
			last := core[len(core)-1]
			if last == ',' || last == '.' || last == ';' || last == ':' || last == ')' || last == ']' {
				suffix = string(last) + suffix
				core = core[:len(core)-1]
				continue
			}
			break
		}
		return core, suffix
	}

	for i, part := range parts {
		core, suffix := trimPunctuation(part)
		lower := strings.ToLower(core)
		if strings.HasPrefix(lower, "socks4://") || strings.HasPrefix(lower, "socks5://") {
			if parsed, parseErr := proxyconfig.Parse(core); parseErr == nil {
				parts[i] = parsed.RedactedURL() + suffix
			}
		}
	}
	return strings.Join(parts, " ")
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

type connectionState uint8

const (
	stateDisconnected connectionState = iota
	stateConnecting
	stateConnected
	stateDisconnecting
)

var (
	disconnectedColor  = color.NRGBA{R: 0xd9, G: 0x32, B: 0x32, A: 0xff}
	connectingColor    = color.NRGBA{R: 0xf0, G: 0xa2, B: 0x02, A: 0xff}
	connectedColor     = color.NRGBA{R: 0x20, G: 0xb1, B: 0x5a, A: 0xff}
	logBackgroundColor = color.NRGBA{R: 0x18, G: 0x1b, B: 0x20, A: 0xff}
	logForegroundColor = color.NRGBA{R: 0xe8, G: 0xea, B: 0xed, A: 0xff}
)

func (s connectionState) color() color.Color {
	switch s {
	case stateConnecting, stateDisconnecting:
		return connectingColor
	case stateConnected:
		return connectedColor
	default:
		return disconnectedColor
	}
}

type trafficPanel struct {
	View          fyne.CanvasObject
	totals        *widget.Label
	downloadMeter *widget.ProgressBar
	uploadMeter   *widget.ProgressBar
	downloadRate  uint64
	uploadRate    uint64
}

func newTrafficPanel() *trafficPanel {
	panel := &trafficPanel{
		totals:        widget.NewLabel("Session traffic: ↓ Download 0 B    ↑ Upload 0 B"),
		downloadMeter: widget.NewProgressBar(),
		uploadMeter:   widget.NewProgressBar(),
	}
	panel.downloadMeter.TextFormatter = func() string {
		return "↓ " + client.FormatRate(panel.downloadRate)
	}
	panel.uploadMeter.TextFormatter = func() string {
		return "↑ " + client.FormatRate(panel.uploadRate)
	}
	panel.View = container.NewVBox(
		panel.totals,
		widget.NewLabel("Current speed"),
		container.NewGridWithColumns(2, panel.downloadMeter, panel.uploadMeter),
	)
	panel.apply(client.Statistics{})
	return panel
}

func (p *trafficPanel) Reset() {
	p.apply(client.Statistics{})
}

func (p *trafficPanel) Update(value client.Statistics) {
	fyne.Do(func() { p.apply(value) })
}

func (p *trafficPanel) apply(value client.Statistics) {
	p.downloadRate = value.DownloadBytesPerSecond
	p.uploadRate = value.UploadBytesPerSecond
	p.totals.SetText(fmt.Sprintf(
		"Session traffic: ↓ Download %s    ↑ Upload %s",
		client.FormatBytes(value.DownloadedBytes),
		client.FormatBytes(value.UploadedBytes),
	))
	scale := speedMeterScale(max(value.DownloadBytesPerSecond, value.UploadBytesPerSecond))
	p.downloadMeter.Max = float64(scale)
	p.uploadMeter.Max = float64(scale)
	p.downloadMeter.SetValue(float64(value.DownloadBytesPerSecond))
	p.uploadMeter.SetValue(float64(value.UploadBytesPerSecond))
}

func speedMeterScale(value uint64) uint64 {
	scale := uint64(1 << 10)
	for scale < value && scale <= ^uint64(0)/8 {
		scale *= 8
	}
	return scale
}

type guiWriter struct {
	mu     sync.Mutex
	grid   *widget.TextGrid
	buffer string
}

func newGUIWriter(grid *widget.TextGrid) *guiWriter {
	return &guiWriter{grid: grid}
}

func (w *guiWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	w.buffer += string(data)
	w.trimLocked()
	current := w.buffer
	w.mu.Unlock()
	w.render(current)
	return len(data), nil
}

func (w *guiWriter) Reset() {
	w.mu.Lock()
	w.buffer = ""
	w.mu.Unlock()
	w.render("")
}

func (w *guiWriter) trimLocked() {
	const maxLogBytes = 64 << 10
	if len(w.buffer) <= maxLogBytes {
		return
	}
	w.buffer = w.buffer[len(w.buffer)-maxLogBytes:]
	if newline := strings.IndexByte(w.buffer, '\n'); newline >= 0 {
		w.buffer = w.buffer[newline+1:]
	}
}

func (w *guiWriter) render(text string) {
	fyne.Do(func() {
		w.grid.SetText(text)
		style := &widget.CustomTextGridStyle{
			FGColor: logForegroundColor,
			BGColor: logBackgroundColor,
		}
		for row := range w.grid.Rows {
			w.grid.SetRowStyle(row, style)
		}
		w.grid.ScrollToBottom()
	})
}
