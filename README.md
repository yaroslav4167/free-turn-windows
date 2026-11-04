# FreeTurn Desktop for Windows

Windows fork with the interface and shared functionality from
[yaroslav4167/free-turn-linux](https://github.com/yaroslav4167/free-turn-linux).
Original Windows project: [purkalolka/free-turn-windows](https://github.com/purkalolka/free-turn-windows).

## Interface and features

- Separate connection, servers, settings and event-log pages; 900 × 600 initial window.
- Rounded-star/circle connection animation, expressive loader, traffic counters/charts,
  server ICMP ping, and hover information panels.
- A pending captcha does not interrupt working streams. The connected button turns
  amber and a browser-verification action remains visible on every page.
- English/Russian JSON catalogs, system-language detection and persistent language
  selection without restarting. See [localization](docs/localization.md).
- Editable VPN allow/deny routing lists with domain snapshots, preserved profile DNS
  routes, and bounded endpoint DNS resolution before changing the connection.
- Persistent settings, Windows tray, optional elevated logon task and startup connection.
- Existing VPS detection, local profile import, VK multiline editing, and WireGuard
  users with legacy and hackdiaz-dev name metadata support.
- Fork-specific update checking, SHA-256 verified ZIP downloads, PE architecture and
  archive-path validation, staged installation, backup and startup-confirmed rollback.

The shared interface follows the Linux fork; reference images:

![Connection](docs/screenshots/connection.png)
![Profiles](docs/screenshots/servers.png)
![Settings](docs/screenshots/settings.png)

These reference images originate from the Linux fork, not a native Windows screenshot.

## Windows requirements

Windows 10/11 amd64 with Microsoft Edge WebView2 Runtime. Wintun is embedded.
The application requests administrator rights when launched to configure the VPN
adapter/routes, retaining the upstream Windows security model. DTLS fingerprint
pinning, anti-replay protection and existing configuration import are preserved.
No Linux privileged helper, systemd or shell installer is used on Windows.

## Install or run portable

After a Windows release is published, unpack `FreeTurn-Windows-amd64-<version>.zip`
and verify `checksums.txt`. Run `freeturn-desktop.exe` (or `Run_Admin.bat`) for portable
use, or `Install.cmd` to install to `%LOCALAPPDATA%\Programs\FreeTurn`. The installer
asks whether to create a desktop shortcut and adds a Start menu launcher.

The optional startup task is scoped to the current user's SID, runs interactively
with that user's elevated token, and stores no password. Existing unrelated tasks
and shortcuts are not overwritten. A first-use UAC prompt is expected.

`Uninstall.cmd` asks for confirmation and removes only the managed local client,
its launcher and startup task. Profiles/keys/settings and VPS installations are kept.
The shared Wintun driver is not removed. Close other FreeTurn instances before
installing or uninstalling.

## Existing servers and names

Do not deploy another server just to add a profile. The VPS wizard checks for known
installations and switches to local profile creation without reinstalling or starting
the remote service. Supply the existing connection parameters and client configuration.

WireGuard names support `ft-user`, `client`, `ft-client` comments and `clients.list`
metadata matched by exact public key. Unknown layouts may need manual import.
See [existing-server notes](docs/existing-server.md).

Deleting a local profile does not uninstall the VPS. The separate remote uninstall
action affects the remote installation and all profiles sharing it.

## Build and checks

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@v2.16.0
npm --prefix desktop/frontend ci
cd desktop
wails build -ldflags "-checklinkname=0 -s -w" -trimpath
```

Native Windows CI builds the EXE, tests frontend/Go packages with the race detector,
and runs vet. Linux CI retains regression coverage of shared code and ELF updates.
Release workflows inject the fork repository and desktop version into update checks.

Live Wintun routing, UAC, scheduled logon and update/restart behavior require a real
Windows validation pass in addition to automated tests. This branch is a port under
validation, not a claim that every native integration has been manually verified.
