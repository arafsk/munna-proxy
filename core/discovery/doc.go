// Package discovery advertises hosts on the local network and encodes
// pairing information for the GUIs.
//
// LAN discovery uses mDNS (_munna-proxy._tcp). Pairing uses a compact
// invite URI that can be rendered as a QR code, including the host ID,
// endpoints, relay URL and a single-use token.
//
// TODO(M1): mDNS advertise and browse.
// TODO(M1): invite URI parsing and QR rendering.
// TODO(M6): relay endpoint selection and expiry.
package discovery
