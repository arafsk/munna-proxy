// Package expose publishes selected host-local ports to paired clients.
//
// A proxy can reach 127.0.0.1 on the host, but a remote application that
// hard-codes 127.0.0.1 talks to itself. Expose closes that gap by listening
// on the LAN (optionally behind TLS and authentication) and forwarding to a
// host loopback port. The magic name *.host.munna provides the same mapping
// for clients that resolve names through the tunnel.
//
// TODO(M2): TCP listener with auth and idle timeouts.
// TODO(M2): wildcard hostname mapping through the client resolver.
package expose
