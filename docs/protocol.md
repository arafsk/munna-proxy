# Protocol

Wire formats used by Munna Proxy. This document is a **draft** for protocol
version 1 and will change before the first release.

- Status: draft
- Related: [architecture](architecture.md), [threat model](threat-model.md)

## Conventions

- Integers are big-endian unless stated otherwise.
- `u8`, `u16`, `u32`, `u64` are unsigned integers of that width.
- Addresses use the SOCKS5 address encoding (`ATYP` + address + `u16` port).
- Unknown control frames are ignored; unknown required extensions cause a
  `PROTOCOL_ERROR`.
- All strings are UTF-8 unless stated otherwise.
- Protocol version 1 is identified by the QUIC ALPN token `munna/1` and the
  relay ALPN `munna-relay/1`.

## SOCKS5 profile

Munna implements RFC 1928 (SOCKS5) and RFC 1929 (username/password
authentication) with the restrictions below.

### Transport

- Plain TCP for LAN use.
- TLS 1.3 for direct WAN use, with ALPN `munna-socks/1` and a pinned
  self-signed certificate. Clients verify the fingerprint from the invite.
- SOCKS5 inside the Munna tunnel is **not** used; the tunnel has its own
  framing (see below). The client shim translates locally.

### Authentication

| Method | Value | Policy |
| --- | --- | --- |
| No authentication | `0x00` | Rejected unless explicitly enabled and the peer is loopback |
| Username/password | `0x02` | Required by default |

Credentials are generated per host, compared in constant time, and rate
limited per peer: five failures in a minute trigger a five-minute lockout.
Failed attempts are logged without the supplied password.

### Commands

| Command | Value | Support |
| --- | --- | --- |
| CONNECT | `0x01` | Yes |
| BIND | `0x02` | No — reply `0x07` (command not supported) |
| UDP ASSOCIATE | `0x03` | Yes (v1.1) |

### Address types

| ATYP | Meaning | Notes |
| --- | --- | --- |
| `0x01` | IPv4 | 4 bytes |
| `0x03` | Domain name | 1-byte length + name; resolved **on the host** |
| `0x04` | IPv6 | 16 bytes |

Domain names are resolved by `core/dialer` using the host's resolver. Clients
must send names rather than resolving locally to preserve split-horizon DNS
behaviour.

### Policy defaults

- Deny cloud metadata endpoints: `169.254.169.254`, `fd00:ec2::254`, and
  equivalents, unless explicitly allowed.
- Allow loopback targets for authenticated peers; this is the mechanism behind
  host-localhost access.
- Optional deny list for dangerous ports (for example `25`, `445`), disabled by
  default.
- Per-peer connection caps and idle timeouts; defaults are set in `core/config`.

### Reply codes

The standard RFC 1928 codes are used. Munna-specific failures map as follows:

| Condition | Reply |
| --- | --- |
| Target refused | `0x05` connection refused |
| DNS failure | `0x04` host unreachable |
| Network unreachable | `0x03` |
| Policy denied | `0x02` connection not allowed |
| Timeout | `0x06` TTL expired |

## Munna tunnel v1

The tunnel carries proxied traffic between a client and a host over QUIC. When
a relay is used, the QUIC connection terminates at the relay and an inner Noise
channel provides end-to-end confidentiality.

### Layers

```
QUIC (TLS 1.3)            client ──▶ relay ──▶ host   (or direct to host)
└─ Noise XX/IK            client ◀──────────────▶ host  (end-to-end)
   ├─ control stream      frames (CBOR)
   ├─ data streams        one per TCP connection
   └─ datagrams           proxied UDP
```

### Control frames

A control frame is:

```
u8 type | u32 length | length bytes of CBOR payload
```

| Type | Name | Direction | Purpose |
| --- | --- | --- | --- |
| `0x01` | HELLO | C→H | Protocol version, client ID, capabilities |
| `0x02` | HELLO_OK | H→C | Version, host ID, limits, server time |
| `0x03` | CONNECT | C→H | Target address for a new stream |
| `0x04` | CONNECT_OK | H→C | Bound address, stream accepted |
| `0x05` | CONNECT_ERR | H→C | Error code and message |
| `0x06` | UDP_ASSOCIATE | C→H | Request a UDP association |
| `0x07` | UDP_ASSOCIATE_OK | H→C | Association ID and lifetime |
| `0x08` | PING | both | Liveness |
| `0x09` | PONG | both | Liveness |
| `0x0A` | CLOSE | both | Graceful shutdown |
| `0x0B` | GOAWAY | both | Stop accepting new streams |

### TCP streams

1. The client opens a bidirectional QUIC stream.
2. It sends a `CONNECT` frame containing the target address.
3. The host replies with `CONNECT_OK` or `CONNECT_ERR` on the same stream.
4. After `CONNECT_OK`, both sides exchange raw bytes until EOF or reset.
5. A client half-close maps to `FIN`; the host may keep sending until it
   closes.

One stream per TCP connection gives independent flow control and avoids
head-of-line blocking between connections.

