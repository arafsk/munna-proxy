// Package api is the local control plane used by the desktop and Android
// applications.
//
// It listens on a unix socket (macOS/Linux) or named pipe (Windows) and
// speaks a small JSON/HTTP API authenticated with a token that only the
// current user can read. It must never be exposed on a network interface.
//
// Planned endpoints:
//
//	GET  /status
//	POST /host/start,  POST /host/stop
//	POST /client/connect, POST /client/disconnect
//	GET  /peers, POST /pair
//	GET  /logs/stream        (server-sent events)
//
// TODO(M3): transport, authentication and lifecycle.
// TODO(M5): Android in-process binding instead of a socket.
package api
