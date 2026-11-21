// Package socks5 implements the server and client sides of SOCKS5
// (RFC 1928) with username/password authentication (RFC 1929).
//
// The server is the host-side entry point. It accepts CONNECT and UDP
// ASSOCIATE requests from LAN clients and from the Munna tunnel, and dials
// targets using the host's network stack. The client side is used by the
// local shim behind tun2socks.
//
// Planned surface:
//
//	Server{Config}        // ListenAndServe, Close, Addr
//	Request, Reply        // wire types
//	Authenticator         // pluggable, constant-time comparison
//	Policy                // allow/deny rules, loopback and metadata guards
//
// TODO(M1): CONNECT, domain-name ATYP, IPv4 and IPv6, reply codes.
// TODO(M2): UDP ASSOCIATE and association lifetimes.
// TODO(M6): fuzz the request parser and add auth rate limiting.
package socks5
