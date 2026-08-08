# go-socks2vpn

[English](README.md) | Русский

[Сайт](https://santaklouse.github.io/go-socks2vpn/) · [Последний релиз](https://github.com/santaklouse/go-socks2vpn/releases/latest)

[![CI](https://github.com/santaklouse/go-socks2vpn/actions/workflows/build.yml/badge.svg?branch=main)](https://github.com/santaklouse/go-socks2vpn/actions/workflows/build.yml)
[![Latest tag](https://img.shields.io/github/v/tag/santaklouse/go-socks2vpn)](https://github.com/santaklouse/go-socks2vpn/tags)
[![Go version](https://img.shields.io/github/go-mod/go-version/santaklouse/go-socks2vpn)](https://github.com/santaklouse/go-socks2vpn/blob/main/go.mod)

Кроссплатформенное приложение на Go, которое превращает удалённый SOCKS4- или SOCKS5-прокси в системный VPN на macOS, Linux, Windows, Android и внутри Linux-контейнера.

В проект входят:

- `cmd/socks2vpn` — CLI для macOS, Linux и Windows;
- `desktop-gui` — минимальный Fyne GUI для macOS, Linux и Windows;
- `android` — Android-приложение на `VpnService` без root;
- `Dockerfile` и `compose.yaml` — multi-arch контейнер и sidecar-сценарий;
- официальный `xjasonlyu/tun2socks/v2` встроен как обычная Go-зависимость;
- SHA-256 проверка Windows-драйвера Wintun и полный откат сетевых изменений.

## Установка CLI

Требуется Go 1.26.3 или новее — это минимальная версия текущего `tun2socks` v2.7.0.

Установить последнюю опубликованную версию:

```bash
go install github.com/santaklouse/go-socks2vpn/cmd/socks2vpn@latest
```

Go устанавливает бинарник в каталог, который возвращает `go env GOBIN`, либо в `$(go env GOPATH)/bin`, если `GOBIN` не задан. Пример запуска на macOS и Linux:

```bash
sudo "$(go env GOPATH)/bin/socks2vpn" --proxy 'socks4://192.168.192.100:9050'
```

В Windows откройте PowerShell от имени администратора:

```powershell
& "$(go env GOPATH)\bin\socks2vpn.exe" --proxy "socks4://192.168.192.100:9050"
```

Собрать CLI из локальной копии исходного кода:

```bash
go build -o bin/socks2vpn ./cmd/socks2vpn
sudo ./bin/socks2vpn --proxy '192.0.2.10:1080:alice:correct-horse-battery-staple'
```

В Windows откройте PowerShell от имени администратора:

```powershell
.\socks2vpn.exe --proxy "192.0.2.10:1080:alice:correct-horse-battery-staple"
```

Без `--proxy` программа запросит эту строку интерактивно. Поддерживаются также URL и IPv6:

```bash
sudo ./bin/socks2vpn --proxy 'socks5://alice:correct-horse-battery-staple@192.0.2.10:1080'
sudo ./bin/socks2vpn --proxy '[2001:db8::10]:1080:alice:correct-horse-battery-staple'
sudo ./bin/socks2vpn --proxy 'socks4://192.168.192.100:9050'
```

Добавьте `--stats`, чтобы раз в секунду выводить суммарный входящий/исходящий трафик сеанса и текущую скорость:

```bash
sudo ./bin/socks2vpn --stats --proxy 'socks5://alice:correct-horse-battery-staple@192.0.2.10:1080'
```

Счётчики измеряют IP-байты, прошедшие через TUN за текущий сеанс. `download` — данные, доставленные из VPN приложениям, `upload` — данные, отправленные приложениями в VPN. SOCKS framing и транспортный overhead вне TUN в эти значения не входят.

Проверить план без загрузки и изменения сети:

```bash
./bin/socks2vpn --dry-run --proxy '192.0.2.10:1080'
```

Проверить SOCKS handshake без root и без создания VPN:

```bash
./bin/socks2vpn --proxy 'socks4://192.168.192.100:9050' --check-proxy --check-target '1.1.1.1:443'
```

Отдельный бинарник `tun2socks` не загружается и не запускается: сетевой стек компилируется прямо в `socks2vpn`. На Windows при первом запуске автоматически загружается только официальный нативный компонент Wintun 0.14.1; закреплённые SHA-256 проверяют и архив, и DLL нужной архитектуры, включая последующие запуски из кэша. На macOS и Linux во время работы ничего не скачивается.

До установки full-tunnel маршрутов клиент разрешает адрес SOCKS-сервера и определяет интерфейс, через который он сейчас доступен. Встроенный движок привязывает исходящие соединения с прокси к этому интерфейсу, поэтому соединение с самим SOCKS-сервером не попадает в TUN, даже если сервер доступен через отдельный overlay-, tunnel- или локальный сетевой интерфейс.

## Desktop GUI

Установка последнего релиза одной командой на macOS и Linux:

```bash
curl -fsSL https://raw.githubusercontent.com/santaklouse/go-socks2vpn/main/scripts/install-gui.sh | sh
```

Установщик определяет ОС и архитектуру, скачивает подходящий GUI-архив, проверяет его по `SHA256SUMS` и устанавливает бинарник в `/usr/local/bin`. На macOS он также устанавливает поддерживающий URL-схемы bundle `go-socks2vpn.app` в `/Applications`. В Linux он устанавливает недостающие OpenGL, Wayland, XKB, X11, PolicyKit и `xdg-utils` runtime-зависимости и регистрирует desktop URL handlers.

После установки запустите GUI с правами администратора:

```bash
sudo -E socks2vpn-gui
```

Установка в Windows из PowerShell:

```powershell
irm https://raw.githubusercontent.com/santaklouse/go-socks2vpn/main/scripts/install-gui.ps1 | iex
```

Windows-установщик проверяет SHA-256, устанавливает программу в `%LOCALAPPDATA%\Programs\go-socks2vpn`, добавляет её в пользовательский `PATH`, создаёт ярлык `go-socks2vpn` в меню «Пуск» и регистрирует handlers ссылок конфигурации. Ярлык и ссылки автоматически показывают стандартный UAC-запрос. Windows ARM64 использует amd64-сборку через системную x64-эмуляцию.

Поддерживаются готовые GUI-релизы для macOS amd64/arm64, Linux amd64 и Windows amd64. Для установки конкретной версии на macOS или Linux задайте, например, `GO_SOCKS2VPN_VERSION=v1.0.0`; в Windows скачайте скрипт и запустите его с параметром `-Version v1.0.0`.

Ручная сборка из исходного кода:

```bash
make gui
sudo ./bin/socks2vpn-gui
```

На Windows запускайте `socks2vpn-gui.exe` через «Запуск от имени администратора». На Linux можно использовать `sudo -E`; на macOS — `sudo` из Terminal. GUI проверяет права до инициализации VPN: без root/Administrator основное окно не открывается, показывается модальный alert с правильной командой запуска, после его закрытия приложение завершается. GUI сохраняет сервер, порт и имя пользователя, но никогда не сохраняет пароль.

Desktop GUI всегда показывает лампу состояния: красная означает отключение или ошибку, жёлтая — подключение или отключение в процессе, зелёная — активное соединение. Суммарный download/upload за сеанс и две шкалы текущей скорости включены по умолчанию. У журнала фиксированный тёмный фон и контрастный светлый шрифт независимо от системной темы.

## Ссылки конфигурации

Desktop- и Android-приложения принимают основную схему `socks2vpn://` и совместимый alias `socks2vps://`. Открытие ссылки заполняет форму, но никогда не подключает VPN автоматически:

```text
socks2vpn://socks5-proxyuser:proxypass@proxyhost:1080
socks2vpn://socks4-proxyuser@proxyhost:9050
```

Префикс протокола обязателен. Если у прокси нет имени пользователя, укажите `socks5` или `socks4` без дефиса. Зарезервированные символы в реквизитах нужно percent-encode, например `user%40example` и `p%40ss%3Aword`. Импортированный пароль остаётся только в памяти, никогда не сохраняется и не записывается в логи.

Проверить зарегистрированный desktop handler после установки GUI:

```bash
# macOS
open 'socks2vpn://socks5-proxyuser:proxypass@proxyhost:1080'

# Linux
xdg-open 'socks2vpn://socks5-proxyuser:proxypass@proxyhost:1080'
```

```powershell
# Windows
Start-Process 'socks2vpn://socks5-proxyuser:proxypass@proxyhost:1080'
```

Ссылки с паролями могут оставаться в истории браузера или мессенджера, буфере обмена и метаданных операционной системы. Если ссылка проходит через недоверенное приложение, лучше не включать в неё пароль.

## Android

Android использует системный `VpnService`, поэтому root не нужен. Первый запуск показывает стандартный Android-диалог разрешения VPN. Собственный UID приложения исключён из TUN-маршрута, чтобы соединение Go-движка с SOCKS-сервером не попало обратно в VPN.

Требования для локальной сборки:

- JDK 17;
- Android SDK Platform 36 и Build Tools 36.0.0;
- Android NDK `28.2.13676358`;
- Go 1.26.3 для upstream tun2socks `v2.7.0`.

```bash
cd android
./gradlew assembleDebug
```

Скрипт Gradle сам собирает общий Go-движок в `app/libs/tun2socks.aar` через `gomobile`; это библиотека внутри APK, не отдельный процесс. Готовый APK находится в `android/app/build/outputs/apk/debug/app-debug.apk`.

Android-приложение создаёт IPv4-only VPN и работает как foreground VPN service. IPv6-адрес, маршрут и DNS-сервер не добавляются, поэтому Android блокирует семейство IPv6, а не выпускает его через основную сеть в обход VPN. До установки full-tunnel маршрута приложение выполняет через указанный прокси полноценный SOCKS handshake с авторизацией, поэтому доступный порт с неверными реквизитами приводит к явной ошибке, а не к ложному состоянию Connected. DNS-запросы передаются через тот же SOCKS-прокси как RFC 8484 DNS-over-HTTPS по TCP на `1.1.1.1:443`, а AAAA-запросы получают NODATA, поскольку многие SOCKS-серверы отклоняют IPv6-назначения. Android также отклоняет QUIC на UDP-порту 443 ответом ICMP, поэтому браузеры сразу переключаются на HTTPS поверх TCP. Остальной UDP-трафик в режиме SOCKS5 остаётся включённым. В GUI есть такая же лампа состояния, суммарный трафик сеанса и живые шкалы download/upload. Пароль живёт только в памяти процесса до отключения.

## Docker

Образ собирается для `linux/amd64` и `linux/arm64`. Внутри контейнера нужны capability `NET_ADMIN` и устройство `/dev/net/tun`; готовый `compose.yaml` уже задаёт их. По умолчанию используется предоставленный SOCKS4-сервер `192.168.192.100:9050`.

Запустить опубликованный образ напрямую (для остановки нажмите `Ctrl+C`):

```bash
docker run --rm --init --name socks2vpn --cap-add=NET_ADMIN --device=/dev/net/tun:/dev/net/tun ghcr.io/santaklouse/go-socks2vpn:latest --proxy='socks4://192.168.192.100:9050' --interface=eth0
```

Эта команда меняет маршруты только внутри контейнера. Чтобы направить через VPN трафик другого контейнера, подключите его через `--network=container:socks2vpn` либо используйте описанную ниже sidecar-конфигурацию Compose.

Проверить SOCKS4 handshake тем же кодом, который использует TUN-движок, без изменения маршрутов:

```bash
make docker-check
```

Запустить VPN внутри отдельного container network namespace:

```bash
make docker-up
```

Проверить TCP-трафик через созданный `tun0`:

```bash
make docker-traffic-check
```

Другие контейнеры можно подключать как sidecar через `network_mode: service:socks2vpn`. Маршруты меняются только внутри этого network namespace: Docker-контейнер не превращает сетевой стек хоста macOS или Windows в VPN — для этого используйте нативный GUI.

Сервер `192.168.192.100:9050` проверен с SOCKS4A и SOCKS5. Чтобы получить UDP и DNS через прокси, переключите Compose на SOCKS5:

```bash
SOCKS_PROXY=socks5://192.168.192.100:9050 docker compose up --build socks2vpn
```

SOCKS4 не имеет UDP relay и не поддерживает IPv6 назначения. В режиме `socks4://` TCP к IPv4 работает, но UDP-приложения, IPv6-соединения и DNS-запросы, попавшие в TUN, не смогут пройти через прокси.

## Проверки и сборка

```bash
make test
./scripts/build-all.sh
./scripts/build-gui.sh
```

`scripts/build-all.sh` создаёт CLI для macOS amd64/arm64, Linux 386/amd64/arm/arm64/riscv64 и Windows 386/amd64/arm64. GUI собирается нативно на каждой desktop-системе. GitHub Actions отдельно проверяет unit tests, все CLI targets, четыре GUI runner и Android APK.

## GitHub Actions и релизы

Workflow `.github/workflows/build.yml` запускает unit tests, race detector, `go vet`, все CLI-сборки, четыре нативные GUI-сборки, Android APK и multi-arch Docker build при каждом push и pull request. macOS GUI собирается отдельно для Apple Silicon и Intel.

Push семантического тега автоматически создаёт GitHub Release с архивами программ, подписанным Android APK, автоматически сгенерированным описанием и файлом `SHA256SUMS`:

```bash
git tag v1.0.0
git push origin v1.0.0
```

Теги вида `v1.0.0-rc.1` публикуются как prerelease. Release создаётся только после успешного завершения всех тестов и всех платформенных сборок. Дополнительные токены или сторонние release actions не нужны: публикация выполняется официальным `GITHUB_TOKEN` через установленный на runner GitHub CLI.

Тот же тег публикует образы `linux/amd64` и `linux/arm64` в GitHub Container Registry:

```bash
docker pull ghcr.io/santaklouse/go-socks2vpn:v1.0.0
```

Для подписанного Android APK один раз добавьте в настройках репозитория четыре GitHub Actions secret:

- `ANDROID_KEYSTORE_BASE64` — release-keystore целиком в Base64;
- `ANDROID_KEYSTORE_PASSWORD` — пароль keystore;
- `ANDROID_KEY_ALIAS` — alias ключа;
- `ANDROID_KEY_PASSWORD` — пароль ключа.

Создать release-keystore можно интерактивно — пароли не попадут в историю shell:

```bash
keytool -genkeypair -v -keystore go-socks2vpn-release.jks -alias go-socks2vpn -keyalg RSA -keysize 3072 -validity 10000
```

Получить значение `ANDROID_KEYSTORE_BASE64`:

```bash
# macOS
base64 -i go-socks2vpn-release.jks | tr -d '\n'

# Linux
base64 -w 0 go-socks2vpn-release.jks
```

Сам файл `go-socks2vpn-release.jks` и его пароли храните отдельно от репозитория: потеря ключа лишит будущие APK возможности обновлять уже установленное приложение.

При push/PR workflow собирает устанавливаемый debug APK. Для тега он требует эти четыре секрета, подставляет версию из тега, использует монотонный номер workflow как Android `versionCode` и публикует подписанный `go-socks2vpn-android.apk`. Без ключа теговый pipeline завершится ошибкой до создания неполного GitHub Release.

## Важные ограничения

- Нужен SOCKS-сервер с поддержкой нужных приложению протоколов. SOCKS4 передаёт только TCP; для UDP и DNS через прокси используйте SOCKS5 с UDP ASSOCIATE.
- SOCKS не передаёт ICMP. Пользовательский сетевой стек может отвечать на `ping` локально, поэтому задержка меньше миллисекунды и TTL 64 не означают, что пакеты дошли до удалённого узла. Проверяйте туннель TCP-трафиком, например `curl https://icanhazip.com`.
- Desktop-маршруты меняются глобально и требуют повышенных прав. Процесс нельзя принудительно убивать через `kill -9`, иначе он не успеет выполнить откат. После обычного Ctrl+C откат автоматический.
- На всех платформах используется один встроенный Go-движок из пакета `engine`. Android передаёт ему файловый дескриптор от `VpnService` через тонкую `gomobile`-обёртку.
