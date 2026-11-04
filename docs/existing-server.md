# Existing VPS installations

The SSH wizard checks for the managed `/opt/free-turn-proxy` binary, known
systemd units (`freeturn-server`, `free-turn-server`, `free-turn-proxy`, and
`vk-turn-proxy`), and common `freeturn-server` installation paths. It repeats
the check before installation. Detection is read-only; it does not execute
unknown binaries or source their configuration files.

If an installation is found, the wizard skips installation, WireGuard setup,
and server startup. Enter the existing connection parameters, authorized client
ID, obfuscation key, and client WireGuard/AmneziaWG configuration to save a
local profile. These values are not recovered automatically. SSH management is
not attached to this profile, so it cannot accidentally reconfigure an unknown
installation layout.

Custom service names, container-only deployments, and nonstandard paths may not
be detected. For these, add/import an existing profile instead of deploying a
new VPS.

Deleting a profile from the application's server list removes only local data.
The separate **Uninstall from VPS** management action removes the remote managed
installation and can affect every profile using that same installation.

## SSH management over VPN on Linux

The relay's own sockets are bound to the physical network interface. The VPS
address therefore does not need a global host-route exemption on Linux: that
exemption also bypassed the VPN for SSH, causing timeouts on networks where the
VPS cannot be reached directly. SSH now follows the profile's VPN routes.

With split routing, the VPS address must be included in the VPN routes (and not
excluded by a blacklist). VPN connectivity does not override a firewall that
blocks SSH on the VPS, a wrong port, or a stopped SSH service.

WireGuard names are read from `# ft-user: <base64>` and older `# client: <name>`
comments in the managed server configuration. WireGuard itself stores public
keys and addresses, not names; if the comments were removed, there is no name
to recover from its live peer list.
