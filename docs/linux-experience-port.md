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

## Pending

- Shared UI: animated connection control, amber captcha notice, traffic polling,
  compact settings, hover information, and multiline VK links.
- English/Russian JSON localization and system language detection.
- Routing lists and bounded endpoint DNS resolution before network mutations.
- ICMP server ping using Windows process invocation.
- Safe detection/reuse of existing VPS installations.
- Fork-specific update delivery, Windows installer/uninstaller, screenshots.
- Native Windows build/tests and manual Wintun/connect/disconnect verification.

Linux's privileged helper and shell installer must not replace Windows network
or installation code. This branch is not a completed or released Windows port.
