// Package tunnel implements the Munna transport: a QUIC connection that
// carries control frames, one stream per proxied TCP connection, and QUIC
// datagrams for proxied UDP.
//
// The transport is deliberately unaware of SOCKS5 semantics; core/socks5 and
// core/tun translate between their protocols and the frames defined here.
// See docs/protocol.md for the wire format.
//
// TODO(M6): ALPN negotiation, control stream, stream-per-connection.
// TODO(M6): UDP datagram framing and association IDs.
// TODO(M6): TCP/443 fallback for restrictive networks.
package tunnel
