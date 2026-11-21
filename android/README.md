# Android application

Status: **planned for M5**. This directory will hold the Kotlin + Jetpack
Compose app and the gomobile-built engine archive.

## Stack

- Kotlin, Jetpack Compose, minSdk 26 (Android 8), targetSdk current.
- Engine compiled to an AAR with `gomobile bind`.
- `VpnService` owns the TUN file descriptor; the Go engine runs tun2socks and
  the tunnel client.

## Intended layout

```
android/
├── settings.gradle.kts
├── build.gradle.kts
├── app/
│   ├── build.gradle.kts
│   ├── src/main/AndroidManifest.xml
│   ├── src/main/java/...   # UI, VpnService, QR scanner
│   └── libs/munna.aar      # generated, not committed
└── gradle/
```

## Building the engine archive

```bash
go install golang.org/x/mobile/cmd/gomobile@latest
gomobile init
gomobile bind -target=android/arm64,android/arm,android/amd64 \
  -o android/app/libs/munna.aar ./mobile
```

`android/app/libs/munna.aar` is generated and ignored by git.

## VpnService flow

1. Request consent with `VpnService.prepare`.
2. Build the interface: `addAddress`, `addRoute("0.0.0.0", 0)`,
   `addRoute("::", 0)`, `addDnsServer`, `setMtu`.
3. Call `establish()` to obtain the file descriptor.
4. Pass the descriptor to the Go engine, which starts tun2socks and connects to
   the host.
5. Run a foreground service with a persistent notification.
6. On stop, close the descriptor and tear down routes.

Use `setBlocking(true)`, `setUnderlyingNetworks`, and per-app rules via
`addAllowedApplication` / `addDisallowedApplication` for split tunneling.

## Permissions

- `android.permission.INTERNET`
- `android.permission.FOREGROUND_SERVICE`
- `android.permission.POST_NOTIFICATIONS` (Android 13+)
- `BIND_VPN_SERVICE` on the service declaration
- `android:usesCleartextTraffic="false"` unless a LAN-only build needs it

## Behaviour notes

- Only one VPN can be active at a time; detect and explain conflicts.
- Offer always-on VPN with "Block connections without VPN" as the kill switch.
- Request a battery-optimization exemption; doze can kill long-lived tunnels.
- Keep the notification actionable (disconnect, status, traffic counters).

## Build

```bash
cd android
./gradlew assembleDebug      # debug APK
./gradlew assembleRelease    # self-signed release APK (until a Play account exists)
```

## Testing

- Emulator for UI and permission flows.
- A physical device for real network handoff (Wi-Fi to cellular), doze and
  battery behaviour.
- Verify that a proxy-blind app (for example a game or a native client) is
  captured by the tunnel.
