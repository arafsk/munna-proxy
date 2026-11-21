# Documentation

| Document | What it covers |
| --- | --- |
| [Architecture](architecture.md) | Components, data flows, trust boundaries, failure modes |
| [Protocol](protocol.md) | SOCKS5 profile, tunnel and relay wire formats, pairing |
| [Threat model](threat-model.md) | Assets, actors, threats, mitigations, residual risk |
| [Roadmap](roadmap.md) | Milestones, acceptance criteria, cut lines |
| [Development](development.md) | Build, test, cross-compile, platform notes |

## Architecture decision records

| ADR | Decision |
| --- | --- |
| [0001](adr/0001-socks5-as-universal-interface.md) | SOCKS5 is the universal interface |
| [0002](adr/0002-quic-relay-transport.md) | QUIC tunnel with an end-to-end encrypted relay |
| [0003](adr/0003-tun-privileged-helper.md) | TUN is managed by a narrow privileged helper |
| [0004](adr/0004-identity-and-pairing.md) | Static keys, Noise handshakes and QR pairing |
| [0005](adr/0005-ui-stack.md) | Wails for desktop, Kotlin/Compose for Android |

## Conventions

- Documents are written in Markdown and kept short enough to review.
- Wire formats live in [protocol.md](protocol.md); do not duplicate them in
  code comments beyond a reference.
- Any change to a trust boundary updates [threat-model.md](threat-model.md).
- Structural decisions get an ADR; superseded ADRs are marked, not deleted.
