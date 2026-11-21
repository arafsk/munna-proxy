# ADR 0005: Wails for desktop, Kotlin/Compose for Android

- Status: Accepted
- Date: 2026-11-21
- Related: [development](../development.md)

## Context

The engine is written in Go and must ship with a desktop GUI on macOS, Windows
and Linux, plus a separate Android app. The desktop GUI needs to talk to the
engine and to a privileged helper. The Android app needs tight control over
`VpnService`, foreground services, notifications and QR scanning.

## Decision

- **Desktop:** Wails v2. The Go backend links the engine directly; the
  frontend is standard web technology. One language for the core and the shell,
  and no IPC between the UI and the engine on desktop.
- **Android:** Kotlin with Jetpack Compose, calling a gomobile-built AAR. Native
  control over `VpnService` and platform lifecycle.
- **Shared contract:** both frontends talk to `core/api` (in-process on
  desktop, direct calls on Android) so behaviour and state stay identical.

## Consequences

### Positive

- Desktop development stays in Go; no second backend language.
- Android gets first-class platform integration without a bridge layer.
- The engine remains independent of either UI.
- Wails produces small binaries compared with Electron.

### Negative

- Two UI codebases to maintain.
- Wails support for Windows arm64 may lag; the fallback is an x64 build under
  emulation.
- gomobile has a limited type surface and produces a large AAR.
- Design consistency must be maintained by convention, not shared components.

## Alternatives considered

| Alternative | Why not |
| --- | --- |
| Flutter everywhere | Adds a bridge for `VpnService` and uncertain Windows arm64 desktop support |
| Tauri 2 | Strong platform reach, but adds Rust and a sidecar IPC layer |
| Electron | Large binaries, heavy runtime, no advantage here |
| Fully native per platform | Three desktop UIs plus Android; too much duplication |
| Fyne | Pure Go, but weak desktop polish and limited Android integration |
