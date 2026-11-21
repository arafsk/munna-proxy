# Threat model

This document describes what Munna Proxy protects, who might attack it, and
which mitigations are required. It is a design document, not a claim that the
current scaffold is secure.

- Status: draft for v1
- Related: [architecture](architecture.md), [protocol](protocol.md),
  [security policy](../SECURITY.md)

## Scope and assumptions

In scope: the proxy engine, tunnel, relay, client TUN stack, privileged helper,
pairing and configuration.

Assumed:

- The host and client operating systems are not compromised.
- Users install the app from a trusted source.
- The relay is operated by someone who may be curious or malicious, but who
  wants to keep the service running.
- The local network is hostile (public Wi-Fi, shared office, hotel).

Out of scope: compromised kernels, physical access to an unlocked device,
nation-state adversaries, anonymity guarantees, and legal or policy compliance
in the user's jurisdiction.

## Assets

| Asset | Why it matters |
| --- | --- |
| Proxied traffic | May contain credentials, source code, personal data |
| Device static keys | Identity; compromise allows impersonation |
| Host credentials | Access to the proxy |
| Pairing tokens | One-time authorization to join a host |
| Client allowlist | Decides which devices may use a host |
| Host availability | The host's network is the product |
| Metadata | Who talks to which host, when, and how much |

## Actors

- **On-path network attacker** — same Wi-Fi, ISP, or compromised router.
- **Malicious relay operator** — controls the rendezvous server.
- **Unauthorized client** — wants to use a host as a free open proxy.
- **Malicious LAN device** — can reach the host's SOCKS5 port.
- **Local unprivileged user** — another account on the same machine.
- **Device thief** — has the physical device.
- **Supply-chain attacker** — compromises a dependency or build.

## Trust boundaries

1. Client application ↔ network.
2. Network ↔ relay.
3. Relay ↔ host.
4. Host ↔ LAN targets.
5. Unprivileged UI ↔ privileged helper.
6. Application ↔ configuration and key store.

## Threats and mitigations

| ID | Threat | Impact | Mitigation | Residual risk |
| --- | --- | --- | --- | --- |
| T1 | Unauthenticated access to the proxy | Open proxy, traffic attribution to host | Auth required by default; no-auth only on loopback and explicitly enabled | Misconfiguration |
| T2 | Credential brute force | Account takeover | Constant-time compare, per-peer rate limits, lockout, high-entropy generated credentials | Slow distributed guessing |
| T3 | On-path MITM on LAN | Traffic disclosure or injection | TLS 1.3 with pinned certificate for direct WAN; Noise E2E through relay | LAN clients that opt out of TLS |
| T4 | Relay reads traffic | Full disclosure | End-to-end Noise inside the relay bridge; relay never holds keys | Compromised client or host endpoint |
| T5 | Relay metadata correlation | Who-talks-to-whom, volume, timing | No destination logging; in-memory registry; minimal retention; document visibility | Traffic analysis remains possible |
| T6 | DNS leaks | Queries bypass the tunnel and reveal intent | Client resolver forwards through the tunnel; fake DNS for `*.host.munna`; block plaintext DNS outside the tunnel | Apps using DoH to a hard-coded resolver still traverse the tunnel, but the host sees the name |
| T7 | Traffic leaks when the tunnel drops | Real IP and destinations exposed | Kill switch: firewall rules on desktop, always-on VPN with block-without-VPN on Android | Brief window during setup or teardown |
| T8 | Open-proxy abuse and SSRF | Host used to attack third parties or cloud metadata | Authenticated clients only; metadata endpoints denied; optional port deny list; connection caps | Authorized malicious client |
| T9 | Helper privilege escalation | Full local root/admin | Narrow fixed-schema API, peer credential checks, no shell, config validation, signed client verification | Kernel or helper implementation bug |
| T10 | Local key theft | Impersonation | OS key stores (Keychain, DPAPI, Secret Service, Keystore); no plaintext keys on disk | Malware running as the user |
| T11 | Invite token replay | Unauthorized pairing | Single-use, short-lived tokens; bound to the Noise XX handshake; invalidation on first use | Token captured before first use |
| T12 | Resource exhaustion | Host or relay outage | Per-peer connection caps, idle timeouts, datagram size limits, relay rate limits | Large-scale DDoS |
| T13 | Supply-chain compromise | Arbitrary code execution | Minimal dependencies, license review, pinned versions, dependency scanning, reproducible builds where possible | Upstream compromise |
| T14 | Log leakage | Destination and identity disclosure | No destination logging at info level; redaction; opt-in debug logs with warning | Local debug logs |
| T15 | Protocol downgrade | Weaker or plaintext transport | ALPN and version checks; refuse plaintext on WAN; pin fingerprints | User explicitly disabling TLS |
| T16 | Client impersonation | Unauthorized use of a host | Static client keys in the host allowlist; Noise mutual authentication | Stolen client key |
| T17 | Host impersonation | Client sends traffic to an attacker | Host key pinning from the invite fingerprint; relay cannot substitute keys because of E2E | User ignoring a fingerprint change |
| T18 | IPv6 leak | Traffic bypasses the tunnel | Route both `0.0.0.0/0` and `::/0`; block native IPv6 outside the tunnel | OS-level split routing bugs |
| T19 | WebRTC and similar leaks | Real IP disclosure | Document browser settings; TUN captures UDP so most WebRTC traffic follows the tunnel | Browser-specific bypass features |
| T20 | Host exposed on a public network | Unauthorized access, scanning | Auth, rate limits, optional TLS-only mode, warning when binding to a non-private interface | Zero-day in the parser |

## Security requirements

These are release blockers for v1:

- [ ] Authentication enabled by default; anonymous mode does not exist.
- [ ] Constant-time credential comparison and auth rate limiting.
- [ ] TLS 1.3 for direct WAN connections with certificate pinning.
- [ ] Noise end-to-end channel through the relay.
- [ ] Metadata endpoint denial and policy validation in `core/dialer`.
- [ ] Helper API with peer credential checks and no command execution.
- [ ] Keys stored in OS key stores.
- [ ] Kill switch on desktop and Android always-on support.
- [ ] No destination logging at info level.
- [ ] Fuzz tests for SOCKS5 and tunnel parsers.
- [ ] Dependency and license review.
- [ ] Threat model reviewed before the first public release.

## Residual risks

- The relay always learns metadata (who, when, how much), even though not
  content.
- The host sees plaintext traffic it proxies; a compromised host is equivalent
  to a compromised network.
- ICMP and multicast are not carried, so some LAN discovery and diagnostics do
  not work.
- Applications that hard-code `127.0.0.1` cannot be redirected to the host
  without `munna expose`.
- Users can weaken security by disabling TLS, enabling anonymous mode, or
  ignoring fingerprint changes.

## Review triggers

Revisit this document when any of the following change:

- the transport or relay protocol,
- the pairing and key model,
- the privileged helper surface,
- the set of supported platforms,
- the release and update process.
