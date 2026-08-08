#!/usr/bin/env bash
set -euo pipefail

project_dir=$(cd "$(dirname "$0")/.." && pwd)
output_dir="$project_dir/dist"
version=${VERSION:-dev}
go_bin=${GO_BIN:-go}

mkdir -p "$output_dir"

targets=(
    "darwin amd64"
    "darwin arm64"
    "linux 386"
    "linux amd64"
    "linux arm"
    "linux arm64"
    "linux riscv64"
    "windows 386"
    "windows amd64"
    "windows arm64"
)

for target in "${targets[@]}"; do
    read -r target_os target_arch <<<"$target"
    extension=""
    if [[ "$target_os" == "windows" ]]; then
        extension=".exe"
    fi
    destination="$output_dir/socks2vpn-$target_os-$target_arch$extension"
    echo "Building $target_os/$target_arch"
    (
        cd "$project_dir"
        CGO_ENABLED=0 GOOS="$target_os" GOARCH="$target_arch" \
            "$go_bin" build -trimpath -ldflags "-s -w -X main.version=$version" \
            -o "$destination" ./cmd/socks2vpn
    )
done

echo "CLI binaries are in $output_dir"
