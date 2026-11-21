# Development

How to build, test and run Munna Proxy locally.

- Related: [architecture](architecture.md), [roadmap](roadmap.md)

## Prerequisites

| Tool | Version | Needed for |
| --- | --- | --- |
| Go | 1.27+ | everything |
| Git | any recent | source control |
| Make | any | task runner |
| golangci-lint | 2.x | `make lint` |
| Node.js | 22+ | Wails frontend (M3+) |
| Wails CLI | v2 | desktop app (M3+) |
| Android Studio | current | Android app (M5+) |
| JDK | 17+ | Android build |
| gomobile | latest | Android AAR (M5+) |
| GoReleaser | v2 | local release snapshots |

Install the Go toolchain and linter on macOS:

```bash
brew install go golangci-lint goreleaser
```

## Bootstrap

```bash
git clone https://github.com/arafsk/munna-proxy.git
cd munna-proxy
make build
./bin/munna version
make ci
```

`make help` lists every target.

## Host and client in two minutes

Start the host. It generates a password when `-password` is omitted and prints
the invite plus a QR code:

```bash
make build
./bin/munna serve
```

By default it listens on `0.0.0.0:1080`, advertises over mDNS, and denies cloud
metadata endpoints. Useful flags:

| Flag | Default | Purpose |
| --- | --- | --- |
| `-listen` | `0.0.0.0:1080` | SOCKS5 listen address |
| `-user` / `-password` | `munna` / generated | RFC 1929 credentials |
| `-no-auth` | off | Disable auth; loopback listeners only |
| `-advertise` | detected LAN IP | `host:port` written into the invite |
| `-name` | system hostname | Name shown to clients |
| `-mdns` | `true` | Advertise `_munna-proxy._tcp` on the LAN |
| `-debug` | off | Verbose connection logs |

From another device on the same LAN, replace `192.168.1.10` with the host's
address:

```bash
# Traffic exits from the host's network.
curl --socks5-hostname user:password@192.168.1.10:1080 https://ifconfig.me

# The host's loopback is reachable.
curl --socks5-hostname user:password@192.168.1.10:1080 http://127.0.0.1:3000
```

In a browser, start a separate profile with the proxy configured:

```bash
# macOS Chrome example. Note the loopback bypass override.
"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" \
  --user-data-dir=/tmp/munna-chrome \
  --proxy-server="socks5://192.168.1.10:1080" \
  --proxy-bypass-list="<-loopback>"
```

Firefox needs `network.proxy.socks_remote_dns=true` and
`network.proxy.allow_hijacking_localhost=true` in `about:config`.

If mDNS or the invite is unavailable, `ssh -D 0.0.0.0:1080 -N localhost` and
`microsocks -p 1080` remain useful unauthenticated fallbacks on a trusted LAN.

Stop the host with `Ctrl-C`.

## Make targets

| Target | Purpose |
| --- | --- |
| `make build` | Build `munna`, `munna-relay`, `munna-helper` into `bin/` |
| `make run` | Run the CLI from source |
| `make test` | Unit tests with the race detector |
| `make cover` | Write `coverage.out` |
| `make vet` | `go vet` |
| `make fmt` | Format tracked Go files |
| `make fmt-check` | Fail on unformatted files |
| `make lint` | golangci-lint |
| `make ci` | Everything CI runs |
| `make cross` | Compile all six release targets |
| `make snapshot` | Local GoReleaser snapshot in `dist/` |
| `make clean` | Remove build artifacts |

## Testing

```bash
make test
go test -run TestName -v ./core/socks5
go test -fuzz=FuzzRequest -fuzztime=30s ./core/socks5
```

Guidelines:

- Table-driven tests for protocol parsing.
- Fuzz every wire parser; keep the corpus in `test/fuzz`.
- Integration tests start real listeners on loopback.
- End-to-end tests live in `test/` and are gated behind a build tag when they
  need elevated privileges or a TUN device.

## Cross-compilation

```bash
make cross
```

Equivalent to:

