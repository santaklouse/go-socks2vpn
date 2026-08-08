#!/usr/bin/env bash
set -euo pipefail

project_dir=$(cd "$(dirname "$0")/.." && pwd)
output_dir="$project_dir/dist"
go_bin=${GO_BIN:-go}
mkdir -p "$output_dir"

extension=""
if [[ "$("$go_bin" env GOOS)" == "windows" ]]; then
    extension=".exe"
fi

cd "$project_dir/desktop-gui"
"$go_bin" build -trimpath -o "$output_dir/socks2vpn-gui$extension" .
echo "GUI binary: $output_dir/socks2vpn-gui$extension"

if [[ "$($go_bin env GOOS)" == "darwin" ]]; then
    "$project_dir/scripts/package-macos-gui.sh" \
        "$output_dir/socks2vpn-gui" "$output_dir/go-socks2vpn.app"
elif [[ "$($go_bin env GOOS)" == "linux" ]]; then
    cp "$project_dir/packaging/linux/socks2vpn-url-handler" "$output_dir/"
    cp "$project_dir/packaging/linux/go-socks2vpn-url.desktop" "$output_dir/"
    chmod 0755 "$output_dir/socks2vpn-url-handler"
fi
