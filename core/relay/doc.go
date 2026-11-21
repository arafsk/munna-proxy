// Package relay contains both sides of the relay protocol.
//
// The host keeps an outbound control connection to the relay and registers
// its host ID. A client asks the relay for a host ID; the relay bridges a
// stream between them. Authorization is decided by the host, not the relay.
//
// TODO(M6): host registration, heartbeats and reconnects.
// TODO(M6): client rendezvous and stream bridging.
// TODO(M6): rate limits, connection caps and abuse controls.
package relay
