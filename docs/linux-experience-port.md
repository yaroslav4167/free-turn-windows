# Desktop experience port

Source: `yaroslav4167/free-turn-linux`, branch `feat/linux-desktop-experience`.
Target: this fork's `feat/windows-desktop-experience` branch.

## Preserve from Windows upstream

- Wintun lifecycle, administrator requirements, route/DNS cleanup.
- DTLS certificate fingerprint pinning, anti-replay handling, and protected
  stored credentials from the hardened core.
- Existing configuration import and server-control compatibility.

## Started

- Keep working streams connected when another VK link requires captcha;
  regression test added (pending CI).
- Default window size 900 × 600, matching the minimum supported dimensions.
- Bounded, cancellable endpoint DNS resolution before stopping the old session.
- Captcha URL in status snapshots and a persistent browser-verification notice.
- Traffic sampling/polling with stale-response protection and visible counters.
- Multiline VK editing preserves blank lines and spaces while typing.
- Read-only detection of common existing VPS installations; detected servers
  bypass install/WireGuard/start and save only a local profile with manually
  supplied existing credentials/configuration. Custom paths and containers may
  not be detected. SSH management is not attached to unknown layouts.
- SSH connection timers are captured per connection to avoid races during test
  cleanup; race detection remains enabled for the complete backend suite.

## Pending

- Remaining shared UI: animated connection control, compact settings, hover
  information, and traffic charts.
- English/Russian JSON localization and system language detection.
- Routing lists and Windows route application.
- ICMP server ping using Windows process invocation.
- Fork-specific update delivery, Windows installer/uninstaller, screenshots.
- Native Windows build/tests and manual Wintun/connect/disconnect verification.

Linux's privileged helper and shell installer must not replace Windows network
or installation code. This branch is not a completed or released Windows port.
