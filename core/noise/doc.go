// Package noise implements device identity and the end-to-end handshake
// that runs between client and host inside a relayed connection.
//
// The relay terminates QUIC, so a second, inner handshake is required to
// keep the relay zero-knowledge. First contact uses Noise XX authenticated
// by a single-use pairing token; subsequent connections use Noise IK with
// pinned static keys.
//
// TODO(M6): static key generation and secure storage integration.
// TODO(M6): XX and IK handshakes with token binding.
// TODO(M6): key rotation and revocation.
package noise
