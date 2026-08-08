# go-socks2vpn

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

Проверить план без загрузки и изменения сети:

```bash
./bin/socks2vpn --dry-run --proxy '192.0.2.10:1080'
```

Проверить SOCKS handshake без root и без создания VPN:

```bash
./bin/socks2vpn --proxy 'socks4://192.168.192.100:9050' --check-proxy --check-target '1.1.1.1:443'
```

Отдельный бинарник `tun2socks` не загружается и не запускается: сетевой стек компилируется прямо в `socks2vpn`. На Windows при первом запуске автоматически загружается только официальный нативный компонент Wintun 0.14.1; закреплённые SHA-256 проверяют и архив, и DLL нужной архитектуры, включая последующие запуски из кэша. На macOS и Linux во время работы ничего не скачивается.

## Desktop GUI

```bash
make gui
sudo ./bin/socks2vpn-gui
```

На Windows запускайте `socks2vpn-gui.exe` через «Запуск от имени администратора». На Linux можно использовать `sudo -E`; на macOS — `sudo` из Terminal. GUI сохраняет сервер, порт и имя пользователя, но никогда не сохраняет пароль.

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

Android-приложение поддерживает IPv4 и IPv6, задаёт DNS `1.1.1.1`, выполняет TCP preflight прокси и работает как foreground VPN service. Пароль живёт только в памяти процесса до отключения.

## Docker

Образ собирается для `linux/amd64` и `linux/arm64`. Внутри контейнера нужны capability `NET_ADMIN` и устройство `/dev/net/tun`; готовый `compose.yaml` уже задаёт их. По умолчанию используется предоставленный SOCKS4-сервер `192.168.192.100:9050`.

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
- Desktop-маршруты меняются глобально и требуют повышенных прав. Процесс нельзя принудительно убивать через `kill -9`, иначе он не успеет выполнить откат. После обычного Ctrl+C откат автоматический.
- На всех платформах используется один встроенный Go-движок из пакета `engine`. Android передаёт ему файловый дескриптор от `VpnService` через тонкую `gomobile`-обёртку.
