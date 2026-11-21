# Architecture

This document describes how Munna Proxy is put together, the data flows between
its parts, and the trust boundaries the design depends on. It is a living
document; update it with any structural change.

- Status: draft for the v1 design
- Related: [protocol](protocol.md), [threat model](threat-model.md),
  [ADRs](adr/)

## Goals

1. **One engine, many frontends.** All proxy behaviour lives in `core/`. The
   CLI, the Wails desktop app, and the Android app are thin clients of the same
   engine.
2. **No privilege on the host.** The host dials on behalf of clients; it never
   routes packets. This keeps host mode usable on macOS, Windows, Linux and
   Android without root or admin rights.
3. **Whole-device mirroring on clients.** A TUN device plus tun2socks captures
   traffic from applications that know nothing about proxies.
4. **Internet reach without exposing the host.** Hosts and clients meet through
   a relay, with an end-to-end encrypted channel the relay cannot read.
5. **Auditable security.** Small parsers, explicit policy, no anonymous mode,
   no destination logging on the relay.

## Non-goals

- Anonymity or censorship resistance. This is not Tor.
- A general-purpose L3 VPN. ICMP and multicast are not carried, so `ping` and
  LAN discovery protocols do not work through the tunnel.
- Bypassing network policy. Operators can detect and block it like any proxy.
- Hosting other people's traffic. The relay is for paired devices only.

## Modes

| Mode | Runs on | Purpose |
| --- | --- | --- |
| Host | macOS, Windows, Linux, Android | Accepts SOCKS5 and tunnel clients, dials targets on the host's network |
| Client | macOS, Windows, Linux, Android | Creates a TUN device, runs tun2socks, forwards to a host or relay |
| Relay | Linux (VPS) | Introduces hosts and clients, bridges an encrypted stream |
| Helper | macOS, Windows, Linux | Creates the TUN device, installs routes and DNS for the client |

A single installation can act as host and client at the same time.

## Components

| Package | Responsibility | Key dependencies |
| --- | --- | --- |
| `core/socks5` | SOCKS5 server: CONNECT, auth, reply mapping (UDP associate planned) | standard library |
| `core/dialer` | Outbound dialing, DNS, policy, happy eyeballs, limits | `net`, `net/netip` |
| `core/tunnel` | QUIC transport, control frames, streams, datagrams | `quic-go` |
| `core/noise` | Device keys, XX/IK handshakes, pairing tokens | `flynn/noise` |
| `core/relay` | Registration, rendezvous, bridging | `quic-go` |
| `core/tun` | TUN lifecycle, routes, DNS, kill switch | `wireguard-go/tun`, `tun2socks` |
| `core/expose` | Publish host loopback ports | `net`, `crypto/tls` |
| `core/discovery` | mDNS, invite URIs, QR payloads | `libp2p/zeroconf`, `qrterminal` |
| `core/config` | Schema, validation, key storage | OS key stores |
| `core/api` | Local control API for GUIs | `net/http` over unix socket/named pipe |

## Data flows

### 1. LAN direct

```
client app ──SOCKS5──▶ host:1080 ──▶ core/socks5 ──▶ core/dialer ──▶ target
```

The simplest path. Any SOCKS5-capable application can use it, including
browsers, `curl` and `proxychains`. Direct WAN use wraps the same stream in
TLS 1.3.

### 2. WAN through the relay

```
client ──QUIC──▶ relay ◀──QUIC── host
          └──── E2E Noise channel ────┘
```

The host keeps an outbound connection to the relay and registers its host ID.
The client asks the relay for that host ID. The relay bridges a bidirectional
stream; the client and host run Noise inside it, so the relay forwards bytes it
cannot decrypt.

### 3. Client TUN

```
app ──▶ TUN ──▶ tun2socks ──▶ 127.0.0.1:local SOCKS5 ──▶ shim ──▶ transport
```

tun2socks always speaks SOCKS5 to a loopback listener owned by the client. The
shim decides whether to send the request to a LAN host, through the relay, or
to an exposed host port. Keeping one local interface means tun2socks is never
aware of the transport.

### 4. Loopback access

A SOCKS5 request for `127.0.0.1:3000` reaches the **host's** loopback, because
the host makes the connection. Two mechanisms make this usable:

- **`munna expose 3000`** — the host listens on the LAN (optionally with TLS and
  auth) and forwards to `127.0.0.1:3000`. This works for every client,
  including ones that hard-code loopback.
- **`*.host.munna`** — the client resolver maps the name to a synthetic address;
  the shim translates it back to a host-loopback target. This works for
  applications that resolve names, not for hard-coded `127.0.0.1`.

The kernel never routes a client's own loopback through a TUN device, so these
indirections are required. They are a deliberate design constraint, not a bug.

## Trust boundaries

```
┌──────────────┐   untrusted network   ┌──────────┐   untrusted network   ┌──────────────┐
│ client device│ ────────────────────▶ │  relay   │ ◀──────────────────── │  host device │
│ (TUN + app)  │      E2E Noise        │ (VPS)    │      E2E Noise        │  (core)      │
└──────┬───────┘                       └──────────┘                       └──────┬───────┘
       │ unprivileged app                                                     │ trusted LAN
       ▼                                                                      ▼
┌──────────────┐                                                       ┌──────────────┐
│ munna-helper │  privileged, narrow API                               │ LAN targets  │
└──────────────┘                                                       └──────────────┘
```

- **Relay operator** is untrusted for confidentiality but trusted for
  availability. It learns host IDs, client IPs, timing and volume — never
  destinations or payloads.
- **LAN** is untrusted. Direct connections require authentication; TLS is
  available and recommended for WAN.
- **Helper** is trusted but minimal. It accepts a fixed schema over a local
  socket, validates every field, and never runs shell commands.
- **Host** is trusted for everything it proxies. A compromised host can observe
  and modify traffic, exactly like a compromised router.

## Deployment topologies

| Topology | Use case | Requirements |
| --- | --- | --- |
| LAN only | Same Wi-Fi, home lab, office | mDNS or manual address |
| Direct WAN | Host has a public IP or port forward | TLS, DDNS or static address |
| Relay | Host behind CGNAT or a mobile network | Relay VPS |
| Both | Roaming clients that move between LAN and cellular | Relay plus LAN discovery |

## Failure modes

| Failure | Behaviour |
| --- | --- |
| Relay unreachable | Client falls back to LAN/direct if configured; otherwise reports offline |
| Host offline | Relay returns `HOST_OFFLINE`; client backs off and retries |
| Tunnel drops | Kill switch blocks traffic (client); host closes streams |
| DNS failure on host | Requests fail with `HOST_UNREACHABLE`; client resolver surfaces the error |
| Sleep/wake | Client reconnects and re-establishes the TUN configuration |
| Key loss | Device must re-pair; host revokes the old key |

## Observability

- Structured logs via `log/slog`, JSON in daemon mode.
- Counters: active connections, bytes in/out, auth failures, relay reconnects.
- **No destination logging at info level**, on the host or the relay.
- Debug mode may log destinations locally, with an explicit warning.

## Future work

- **L3 mode**: WireGuard-style IP tunneling with host-side NAT, to carry ICMP
  and multicast. Requires root on the host and is out of scope for v1.
- **Hole punching**: STUN-assisted direct connections with relay fallback.
- **Per-app split tunneling**: already planned for Android; add per-process
  rules on desktop.
- **Multiple hosts**: a client profile list with automatic selection.
