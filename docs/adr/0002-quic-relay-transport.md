# ADR 0002: QUIC tunnel with an end-to-end encrypted relay

- Status: Accepted
- Date: 2026-11-21
- Related: [protocol](../protocol.md), [threat model](../threat-model.md)

## Context

Clients must reach hosts over the internet, often through CGNAT or mobile
networks where inbound connections are impossible. A relay solves reachability,
but it introduces a party that could read or modify traffic. The transport also
needs to carry many independent TCP streams and UDP datagrams without
head-of-line blocking.

## Decision

Use QUIC as the tunnel transport and a relay as the rendezvous point:

- QUIC provides TLS 1.3, multiplexed streams, datagrams and connection
  migration on network changes.
- The host keeps an outbound connection to the relay and registers its host ID.
- Clients ask the relay for a host ID; the relay bridges a bidirectional
  stream.
- Because the relay terminates QUIC, client and host run a Noise handshake
  **inside** the bridge. The relay only forwards ciphertext.
- Authorization is enforced by the host (client key allowlist), not the relay.

A TCP/443 TLS fallback is planned for networks that block UDP.

## Consequences

### Positive

- Works behind CGNAT and on mobile networks without port forwarding.
- No head-of-line blocking between proxied connections.
- Native datagram support for UDP.
- The relay is a dumb, stateless-in-memory forwarder; it cannot read traffic.
- Connection migration helps with Wi-Fi to cellular handoffs.

### Negative

- Two layers of encryption add overhead and complexity.
- QUIC is sometimes throttled or blocked; the TCP fallback is mandatory.
- The relay still learns metadata (who, when, how much).
- Running a relay means operational cost and abuse handling.

## Alternatives considered

| Alternative | Why not |
| --- | --- |
| TLS over TCP | Head-of-line blocking; UDP over TCP is awkward and lossy |
| WireGuard between peers | Needs a coordination plane and root on the host for NAT |
| WebRTC data channels | Heavy signalling, browser-oriented, poor mobile behaviour |
| Tailscale/ZeroTier | Excellent, but a third-party dependency and less control over the protocol |
| Direct port forwarding only | Fails behind CGNAT and most mobile networks |
| Custom UDP protocol | Reinventing congestion control, encryption and migration |