```bash
for target in darwin/amd64 darwin/arm64 windows/amd64 windows/arm64 linux/amd64 linux/arm64; do
  GOOS=${target%/*} GOARCH=${target#*/} CGO_ENABLED=0 go build ./...
done
```

Host, client and relay binaries are pure Go (`CGO_ENABLED=0`). The Android
library is built separately with gomobile and requires cgo and the NDK:

```bash
gomobile bind -target=android/arm64,android/arm,android/amd64 \
  -o android/app/libs/munna.aar ./mobile
```

## Local relay testing

The relay protocol can be exercised without a VPS:

```bash
# Terminal 1: relay
go run ./cmd/munna-relay --listen 127.0.0.1:4433

# Terminal 2: host pointed at the relay
go run ./cmd/munna --config testdata/host-relay.yaml serve

# Terminal 3: client pointed at the relay
go run ./cmd/munna --config testdata/client-relay.yaml connect
```

This validates registration, rendezvous, bridging and the Noise handshake.
Real NAT behaviour is tested later against a free-tier VM or a VPS.

## Platform notes

### macOS

- The host triggers the macOS 15+ local-network privacy prompt. Add
  `NSLocalNetworkUsageDescription` and `NSBonjourServices` to the app's
  `Info.plist`; without them mDNS and LAN binding fail silently.
- TUN uses `utun` through `wireguard-go/tun`. Creating and routing it requires
  root, which is why the helper exists.
- Development builds can be ad-hoc signed (`codesign --sign -`). Notarization
  is a release step, not a development requirement.
- The application firewall may prompt the first time the host listens on the
  LAN.

### Windows

- TUN uses `wintun.dll`, which must ship next to the helper for both `amd64`
  and `arm64`. The DLL is pre-signed by its authors; verify its redistribution
  license before shipping.
- The helper runs as a LocalSystem service and communicates over a named pipe.
- Windows on ARM runs x64 applications under emulation, so an x64 build is an
  acceptable fallback while native arm64 packaging matures.
- Unsigned builds trigger SmartScreen; use "More info → Run anyway" during
  development.

### Linux

- TUN requires `CAP_NET_ADMIN` or a polkit-authorized helper.
- Install the helper as a systemd unit with a restrictive sandbox
  (`NoNewPrivileges`, `ProtectSystem=strict`, `PrivateTmp`).
- Package as `.deb` plus an AppImage for portability.

### Android

- `VpnService` requires explicit user consent and shows a persistent
  notification. Only one VPN can be active at a time.
- Use `setBlocking(true)` and always-on VPN with "Block connections without
  VPN" for the kill switch.
- `INTERNET`, `FOREGROUND_SERVICE`, `POST_NOTIFICATIONS` and
  `BIND_VPN_SERVICE` permissions are required.
- Request a battery-optimization exemption, or the tunnel may be killed during
  doze.
- Sideload a self-signed release APK until a Play Store account exists.

## Signing and distribution (later)

The project intentionally avoids paid accounts during development. Before a
public release:

- **macOS**: Developer ID certificate, hardened runtime, notarization, and a
  signed privileged helper with a matching Team ID.
- **Windows**: Authenticode certificate to avoid SmartScreen warnings.
- **Android**: a release keystore and, optionally, a Play Store account.
- **Relay**: a VPS with a domain, TLS termination and monitoring.

Keep the helper interface identical in development and release so signing is a
packaging change, not a redesign.

## Troubleshooting

| Symptom | Likely cause |
| --- | --- |
| `munna: command not found` | `bin/` is not on `PATH`; use `./bin/munna` |
| Browser ignores the proxy for localhost | Loopback bypass; see the Chrome and Firefox settings above |
| Phone cannot reach the host | Host firewall, client isolation on the router, or wrong interface |
| mDNS does not find the host | macOS local-network permission, or multicast blocked on the network |
| TUN starts but nothing routes | Helper failed to install routes; check helper logs and `netstat -rn` |
| DNS resolves to the wrong place | Client resolver not intercepting; check the TUN DNS configuration |
| Relay reports `HOST_OFFLINE` | Host registration expired; check heartbeats and relay clock |
