# ADR 0004: Static keys, Noise handshakes and QR pairing

- Status: Accepted
- Date: 2026-11-21
- Related: [protocol](../protocol.md), [threat model](../threat-model.md)

## Context

Hosts and clients must authenticate each other without a central account
system. The relay is untrusted for confidentiality and must not be able to
impersonate either side. Pairing has to be easy enough to do by scanning a QR
code, and secure enough to survive a hostile LAN.

## Decision

- Every installation generates a static X25519 keypair, stored in the OS key
  store.
- The host ID is the first 16 bytes of SHA-256(host public key), encoded in
  base32 without padding.
- An invite URI contains the host ID, endpoints, relay URL, a key fingerprint
  and a single-use pairing token; it is rendered as a QR code.
- First contact uses Noise **XX** with the pairing token mixed in as a
  pre-shared key. The host verifies the token, adds the client key to its
  allowlist and invalidates the token.
- Subsequent connections use Noise **IK** with pinned static keys.
- Revocation removes the client key from the host allowlist.

## Consequences

### Positive

- No accounts, no central identity provider, no passwords to share.
- Mutual authentication: the client pins the host, the host allowlists the
  client.
- The relay cannot impersonate either side or decrypt traffic.
- QR pairing is fast and works offline on a LAN.
- Keys never touch disk in plaintext.

### Negative

- Losing a device means re-pairing; there is no account recovery.
- Users must notice and act on fingerprint changes.
- Key storage differs per platform and needs a common interface.
- Token handling is security-critical and needs tests.

## Alternatives considered

| Alternative | Why not |
| --- | --- |
| Shared passwords only | No mutual identity; weak against phishing and replay |
| TLS client certificates | Workable, but certificate lifecycle and pinning are heavier than Noise keys |
| OAuth or a cloud account | Adds a central trust authority and a privacy dependency |
| PAKE (SPAKE2/OPAQUE) | Attractive for password-based pairing, but more complex than a random token |
| Trust on first use without a token | Vulnerable to a first-contact MITM on a hostile LAN |
