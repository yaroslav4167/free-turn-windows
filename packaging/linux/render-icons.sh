#!/bin/sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
source="$root/desktop/assets/icon.svg"

rsvg-convert -w 512 -h 512 -o "$root/desktop/assets/icon.png" "$source"
rsvg-convert -w 64 -h 64 -o "$root/desktop/assets/icon-64.png" "$source"
rsvg-convert -w 64 -h 64 -o "$root/desktop/assets/icon-light-64.png" "$root/desktop/assets/icon-light.svg"
rsvg-convert -w 1024 -h 1024 -o "$root/desktop/build/appicon.png" "$source"
mkdir -p "$root/desktop/frontend/public"
cp "$source" "$root/desktop/frontend/public/freeturn.svg"
