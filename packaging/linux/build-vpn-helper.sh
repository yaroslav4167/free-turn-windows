#!/bin/sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
platform=${1:-linux/$(go env GOARCH)}
binary=${2:-$root/desktop/build/bin/freeturn-desktop}

case "$platform" in
  linux/*) arch=${platform#linux/} ;;
  *) echo "VPN helper requires a Linux build target: $platform" >&2; exit 1 ;;
esac

if [ ! -f "$binary" ]; then
  echo "Build the desktop application first: $binary" >&2
  exit 1
fi

directory=$(CDPATH= cd -- "$(dirname -- "$binary")" && pwd)
GOOS=linux GOARCH="$arch" CGO_ENABLED=0 go -C "$root" build \
  -ldflags '-checklinkname=0 -s -w' -trimpath \
  -o "$directory/freeturn-vpn-helper" ./cmd/freeturn-vpn-helper
