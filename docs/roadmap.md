# Roadmap

Milestones for the first usable release. Estimates assume one developer working
part-time; full-time work compresses them roughly by half.

- Status: draft
- Related: [architecture](architecture.md), [development](development.md)

## Principles

1. **Every milestone ships something usable.** M1 is already a LAN proxy.
2. **Risky integrations come early.** TUN and the privileged helper are spiked
   before UI polish.
3. **Security is not deferred.** Auth, TLS and fuzzing land with the code they
   protect.
4. **Internet access is optional.** LAN + TUN is a complete product; the relay
   is an enhancement.

## Milestones

| # | Milestone | Deliverable | Acceptance criteria | Est. |
| --- | --- | --- | --- | --- |
| M0 | Proof | None (manual) | A second device reaches the host's public IP and `127.0.0.1` through a temporary SOCKS5 server | 0.5 d |
| M1 | SOCKS5 core | `munna` binary, mDNS, QR, 6-target CI | `curl --socks5-hostname` and a browser work over LAN; IPv6 works; parser fuzz-clean | 1–2 w |
| M2 | macOS TUN | `munna connect` on macOS, helper prototype | Whole-device mirror; host DNS; no IPv4, IPv6 or DNS leaks | 2–3 w |
| M3 | Desktop GUI | Wails app with host/client modes | A non-developer can host and connect two Macs in under two minutes | 1–2 w |
| M4 | Windows + Linux | wintun + service, tun + polkit, installers | The same end-to-end suite passes on all three desktops | 2–3 w |
| M5 | Android | APK with VpnService, QR scanner | Phone mirrors the host, including apps that ignore proxies | 2–3 w |
| M6 | Relay | `munna-relay`, QUIC, Noise, ACL | Host behind CGNAT + phone on cellular; relay sees ciphertext only | 2–4 w |
| M7 | Release | Auto-update, docs, signing | Public beta with a documented security review | ongoing |

## M0 — Proof

Manual validation only.

- Run a temporary SOCKS5 server on the host.
- From another device, confirm the public IP matches the host.
- Confirm a host-local service on `127.0.0.1` is reachable.
- Document the browser loopback-bypass settings.

**Done when:** the premise is demonstrated and the findings are recorded in
[development](development.md#two-minute-proof).

## M1 — SOCKS5 core

**Status: complete on `main`.**

- [x] `core/socks5` server: CONNECT, domain/IPv4/IPv6, RFC 1929 auth, reply codes.
- [x] `core/dialer`: host resolver, happy eyeballs, policy, limits.
- [x] `core/discovery`: mDNS advertisement, invite URI, QR in the terminal.
- [x] `munna serve` with a generated credential and a printed invite.
- [x] Unit, integration and fuzz tests.
- [x] GoReleaser builds for the six desktop targets.

**Done when:** a phone or laptop with any SOCKS5 client browses through the
host over LAN, including a host-local service, and `make ci` is green on all
three operating systems.

Deliberate limits carried into M2:

- The listener is plain TCP. TLS and the QUIC tunnel arrive with the relay.
- UDP ASSOCIATE and BIND return "command not supported".
- The invite token is informational until the Noise handshake exists; the
  SOCKS5 username/password is the only authentication today.
- mDNS is advertised, but client-side discovery lands with the GUI.

## M2 — macOS TUN

- `core/tun`: utun creation, default routes, DNS interception, teardown.
- tun2socks integration and the local SOCKS5 shim.
- `munna connect` with a config file and an invite URI.
- Helper prototype installed with an admin prompt; no signing required yet.
- Kill switch and leak tests.

**Done when:** with the tunnel up, `ifconfig.me`, a host-LAN device, a
host-local service and DNS all resolve through the host, and nothing leaks when
the tunnel is killed.

## M3 — Desktop GUI

- Wails shell with host and client modes, status, logs and QR display/scan.
- Helper install and removal flow.
- System-proxy toggle as a no-TUN fallback.
- Packaging: `.app` and `.dmg`.

**Done when:** two Macs can be connected by a non-developer in under two
minutes, and the app recovers from a host restart.

## M4 — Windows and Linux

- Windows: wintun, LocalSystem service, named-pipe helper API, MSI/NSIS.
- Linux: tun, systemd unit, polkit policy, `.deb` and AppImage.
- Platform-specific route and DNS handling.
- CI end-to-end coverage where runners allow it.

**Done when:** the M2 acceptance suite passes on Windows 11 (amd64 and arm64)
and Ubuntu (amd64 and arm64).

## M5 — Android

- `gomobile` AAR with start/stop and status.
- Kotlin + Compose UI, QR scanner, foreground service.
- VpnService with `0.0.0.0/0` and `::/0`, always-on and lockdown support.
- Per-app split tunneling.
- Battery and doze testing.

**Done when:** a phone on Wi-Fi mirrors the host's network for both proxy-aware
and proxy-blind apps, survives sleep/wake, and can be stopped cleanly.

## M6 — Relay

- `munna-relay` with QUIC and TCP/443 fallback.
- Host registration, heartbeats and reconnect.
- Client rendezvous and bridging.
- Noise XX/IK end-to-end channel and pairing/ACL.
- Rate limits and abuse controls.
- Local relay test harness, then a free-tier deployment.

**Done when:** a host behind CGNAT and a phone on cellular connect through the
relay, `tcpdump` on the relay shows only ciphertext, and reconnect works after
a network change.

## M7 — Release

- Auto-update with signature verification.
- Crash reporting, opt-in only.
- Documentation, install guides and troubleshooting.
- External security review or a public bug bounty, if feasible.
- Code signing and notarization once accounts exist.

**Done when:** a public beta can be installed by a stranger without reading the
source, and the security checklist in the threat model is complete.

## Cut lines

If the schedule slips, cut in this order:

1. Relay (M6) → v1.1. LAN + TUN is still useful.
2. Windows arm64 native → ship x64 under emulation.
3. Per-app split tunneling → v1.1.
4. Android as a host → v1.1 (client-only first).
5. Auto-update → manual downloads for the beta.

Do **not** cut: authentication, TLS, Noise, kill switch, fuzzing, or the
metadata-endpoint denial.

## Post-v1

- L3 mode with host-side NAT for ICMP and multicast.
- STUN hole punching with relay fallback.
- Multiple hosts and automatic selection.
- Desktop per-process split tunneling.
- IPv6-only network support.
- Metrics dashboard and connection history (local only).
