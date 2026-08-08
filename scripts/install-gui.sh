#!/bin/sh
set -eu

repository="santaklouse/go-socks2vpn"
requested_version="${GO_SOCKS2VPN_VERSION:-latest}"
install_dir="${GO_SOCKS2VPN_INSTALL_DIR:-/usr/local/bin}"
skip_dependencies="${GO_SOCKS2VPN_SKIP_DEPENDENCIES:-0}"

say() {
    printf '%s\n' "$*"
}

fail() {
    printf 'Error: %s\n' "$*" >&2
    exit 1
}

require_command() {
    command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

run_as_admin() {
    if [ "$(id -u)" -eq 0 ]; then
        "$@"
    elif command -v sudo >/dev/null 2>&1; then
        sudo "$@"
    else
        fail "root or sudo is required to install system files"
    fi
}

sha256_file() {
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$1" | awk '{print tolower($1)}'
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "$1" | awk '{print tolower($1)}'
    elif command -v openssl >/dev/null 2>&1; then
        openssl dgst -sha256 "$1" | awk '{print tolower($NF)}'
    else
        fail "no SHA-256 verification command was found"
    fi
}

install_linux_dependencies() {
    binary_path=$1

    if [ "$skip_dependencies" = "1" ]; then
        say "Linux dependency installation was skipped via GO_SOCKS2VPN_SKIP_DEPENDENCIES=1."
        return
    fi

    if command -v ldd >/dev/null 2>&1 && ! ldd "$binary_path" 2>&1 | grep -q 'not found'; then
        say "All Linux GUI dependencies are already installed."
        return
    fi

    say "Installing Linux GUI dependencies…"
    if command -v apt-get >/dev/null 2>&1; then
        run_as_admin apt-get update
        run_as_admin env DEBIAN_FRONTEND=noninteractive apt-get install -y \
            libgl1 libwayland-client0 libxkbcommon0 libx11-6 libxcursor1 \
            libxrandr2 libxinerama1 libxi6 libxxf86vm1
    elif command -v dnf >/dev/null 2>&1; then
        run_as_admin dnf install -y \
            mesa-libGL wayland-libs libxkbcommon libX11 libXcursor \
            libXrandr libXinerama libXi libXxf86vm
    elif command -v yum >/dev/null 2>&1; then
        run_as_admin yum install -y \
            mesa-libGL wayland-libs libxkbcommon libX11 libXcursor \
            libXrandr libXinerama libXi libXxf86vm
    elif command -v pacman >/dev/null 2>&1; then
        run_as_admin pacman -S --needed --noconfirm \
            mesa wayland libxkbcommon libx11 libxcursor libxrandr \
            libxinerama libxi libxxf86vm
    else
        fail "unsupported package manager; install the OpenGL, Wayland, XKB, and X11 runtime libraries manually"
    fi

    if command -v ldd >/dev/null 2>&1; then
        missing_libraries=$(ldd "$binary_path" 2>&1 | awk '/not found/ {print $1}' | tr '\n' ' ')
        [ -z "$missing_libraries" ] || fail "libraries are still missing after installation: $missing_libraries"
    fi
}

require_command curl
require_command tar
require_command uname
require_command awk
require_command grep
require_command install

case "$(uname -s)" in
    Darwin)
        platform="macos"
        ;;
    Linux)
        platform="linux"
        ;;
    *)
        fail "this command supports macOS and Linux; use install-gui.ps1 on Windows"
        ;;
esac

case "$(uname -m)" in
    x86_64|amd64)
        architecture="amd64"
        ;;
    arm64|aarch64)
        architecture="arm64"
        ;;
    *)
        fail "unsupported architecture: $(uname -m)"
        ;;
esac

if [ "$platform" = "linux" ] && [ "$architecture" != "amd64" ]; then
    fail "the prebuilt Linux GUI is currently available only for amd64"
fi

if [ "$requested_version" = "latest" ]; then
    release_base="https://github.com/$repository/releases/latest/download"
elif printf '%s\n' "$requested_version" | grep -Eq \
    '^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$'; then
    release_base="https://github.com/$repository/releases/download/$requested_version"
else
    fail "GO_SOCKS2VPN_VERSION must be latest or a semantic version tag such as v1.0.0"
fi

asset_name="go-socks2vpn-gui-$platform-$architecture.tar.gz"
temp_dir=$(mktemp -d "${TMPDIR:-/tmp}/go-socks2vpn-gui.XXXXXX")
trap 'rm -rf "$temp_dir"' EXIT HUP INT TERM

say "Downloading ${asset_name}…"
curl --proto '=https' --tlsv1.2 -fsSL --retry 3 \
    "$release_base/$asset_name" -o "$temp_dir/$asset_name"
curl --proto '=https' --tlsv1.2 -fsSL --retry 3 \
    "$release_base/SHA256SUMS" -o "$temp_dir/SHA256SUMS"

expected_hash=$(awk -v file="$asset_name" \
    '$2 == file || $2 == "*" file {print tolower($1); exit}' \
    "$temp_dir/SHA256SUMS")
[ -n "$expected_hash" ] || fail "$asset_name is missing from SHA256SUMS"

actual_hash=$(sha256_file "$temp_dir/$asset_name")
[ "$actual_hash" = "$expected_hash" ] || fail "the downloaded archive has an unexpected SHA-256"
say "SHA-256 verified: $actual_hash"

tar -xzf "$temp_dir/$asset_name" -C "$temp_dir"
binary_path="$temp_dir/socks2vpn-gui"
[ -f "$binary_path" ] || fail "the archive does not contain socks2vpn-gui"
chmod 0755 "$binary_path"

if [ "$platform" = "linux" ]; then
    install_linux_dependencies "$binary_path"
fi

if mkdir -p "$install_dir" 2>/dev/null && [ -w "$install_dir" ]; then
    install -m 0755 "$binary_path" "$install_dir/socks2vpn-gui"
else
    run_as_admin mkdir -p "$install_dir"
    run_as_admin install -m 0755 "$binary_path" "$install_dir/socks2vpn-gui"
fi

say "GUI installed: $install_dir/socks2vpn-gui"
say "Run with administrator privileges: sudo -E $install_dir/socks2vpn-gui"
