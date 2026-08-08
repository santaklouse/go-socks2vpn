#!/bin/sh
set -eu

project_dir=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
binary_path=${1:-"$project_dir/socks2vpn-gui"}
app_path=${2:-"$project_dir/go-socks2vpn.app"}
plist_buddy=/usr/libexec/PlistBuddy

if [ "$(uname -s)" != "Darwin" ]; then
    printf '%s\n' "Error: macOS application packaging requires macOS." >&2
    exit 1
fi
if [ ! -x "$binary_path" ]; then
    printf 'Error: GUI binary is missing or not executable: %s\n' "$binary_path" >&2
    exit 1
fi

rm -rf "$app_path"
osacompile -o "$app_path" "$project_dir/packaging/macos/launcher.applescript"
install -m 0755 "$binary_path" "$app_path/Contents/Resources/socks2vpn-gui"

plist="$app_path/Contents/Info.plist"
set_plist_string() {
    key=$1
    value=$2
    if ! "$plist_buddy" -c "Set :$key $value" "$plist" >/dev/null 2>&1; then
        "$plist_buddy" -c "Add :$key string $value" "$plist"
    fi
}

set_plist_string CFBundleIdentifier com.santaklouse.gosocks2vpn
set_plist_string CFBundleName go-socks2vpn
set_plist_string CFBundleDisplayName go-socks2vpn
"$plist_buddy" -c "Delete :CFBundleURLTypes" "$plist" >/dev/null 2>&1 || true
"$plist_buddy" -c "Add :CFBundleURLTypes array" "$plist"
"$plist_buddy" -c "Add :CFBundleURLTypes:0 dict" "$plist"
"$plist_buddy" -c "Add :CFBundleURLTypes:0:CFBundleURLName string com.santaklouse.gosocks2vpn.config" "$plist"
"$plist_buddy" -c "Add :CFBundleURLTypes:0:CFBundleTypeRole string Viewer" "$plist"
"$plist_buddy" -c "Add :CFBundleURLTypes:0:CFBundleURLSchemes array" "$plist"
"$plist_buddy" -c "Add :CFBundleURLTypes:0:CFBundleURLSchemes:0 string socks2vpn" "$plist"
"$plist_buddy" -c "Add :CFBundleURLTypes:0:CFBundleURLSchemes:1 string socks2vps" "$plist"

codesign --force --deep --sign - "$app_path"
printf 'macOS application: %s\n' "$app_path"
