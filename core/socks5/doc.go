// Package socks5 implements the host side of SOCKS5 (RFC 1928) with
// username/password authentication (RFC 1929).
//
// The server is the host-side entry point. It accepts CONNECT requests from
// LAN clients and from the Munna tunnel, and dials targets using the host's
// network stack. A client-side shim for tun2socks is planned.
//
// Surface:
//
//	Server{Config}        // Listen, ListenAndServe, Serve
//	Authenticator         // pluggable, constant-time comparison
//	StaticAuth            // generated host credentials
//	Dialer                // host resolver, policy and happy eyeballs in core/dialer
//
// TODO(M2): UDP ASSOCIATE and association lifetimes.
// TODO(M2): client side for the local tun2socks shim.
package socks5
