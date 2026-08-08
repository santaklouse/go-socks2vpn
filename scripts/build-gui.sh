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
