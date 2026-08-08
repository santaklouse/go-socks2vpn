#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd "$(dirname "$0")" && pwd)
output_dir="$script_dir/../app/libs"
output_file="$output_dir/tun2socks.aar"
tools_dir="$script_dir/.tools"
gomobile_version="v0.0.0-20260803200217-62cee1672c8e"
go_bin=${GO_BIN:-go}

mkdir -p "$output_dir" "$tools_dir"

engine_changed=""
if [[ -f "$output_file" ]]; then
    engine_changed=$(find "$script_dir/../../engine" -type f -name '*.go' -newer "$output_file" -print -quit)
fi
if [[ -f "$output_file" && -z "$engine_changed" && "$output_file" -nt "$script_dir/mobile.go" && "$output_file" -nt "$script_dir/go.mod" && "$output_file" -nt "$script_dir/../../go.mod" && "$output_file" -nt "$script_dir/../../go.sum" ]]; then
    echo "tun2socks.aar is up to date"
    exit 0
fi

go_version=$("$go_bin" env GOVERSION)
echo "Building Android tun2socks AAR with $go_version"
GOBIN="$tools_dir" "$go_bin" install "golang.org/x/mobile/cmd/gomobile@$gomobile_version"
GOBIN="$tools_dir" "$go_bin" install "golang.org/x/mobile/cmd/gobind@$gomobile_version"
PATH="$tools_dir:$PATH" "$tools_dir/gomobile" init

cd "$script_dir"
"$go_bin" mod download
PATH="$tools_dir:$PATH" "$tools_dir/gomobile" bind \
	-target=android \
    -androidapi=24 \
    -o "$output_file" \
    .

echo "Created $output_file"
