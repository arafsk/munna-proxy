// Package dialer owns every outbound connection made by the host.
//
// It resolves names with the host's resolver, applies happy-eyeballs
// dialing, enforces policy (denied ports, loopback rules, cloud metadata
// endpoints), and records per-peer metrics without logging destinations at
// info level.
//
// TODO(M1): resolver configuration and dual-stack dialing.
// TODO(M2): per-peer connection limits and timeouts.
// TODO(M6): dial through a bound interface for split-tunnel scenarios.
package dialer