### UDP datagrams

QUIC datagrams carry proxied UDP. The payload is:

```
u32 association_id | u8 flags | SOCKS5 UDP request
```

`flags` bit 0 indicates that the SOCKS5 UDP header (`RSV | FRAG | ATYP | ADDR |
PORT`) is present. `FRAG` must be zero in v1; fragmented UDP is not supported.
Association IDs are allocated by the host and expire after the lifetime
returned in `UDP_ASSOCIATE_OK`.

The negotiated maximum datagram size must fit the path MTU; the tunnel
advertises a conservative default and falls back to dropping oversized
datagrams with a counter increment.

### Limits and timeouts

| Limit | Default |
| --- | --- |
| Concurrent TCP streams per client | 512 |
| UDP associations per client | 64 |
| Control idle timeout | 30 s (PING every 10 s) |
| TCP idle timeout | 5 min |
| Handshake timeout | 10 s |

### Reconnection

On disconnect the client reconnects with exponential backoff and reuses its
static key (Noise IK). Session resumption is allowed, but authentication is
always re-established; 0-RTT is not used for data.

## Relay protocol

The relay is a rendezvous and forwarding service. It must not be able to read
tunnel traffic.

### Registration

The host connects to the relay over QUIC (`munna-relay/1`) or TLS on TCP 443
and sends a `REGISTER` frame:

```
u8  type = 0x01
u8  version = 1
16B host_id
32B static_public_key
u64 timestamp
64B signature over "munna-relay-register-v1" || host_id || timestamp
```

The relay verifies the signature and records `host_id → connection` in memory.
It sends `REGISTER_OK` with a heartbeat interval. Registration expires if the
heartbeat stops.

### Rendezvous

A client sends `LOOKUP` with the target `host_id`. The relay replies:

- `FOUND` — a bridge stream follows.
- `HOST_OFFLINE` — no live registration.
- `RATE_LIMITED` — too many attempts.

After `FOUND`, the relay opens a new bidirectional stream between the client and
the host. The two sides immediately start the Noise handshake. The relay
forwards bytes and never inspects them.

### Relay visibility

The relay can see:

- host IDs and client IP addresses,
- connection timing, duration and volume,
- which client talks to which host.

The relay must not see destinations, payloads, credentials or invite tokens.
It must not persist per-connection logs by default.

### Abuse controls

- Per-IP connection and bandwidth limits.
- Per-host registration limits.
- Idle and maximum connection lifetimes.
- Optional allowlist of host IDs in private deployments.

## Pairing and invites

An invite is a URI rendered as a QR code or a share link:

```
munna://v1/<host_id>@<host>:<port>
  ?relay=<relay-url>
  &fp=<fingerprint>
  &t=<token>
  &n=<display-name>
```

| Field | Meaning |
| --- | --- |
| `host_id` | Base32 (no padding) of the first 16 bytes of SHA-256(host static key) |
| `host:port` | Direct LAN or WAN endpoint (optional if relay is set) |
| `relay` | Relay URL (optional) |
| `fp` | Base32 SHA-256 fingerprint of the host static key, for pinning |
| `t` | Single-use pairing token, 32 random bytes, base64url |
| `n` | Human-readable host name |

### First contact

1. Client connects (LAN, direct WAN, or relay bridge).
2. Client and host run Noise **XX** with the pairing token mixed in as a
   pre-shared key.
3. The host checks the token, adds the client static key to its allowlist, and
   invalidates the token.
4. The session continues; subsequent connections use Noise **IK**.

Tokens expire after 10 minutes by default and are invalidated on first use.
Revoking a client removes its key from the host allowlist.

## Error codes

| Code | Name | Meaning |
| --- | --- | --- |
| `0x00` | OK | Success |
| `0x01` | PROTOCOL_ERROR | Malformed frame or unsupported version |
| `0x02` | AUTH_FAILED | Bad credentials or token |
| `0x03` | NOT_PAIRED | Client key is not allowlisted |
| `0x04` | POLICY_DENIED | Target denied by policy |
| `0x05` | HOST_UNREACHABLE | DNS or routing failure on the host |
| `0x06` | TARGET_REFUSED | Target refused the connection |
| `0x07` | TIMEOUT | Dial or idle timeout |
| `0x08` | HOST_OFFLINE | Relay has no live registration |
| `0x09` | RATE_LIMITED | Too many requests |
| `0x0A` | INTERNAL | Unexpected host error |

## Versioning

- Protocol v1 accepts additive frames and fields only.
- A breaking change bumps the ALPN token to `munna/2`.
- `HELLO` carries the version so peers can fail fast with `PROTOCOL_ERROR`.
- Invite URIs carry `v1` in the path.

## Open questions

- Whether UDP through the relay should use QUIC datagrams end to end or a
  stream-based encapsulation for networks that throttle UDP.
- MTU discovery strategy for tunnels over IPv6-only mobile networks.
- Whether to support multiple simultaneous relays per host.
- Token binding to the client's first static key versus a general PSK.
