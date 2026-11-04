#!/bin/sh
# Build the native Linux client and package a self-contained download.
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$root"

if [ "$(uname -s)" != Linux ]; then
  echo 'Build this script on Linux (GTK/WebKit native build).' >&2
  exit 1
fi
for tool in go node npm pkg-config tar sha256sum; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "Missing build tool: $tool. See README.md → Сборка." >&2
    exit 1
  fi
done
node -e 'const [major, minor] = process.versions.node.split(".").map(Number); if (major < 22 || (major === 22 && minor < 18)) { console.error("Node.js 22.18+ is required"); process.exit(1) }'
if ! pkg-config --exists gtk+-3.0 webkit2gtk-4.1; then
  echo 'Install GTK3/WebKit development libraries: libgtk-3-dev libwebkit2gtk-4.1-dev (Debian/Ubuntu).' >&2
  exit 1
fi

# Pin the CLI to the Wails version used by this project.
go install github.com/wailsapp/wails/v2/cmd/wails@v2.16.0
gobin=$(go env GOBIN)
if [ -z "$gobin" ]; then gobin="$(go env GOPATH)/bin"; fi
npm --prefix desktop/frontend ci
version=${VERSION:-$(git describe --tags --exact-match 2>/dev/null || echo dev)}
repository=${UPDATE_REPOSITORY:-}
case "$repository" in
  *[!a-zA-Z0-9_./-]*) echo 'Invalid UPDATE_REPOSITORY' >&2; exit 1 ;;
esac
case "$version" in
  ''|*[!a-zA-Z0-9._+-]*) echo 'Invalid VERSION' >&2; exit 1 ;;
esac
(
  cd desktop
  "$gobin/wails" build -tags webkit2_41 -ldflags "-checklinkname=0 -s -w -X main.desktopVersion=$version -X main.updateRepository=$repository" -trimpath
)

# The Wails Linux post-build hook produces the helper alongside the GUI.
for name in freeturn-desktop freeturn-vpn-helper; do
  if [ ! -f "desktop/build/bin/$name" ]; then
    echo "Build did not produce $name" >&2
    exit 1
  fi
done

arch=$(go env GOARCH)
mkdir -p dist
stage=$(mktemp -d "$root/dist/.desktop-package.XXXXXX")
trap 'rm -r -- "$stage"' EXIT
trap 'exit 1' HUP INT TERM
cp desktop/build/bin/freeturn-desktop desktop/build/bin/freeturn-vpn-helper "$stage/"
cp README.md LICENSE packaging/linux/install-desktop.sh packaging/linux/uninstall-desktop.sh packaging/linux/freeturn.desktop "$stage/"
chmod +x "$stage/install-desktop.sh" "$stage/uninstall-desktop.sh"
cp desktop/assets/icon.svg "$stage/freeturn.svg"
mkdir -p "$stage/desktop/assets" "$stage/docs"
cp desktop/assets/icon.svg "$stage/desktop/assets/"
cp -R docs/screenshots "$stage/docs/"
cp docs/localization.md "$stage/docs/"
archive="FreeTurn-Linux-$arch.tar.gz"
tar -C "$stage" -czf "dist/$archive" .
(
  cd dist
  sha256sum "$archive" > "$archive.sha256"
)
echo "Built: $root/dist/$archive"
