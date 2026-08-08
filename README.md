# go-socks2vpn

English | [Russian](README.RU.md)

[![CI](https://github.com/santaklouse/go-socks2vpn/actions/workflows/build.yml/badge.svg?branch=main)](https://github.com/santaklouse/go-socks2vpn/actions/workflows/build.yml)
[![Latest tag](https://img.shields.io/github/v/tag/santaklouse/go-socks2vpn)](https://github.com/santaklouse/go-socks2vpn/tags)
[![Go version](https://img.shields.io/github/go-mod/go-version/santaklouse/go-socks2vpn)](https://github.com/santaklouse/go-socks2vpn/blob/main/go.mod)

A cross-platform Go application that turns a remote SOCKS4 or SOCKS5 proxy into a system VPN on macOS, Linux, Windows, Android, and inside a Linux container.

The project includes:

- `cmd/socks2vpn` — a CLI for macOS, Linux, and Windows;
- `desktop-gui` — a minimal Fyne GUI for macOS, Linux, and Windows;
- `android` — a rootless Android application based on `VpnService`;
- `Dockerfile` and `compose.yaml` — a multi-architecture container and sidecar setup;
- the official `xjasonlyu/tun2socks/v2` package embedded as a regular Go dependency;
- SHA-256 verification of the Windows Wintun driver and complete rollback of network changes.

## CLI installation

Go 1.26.3 or newer is required. This is the minimum version supported by the current `tun2socks` v2.7.0 dependency.

Install the latest published version:

```bash
go install github.com/santaklouse/go-socks2vpn/cmd/socks2vpn@latest
```

Go installs the binary into the directory returned by `go env GOBIN`, or into `$(go env GOPATH)/bin` when `GOBIN` is not set. Example for macOS and Linux:

```bash
sudo "$(go env GOPATH)/bin/socks2vpn" --proxy 'socks4://192.168.192.100:9050'
```

On Windows, open PowerShell as Administrator:

```powershell
& "$(go env GOPATH)\bin\socks2vpn.exe" --proxy "socks4://192.168.192.100:9050"
```

Build the CLI from a local source checkout:

```bash
go build -o bin/socks2vpn ./cmd/socks2vpn
sudo ./bin/socks2vpn --proxy '192.0.2.10:1080:alice:correct-horse-battery-staple'
```

On Windows, open PowerShell as Administrator:

```powershell
.\socks2vpn.exe --proxy "192.0.2.10:1080:alice:correct-horse-battery-staple"
```

Without `--proxy`, the program prompts for the proxy string interactively. URLs and IPv6 are also supported:

```bash
sudo ./bin/socks2vpn --proxy 'socks5://alice:correct-horse-battery-staple@192.0.2.10:1080'
sudo ./bin/socks2vpn --proxy '[2001:db8::10]:1080:alice:correct-horse-battery-staple'
sudo ./bin/socks2vpn --proxy 'socks4://192.168.192.100:9050'
```

Preview the plan without downloading files or changing the network:

```bash
./bin/socks2vpn --dry-run --proxy '192.0.2.10:1080'
```

Test the SOCKS handshake without root privileges or VPN creation:

```bash
./bin/socks2vpn --proxy 'socks4://192.168.192.100:9050' --check-proxy --check-target '1.1.1.1:443'
```

No separate `tun2socks` executable is downloaded or launched: the network stack is compiled directly into `socks2vpn`. On Windows, only the official native Wintun 0.14.1 component is downloaded on first launch. Pinned SHA-256 hashes verify both the archive and the architecture-specific DLL, including subsequent launches from the cache. Nothing is downloaded at runtime on macOS or Linux.

## Desktop GUI

Install the latest release with one command on macOS or Linux:

```bash
curl -fsSL https://raw.githubusercontent.com/santaklouse/go-socks2vpn/main/scripts/install-gui.sh | sh
```

The installer detects the operating system and architecture, downloads the matching GUI archive, verifies it against `SHA256SUMS`, and installs the binary into `/usr/local/bin`. On Linux, it also installs missing OpenGL, Wayland, XKB, and X11 runtime libraries through `apt`, `dnf`, `yum`, or `pacman`.

After installation, run the GUI with administrator privileges:

```bash
sudo -E socks2vpn-gui
```

Install on Windows from PowerShell:

```powershell
irm https://raw.githubusercontent.com/santaklouse/go-socks2vpn/main/scripts/install-gui.ps1 | iex
```

The Windows installer verifies SHA-256, installs the program into `%LOCALAPPDATA%\Programs\go-socks2vpn`, adds it to the user `PATH`, and creates a `go-socks2vpn` Start menu shortcut. The shortcut automatically shows the standard UAC prompt. Windows ARM64 runs the amd64 build through the operating system's x64 emulation.

Prebuilt GUI releases are available for macOS amd64/arm64, Linux amd64, and Windows amd64. To install a specific version on macOS or Linux, set, for example, `GO_SOCKS2VPN_VERSION=v1.0.0`. On Windows, download the script and run it with `-Version v1.0.0`.

Build manually from source:

```bash
make gui
sudo ./bin/socks2vpn-gui
```

On Windows, launch `socks2vpn-gui.exe` with **Run as administrator**. On Linux, use `sudo -E`; on macOS, use `sudo` from Terminal. The GUI checks privileges before initializing the VPN. Without root or Administrator privileges, the main window does not open; instead, the application displays a modal alert with the correct launch command and exits when the alert is closed. The GUI saves the server, port, and username, but never the password.

## Android

Android uses the system `VpnService`, so root access is not required. The first launch shows the standard Android VPN permission dialog. The application's own UID is excluded from the TUN route so that the Go engine's connection to the SOCKS server does not loop back into the VPN.

Local build requirements:

- JDK 17;
- Android SDK Platform 36 and Build Tools 36.0.0;
- Android NDK `28.2.13676358`;
- Go 1.26.3 for upstream tun2socks `v2.7.0`.

```bash
cd android
./gradlew assembleDebug
```

The Gradle script builds the shared Go engine into `app/libs/tun2socks.aar` through `gomobile`; this is a library inside the APK, not a separate process. The resulting APK is written to `android/app/build/outputs/apk/debug/app-debug.apk`.

The Android application supports IPv4 and IPv6, configures DNS `1.1.1.1`, performs a TCP proxy preflight, and runs as a foreground VPN service. The password remains in process memory only until disconnection.

## Docker

The image is built for `linux/amd64` and `linux/arm64`. The container requires the `NET_ADMIN` capability and the `/dev/net/tun` device; the provided `compose.yaml` already configures both. The supplied SOCKS4 server `192.168.192.100:9050` is used by default.

Test the SOCKS4 handshake with the same code used by the TUN engine, without changing routes:

```bash
make docker-check
```

Start the VPN inside a separate container network namespace:

```bash
make docker-up
```

Test TCP traffic through the created `tun0` interface:

```bash
make docker-traffic-check
```

Other containers can use it as a sidecar with `network_mode: service:socks2vpn`. Routes are changed only inside that network namespace: a Docker container does not turn the macOS or Windows host network stack into a VPN. Use the native GUI for host-wide routing.

The server at `192.168.192.100:9050` has been tested with SOCKS4A and SOCKS5. Switch Compose to SOCKS5 to proxy UDP and DNS:

```bash
SOCKS_PROXY=socks5://192.168.192.100:9050 docker compose up --build socks2vpn
```

SOCKS4 has no UDP relay and does not support IPv6 destinations. In `socks4://` mode, TCP to IPv4 works, but UDP applications, IPv6 connections, and DNS requests entering the TUN interface cannot pass through the proxy.

## Testing and builds

```bash
make test
./scripts/build-all.sh
./scripts/build-gui.sh
```

`scripts/build-all.sh` creates CLI binaries for macOS amd64/arm64, Linux 386/amd64/arm/arm64/riscv64, and Windows 386/amd64/arm64. The GUI is built natively on each desktop system. GitHub Actions separately verifies unit tests, all CLI targets, four GUI runners, and the Android APK.

## GitHub Actions and releases

The `.github/workflows/build.yml` workflow runs unit tests, the race detector, `go vet`, every CLI build, four native GUI builds, an Android APK build, and a multi-architecture Docker build on every push and pull request. The macOS GUI is built separately for Apple Silicon and Intel.

Pushing a semantic version tag automatically creates a GitHub Release containing application archives, a signed Android APK, automatically generated release notes, and `SHA256SUMS`:

```bash
git tag v1.0.0
git push origin v1.0.0
```

Tags such as `v1.0.0-rc.1` are published as prereleases. A release is created only after all tests and platform builds succeed. No additional token or third-party release action is required: publishing uses the official `GITHUB_TOKEN` through the GitHub CLI installed on the runner.

The same tag publishes `linux/amd64` and `linux/arm64` images to GitHub Container Registry:

```bash
docker pull ghcr.io/santaklouse/go-socks2vpn:v1.0.0
```

For a signed Android APK, add four GitHub Actions secrets in the repository settings once:

- `ANDROID_KEYSTORE_BASE64` — the entire release keystore encoded as Base64;
- `ANDROID_KEYSTORE_PASSWORD` — the keystore password;
- `ANDROID_KEY_ALIAS` — the key alias;
- `ANDROID_KEY_PASSWORD` — the key password.

Create a release keystore interactively so that passwords do not enter shell history:

```bash
keytool -genkeypair -v -keystore go-socks2vpn-release.jks -alias go-socks2vpn -keyalg RSA -keysize 3072 -validity 10000
```

Generate the `ANDROID_KEYSTORE_BASE64` value:

```bash
# macOS
base64 -i go-socks2vpn-release.jks | tr -d '\n'

# Linux
base64 -w 0 go-socks2vpn-release.jks
```

Store `go-socks2vpn-release.jks` and its passwords separately from the repository. Losing the key prevents future APKs from updating an already installed application.

For pushes and pull requests, the workflow builds an installable debug APK. For a tag, it requires all four secrets, derives the version from the tag, uses the monotonically increasing workflow run number as the Android `versionCode`, and publishes a signed `go-socks2vpn-android.apk`. Without the key, the tagged pipeline fails before it can create an incomplete GitHub Release.

## Important limitations

- The SOCKS server must support the protocols required by the application. SOCKS4 carries TCP only; use SOCKS5 with UDP ASSOCIATE for UDP and DNS through the proxy.
- Desktop routes are changed globally and require elevated privileges. Do not forcibly terminate the process with `kill -9`, because it will not have a chance to roll back the changes. Rollback is automatic after a regular Ctrl+C.
- Every platform uses the same embedded Go engine from the `engine` package. Android passes the file descriptor from `VpnService` through a thin `gomobile` wrapper.
