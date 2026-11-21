# Tests

Status: **conventions in place**, suites arrive with the engine.

## Test pyramid

| Layer | Location | What it covers |
| --- | --- | --- |
| Unit | next to the package | Parsers, policy, framing, error mapping |
| Fuzz | `test/fuzz`, `*_test.go` | SOCKS5 and tunnel parsers |
| Integration | `test/integration` | Real listeners, auth, UDP associate |
| End-to-end | `test/e2e` | Host + client + relay, leak checks |
| Platform | `test/platform` | TUN, routes, DNS, kill switch per OS |

## Commands

```bash
make test                                  # unit tests with -race
go test -run TestName -v ./core/socks5     # one package
go test -fuzz=FuzzRequest -fuzztime=30s ./core/socks5
```

End-to-end suites that need a TUN device or elevation are gated behind a build
tag and skipped by default:

```bash
go test -tags=e2e ./test/e2e
```

## Leak suite

Every client milestone must pass:

- public IP equals the host's public IP,
- DNS resolves through the host (no local resolver traffic),
- IPv6 does not bypass the tunnel,
- traffic stops when the tunnel is killed (kill switch),
- browser WebRTC does not reveal the client address.

Suggested tools: `ifconfig.me`, `dnsleaktest.com`, `test-ipv6.com`, browser
WebRTC test pages.

## Relay harness

`test/integration` will start a relay on loopback, register a host, connect a
client and verify:

- registration, heartbeat expiry and re-registration,
- rendezvous for a live and an offline host,
- end-to-end encryption (the relay sees no plaintext),
- rate limiting and connection caps.

## CI

`.github/workflows/ci.yml` runs build, vet and tests on Ubuntu, macOS and
Windows, plus lint and a six-target cross-compile. TUN and Android suites run
locally or on self-hosted runners until hosted runners support them.
