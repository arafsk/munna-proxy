// Package mobile is the gomobile entry point for the Android application.
//
// Build the AAR with:
//
//	gomobile bind -target=android/arm64,android/arm,android/amd64 \
//	    -o android/app/libs/munna.aar ./mobile
//
// The Android VpnService creates the TUN file descriptor and hands it to
// this package, which runs tun2socks and the tunnel client. Keep the exported
// surface small and JSON-friendly; gomobile does not support every Go type.
package mobile

import "github.com/arafsk/munna-proxy/internal/version"

// Version returns the engine version for display in the Android UI.
func Version() string {
	return version.Version
}
