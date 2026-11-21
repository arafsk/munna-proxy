# Desktop application

Status: **planned for M3**. The Wails project will be generated here; the
directory is intentionally empty until then so the Go module and CI stay
focused on the engine.

## Stack

- [Wails v2](https://wails.io) — Go backend, web frontend.
- The backend imports `core/api` and links the engine in-process.
- The privileged helper (`cmd/munna-helper`) is a separate binary installed by
  the app; the UI never runs with elevated rights.

## Intended layout

```
desktop/
├── main.go          # Wails bootstrap
├── app.go           # UI-facing methods, binds core/api
├── wails.json
├── frontend/        # web UI (framework chosen at M3)
└── build/           # icons, manifests, platform bundles
```

## Integration notes

- Host mode: start/stop the SOCKS5 server, show the invite QR, list peers.
- Client mode: connect/disconnect, show status, traffic counters and logs.
- Helper: detect, install, update and remove; surface every elevation prompt
  explicitly.
- Permissions: macOS needs local-network usage strings; Windows needs a
  firewall rule for host mode.

## Development

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
wails doctor
wails dev
wails build
```

## Platform notes

- **macOS**: add `NSLocalNetworkUsageDescription` and `NSBonjourServices` to
  `build/darwin/Info.plist`. Development builds can be ad-hoc signed; release
  builds need notarization and a signed helper.
- **Windows**: WebView2 is required at runtime (present on Windows 11). Ship
  `wintun.dll` for amd64 and arm64. If native arm64 packaging lags, ship x64
  and rely on emulation.
- **Linux**: requires `libgtk-3` and `libwebkit2gtk-4.1` at build and runtime.
