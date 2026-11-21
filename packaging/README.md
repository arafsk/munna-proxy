# Packaging and distribution

Status: **partially in place**. GoReleaser produces raw binaries; installers
and app bundles arrive with the desktop and Android milestones.

## Artifacts

| Platform | Artifact | Milestone | Signing |
| --- | --- | --- | --- |
| macOS (universal) | `.app` in `.dmg` | M3 | Ad-hoc for dev, Developer ID later |
| Windows (amd64, arm64) | `.msi` or NSIS `.exe` | M4 | Unsigned, Authenticode later |
| Linux (amd64, arm64) | `.deb` and AppImage | M4 | Unsigned |
| Android | `.apk` | M5 | Self-signed, Play later |
| Relay | Linux binary + systemd unit | M6 | Not applicable |
| CLI | `.tar.gz` / `.zip` | M1 | Checksums |

## Current release flow

```bash
make snapshot          # local build into dist/, no publishing
git tag v0.1.0
git push origin v0.1.0 # triggers .github/workflows/release.yml
```

GoReleaser builds `munna`, `munna-relay` and `munna-helper` for darwin,
windows and linux on amd64 and arm64, writes `checksums.txt`, and creates a
draft GitHub release.

## Install notes for unsigned builds

- **macOS**: right-click the app and choose Open, or remove the quarantine
  attribute: `xattr -dr com.apple.quarantine /Applications/Munna.app`.
- **Windows**: SmartScreen shows "Windows protected your PC"; choose "More
  info" then "Run anyway".
- **Android**: allow installation from the browser or file manager used to
  download the APK.

These steps disappear once the project has signing certificates.

## Signing roadmap

1. Apple Developer ID: sign the app and helper with the same Team ID, enable
   the hardened runtime, notarize and staple.
2. Windows Authenticode: sign the installer and binaries.
3. Android: generate a release keystore, keep it offline, and enroll in Play
   App Signing if publishing.
4. Relay: TLS certificates from a public CA and a documented update process.

## Relay deployment

The relay will ship as a single Linux binary plus a systemd unit. Planned
defaults:

- QUIC on UDP 443, TCP/TLS fallback on 443.
- No destination logging; only aggregate health metrics.
- Per-IP connection and bandwidth limits.
- Configuration through a single file, no secrets in flags.
