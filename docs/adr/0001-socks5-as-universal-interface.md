# ADR 0001: SOCKS5 is the universal interface

- Status: Accepted
- Date: 2026-11-21
- Related: [protocol](../protocol.md), [architecture](../architecture.md)

## Context

The product must let a host share its network with clients on macOS, Windows,
Linux and Android. Two constraints dominate the design:

1. The host should not need root or administrator rights, because Android
   cannot grant them and desktop users should not have to.
2. Devices that know nothing about Munna should still be able to use the host,
   and our own clients should reuse the same code path.

A full L3 VPN would give the best fidelity (ICMP, multicast, arbitrary IP), but
it requires the host to route and NAT packets, which requires root and breaks
the Android-host story.

## Decision

Use SOCKS5 (RFC 1928, with RFC 1929 authentication) as the canonical interface
between clients and the host.

- The host runs a SOCKS5 server and dials targets from its own network stack.
- LAN clients may speak SOCKS5 directly, optionally wrapped in TLS.
- Our TUN clients run tun2socks against a local SOCKS5 shim; the shim forwards
  to a host over the Munna tunnel.
- The tunnel has its own framing because it carries streams and datagrams, but
  semantically it is still "connect to this address".

## Consequences

### Positive

- No privilege on the host on any platform, including Android.
- Third-party interoperability: browsers, `curl`, `proxychains`, databases and
  SSH clients work without our software.
- One dialer and policy layer to secure, test and fuzz.
- The host can be a phone sharing its mobile connection.

### Negative

- ICMP and multicast are not carried, so `ping` and LAN discovery fail.
- UDP support depends on UDP ASSOCIATE and tunnel datagrams.
- Applications that ignore proxy settings need a TUN client.
- Loopback requires an explicit `expose` mechanism or a magic hostname.

## Alternatives considered

| Alternative | Why not |
| --- | --- |
| WireGuard with host-side NAT | Requires root on the host; cannot run on an unrooted Android host |
| Custom protocol only | Loses third-party clients and forces every device to install the app |
| HTTP CONNECT | Narrower than SOCKS5; no UDP; worse tooling support |
| SSH dynamic forwarding | Excellent for testing, but no UDP, no discovery, poor UX |
| IPsec/L2TP | Heavy, OS-specific, poor NAT behaviour, no Android-host story |
