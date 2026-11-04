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
- Rounded-star-to-circle power control, eased rotation and oval-first morphing
  loader. Pending setup is cancellable and not falsely labelled VPN-enabled;
  working connections with captcha remain amber and can be disconnected.
- Multiline VK editing preserves blank lines and spaces while typing.
- Read-only detection of common existing VPS installations; detected servers
  bypass install/WireGuard/start and save only a local profile with manually
  supplied existing credentials/configuration. Custom paths and containers may
  not be detected. SSH management is not attached to unknown layouts.
- SSH connection timers are captured per connection to avoid races during test
  cleanup; race detection remains enabled for the complete backend suite.

## Full shared interface merged

The complete Linux branch was merged from the common Windows ancestor, including
navigation, profile/settings/log pages, charts, hover tips, routing, JSON languages,
updates and profile management. The earlier temporary Windows-only connection page
was replaced by the same shared components rather than kept as a second UI.

Windows adaptations now include native system locale detection, per-user installation,
owned shortcuts, an interactive highest-privilege logon task, conservative removal,
.NET ICMP probing, PE/ZIP update validation and a separate EXE update worker with
startup confirmation/rollback. Windows release packaging remains ZIP, not Linux tar.

## Validation pending

- Native Windows build/race/vet for the complete merged port and adapters.
- Real-machine Wintun/connect/disconnect, UAC, scheduled logon and GUI-update tests.
- Native Windows screenshots (current README reference images are from Linux).

Linux's privileged helper and shell installer must not replace Windows network
or installation code. This branch is not a completed or released Windows port.
