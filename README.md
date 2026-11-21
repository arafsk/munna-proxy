# Munna Proxy

**Turn any device into a SOCKS5 gateway to its own network.**

Munna Proxy lets one device — the *host* — share its network with other devices — the
*clients*. Clients see the host's public IP, reach the host's LAN, and can use services
that only listen on the host's loopback, as if they were sitting on the host's network.

> **Status: M1 complete.** The host runs a SOCKS5 endpoint with authentication,
> host-side DNS, IPv6, policy guards, mDNS advertisement and a terminal QR
> invite. The TUN client, desktop GUI, Android app and relay are next; see
> [docs/roadmap.md](docs/roadmap.md) for milestones and acceptance criteria.

## Why

- Test a development server that only listens on `127.0.0.1` from your phone.
- Reach a home or office LAN while travelling.
- Use a device's network as an egress point without installing a kernel driver.
- Give a whole device the host's network with one toggle, including apps that ignore
  proxy settings.
- Do all of it from one cross-platform app.

## How it works

```
Android / Desktop client
  TUN ─▶ tun2socks ─▶ local SOCKS5 shim ─▶ transport ─┐
                                                      │
                    LAN: SOCKS5 (+TLS) ───────────────┤
                    WAN: QUIC ─▶ relay ─E2E─▶ ────────┤
                                                      ▼
                                                  host core ─▶ host network stack
                                                           └─▶ 127.0.0.1 (expose)
```

The host never routes packets: it dials on behalf of clients. That means the host needs
no root on any platform, including Android. Clients that support a TUN device get
transparent, whole-device mirroring; every other device can simply point at the SOCKS5
endpoint.

## Platform support

| Platform | Host | Client (SOCKS5) | Client (TUN) | Status |
| --- | --- | --- | --- | --- |
| macOS (Intel / Apple silicon) | yes | yes | planned | host ready |
| Windows (amd64 / arm64) | yes | yes | planned | host ready |
| Linux (Ubuntu, amd64 / arm64) | yes | yes | planned | host ready |
| Android (arm64) | planned | planned | planned | scaffold |

## Quick start

Requires Go 1.27 or newer.

```bash
git clone https://github.com/arafsk/munna-proxy.git
cd munna-proxy
make build
./bin/munna version
./bin/munna serve
```

`munna serve` prints the credentials and a `munna://` invite as a QR code. From
another device on the same LAN, point any SOCKS5 client at the host:

```bash
curl --socks5-hostname user:password@192.168.1.10:1080 https://ifconfig.me
curl --socks5-hostname user:password@192.168.1.10:1080 http://127.0.0.1:3000
```

Use `-listen 127.0.0.1:1080 -no-auth` for a local-only proxy without
authentication. See [docs/development.md](docs/development.md#host-and-client-in-two-minutes)
for browser setup, flags and troubleshooting.

## Repository layout

```
.
├── cmd/                  # executables
│   ├── munna/            # CLI and daemon (host + client)
│   ├── munna-relay/      # relay server for internet access
│   └── munna-helper/     # privileged TUN helper
├── core/                 # proxy engine (library)
│   ├── api/              # local control API for the GUIs
│   ├── config/           # configuration schema and key storage
│   ├── dialer/           # host-side dialing and DNS
│   ├── discovery/        # mDNS, QR codes, invite links
│   ├── expose/           # publish host loopback ports
│   ├── noise/            # device identity and end-to-end handshake
│   ├── relay/            # relay client and server logic
│   ├── socks5/           # SOCKS5 server and client
│   ├── tun/              # TUN device, routes, DNS, kill switch
│   └── tunnel/           # QUIC transport and framing
├── internal/version/     # build metadata
├── mobile/               # gomobile entry points for Android
├── desktop/              # Wails desktop application
├── android/              # Kotlin + Jetpack Compose application
├── packaging/            # installers and release packaging
├── test/                 # integration, end-to-end and leak tests
├── docs/                 # architecture, protocol, threat model, ADRs
└── scripts/              # developer helpers
```

## Documentation

- [Architecture](docs/architecture.md) — components, data flows, trust boundaries.
- [Protocol](docs/protocol.md) — SOCKS5 profile, tunnel and relay wire formats.
- [Threat model](docs/threat-model.md) — assets, actors, mitigations.
- [Roadmap](docs/roadmap.md) — milestones and cut lines.
- [Development](docs/development.md) — build, test and platform notes.
- [Architecture decision records](docs/adr/) — why the design looks like this.

## Security

This project exposes a network service. Read [SECURITY.md](SECURITY.md) and the
[threat model](docs/threat-model.md) before deploying it. Never expose an
unauthenticated proxy to an untrusted network.

## License

Proprietary. All rights reserved. Licensing terms are to be determined; see
[LICENSE](LICENSE).
