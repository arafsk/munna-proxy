// Package tun owns the client-side virtual interface: creating the TUN
// device, installing routes, intercepting DNS, and tearing everything down
// cleanly on exit or crash.
//
// Device creation requires elevation. The privileged part lives in
// cmd/munna-helper and is reached over a narrow local API; this package
// contains the unprivileged orchestration and the tun2socks integration.
//
// TODO(M2): macOS utun, routes, DNS and kill switch.
// TODO(M4): Windows wintun and Linux tun backends.
// TODO(M5): Android VpnService file-descriptor handoff.
package tun
