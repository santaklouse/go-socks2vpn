package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/santaklouse/go-socks2vpn/client"
	proxyconfig "github.com/santaklouse/go-socks2vpn/internal/proxy"
)

func main() {
	a := app.NewWithID("com.santaklouse.gosocks2vpn")
	w := a.NewWindow("go-socks2vpn")
	w.Resize(fyne.NewSize(520, 520))

	prefs := a.Preferences()
	protocol := widget.NewSelect([]string{"SOCKS5", "SOCKS4"}, nil)
	protocol.SetSelected(prefs.StringWithFallback("protocol", "SOCKS5"))
	host := widget.NewEntry()
	host.SetPlaceHolder("proxy.example.com или 2001:db8::1")
	host.SetText(prefs.StringWithFallback("host", ""))
	port := widget.NewEntry()
	port.SetText(prefs.StringWithFallback("port", "1080"))
	username := widget.NewEntry()
	username.SetPlaceHolder("необязательно")
	username.SetText(prefs.StringWithFallback("username", ""))
	password := widget.NewPasswordEntry()
	password.SetPlaceHolder("не сохраняется")
	protocol.OnChanged = func(value string) {
		if value == "SOCKS4" {
			password.SetText("")
			password.Disable()
		} else {
			password.Enable()
		}
	}
	protocol.OnChanged(protocol.Selected)

	status := widget.NewLabel("Отключено")
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

	disconnectButton = widget.NewButton("Отключить", func() {
		mu.Lock()
		if cancel != nil {
			cancel()
		}
		mu.Unlock()
	})
	disconnectButton.Importance = widget.DangerImportance
	disconnectButton.Disable()

	connectButton = widget.NewButton("Подключить", func() {
		proxyURL, err := makeProxyURL(protocol.Selected, host.Text, port.Text, username.Text, password.Text)
		if err != nil {
			setConnected(false, "Ошибка: "+err.Error())
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
		setConnected(true, "Подключение…")
		go func() {
			defer close(done)
			err := client.Run(ctx, client.Options{Proxy: proxyURL, DNS: "8.8.8.8", Log: logger})
			mu.Lock()
			cancel = nil
			runDone = nil
			mu.Unlock()
			if err != nil {
				logger.Printf("Ошибка: %v", err)
				setConnected(false, "Ошибка подключения")
				return
			}
			setConnected(false, "Отключено")
		}()
	})
	connectButton.Importance = widget.HighImportance

	form := widget.NewForm(
		widget.NewFormItem("Протокол", protocol),
		widget.NewFormItem("Сервер", host),
		widget.NewFormItem("Порт", port),
		widget.NewFormItem("Пользователь", username),
		widget.NewFormItem("Пароль", password),
	)
	buttons := container.New(layout.NewGridLayout(2), connectButton, disconnectButton)
	help := widget.NewLabel("Для изменения системных маршрутов приложение нужно запустить с правами администратора/root.")
	help.Wrapping = fyne.TextWrapWord
	w.SetContent(container.NewBorder(
		container.NewVBox(widget.NewLabelWithStyle("SOCKS4/SOCKS5 → системный VPN", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), form, buttons, status, help),
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
		setConnected(true, "Отключение…")
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

func makeProxyURL(protocol, hostText, portText, username, password string) (string, error) {
	host := strings.TrimSpace(hostText)
	if host == "" || strings.ContainsAny(host, " /\\\t\r\n") {
		return "", fmt.Errorf("укажите корректный адрес сервера")
	}
	port, err := strconv.Atoi(strings.TrimSpace(portText))
	if err != nil || port < 1 || port > 65535 {
		return "", fmt.Errorf("порт должен быть числом от 1 до 65535")
	}
	if username == "" && password != "" {
		return "", fmt.Errorf("пароль задан без пользователя")
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
