// Package api is the control plane used by the desktop and Android
// applications and by the CLI.
//
// A Controller owns one host session (SOCKS5 server, mDNS advertisement and
// invite) and one client session (local SOCKS5 forwarder that chains through a
// host). It runs in-process: the desktop app binds it directly, and Android will
// bind it through gomobile. Nothing here opens a network port of its own.
//
// TODO(M3): a socket API for a privileged helper, with peer credential checks.
// TODO(M5): Android binding through gomobile.
package api
