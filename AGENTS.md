# AGENTS.md

Munna Proxy turns a host device's network into a SOCKS5 proxy. The engine is Go. Desktop is Wails v2 with a static web UI. Android is not started. Roadmap: `docs/roadmap.md`. M1 (SOCKS5 core) is done; the current work is a test GUI on top of `core/api`.

## Layout

- Module: `github.com/arafsk/munna-proxy`. Go 1.27.4 (`go.mod`).
- `cmd/munna`: CLI. Only `serve` is implemented. The usage text also lists `connect`, `expose` and `status`, which don't exist yet.
- `cmd/munna-relay`, `cmd/munna-helper`: 19-line stubs.
- `core/socks5`: RFC 1928/1929 server (`server.go`), host-side policy hooks, and the client side used by the local forwarder (`client.go`: `Upstream`, `Chain`).
- `core/dialer`: host-side dialing with metadata-endpoint denial. Loopback is allowed on purpose.
- `core/discovery`: invites (`munna://`), mDNS advertise and browse, QR output.
- `core/api`: `Controller` owns one host session and one client session. The CLI and the desktop app both use it. Keep host/client logic here, not in `cmd/` or `desktop/`.
- `core/config`, `core/expose`, `core/noise`, `core/relay`, `core/tun`, `core/tunnel`: doc-only placeholders.
- `mobile/`: gomobile entry point for Android (not built yet).
- `desktop/`: a **separate Go module** (`replace github.com/arafsk/munna-proxy => ../`). Root `go build ./...`, `go test ./...`, `make ci` and CI never touch it.
- `android/`: README placeholder only.
- `internal/version`: set via `-X` ldflags in `Makefile` and `.goreleaser.yaml`.

## Commands

- Before committing: `make ci`. It runs gofmt check, `go vet`, golangci-lint v2, and `go test -race`. `scripts/check.sh` runs the same steps.
- Single package or test: `go test -run TestName -v ./core/socks5`.
- Fuzz (targets are `FuzzReadGreeting`, `FuzzReadAuth`, `FuzzReadRequest` in `core/socks5/socks5_test.go`): `go test -fuzz=FuzzReadRequest -fuzztime=30s ./core/socks5`.
- Desktop vet: `cd desktop && go vet ./...`. Desktop has no tests yet.
- Desktop builds:
  - `desktop/scripts/build-local.sh`: macOS universal and Windows amd64/arm64. Needs the Wails CLI at `$(go env GOPATH)/bin/wails`, v2.17.1.
  - `desktop/scripts/build-linux.sh [amd64|arm64]`: runs in Docker (`desktop/Dockerfile.linux`). It must use `-tags webkit2_41` because Ubuntu 24.04 ships only WebKitGTK 4.1.
  - Output goes to `desktop/build/bin/`, which is git-ignored.
- Wails bindings: after changing exported methods in `desktop/app.go` or the `core/api` types, run `cd desktop && wails generate module`. It writes to `desktop/frontend/wailsjs/`, and those files are committed.
- Two-device LAN test: `test/e2e/run-docker.sh`. It needs Docker and internet access (it fetches example.com). Expected result: `passed=8 failed=0`.
- Cross-compile check: `make cross`.
- Release: pushing a `v*` tag runs `.github/workflows/release.yml` (GoReleaser). `.goreleaser.yaml` sets `draft: true`. Don't create a tag or publish a release without the owner's explicit go-ahead.

## Gotchas

- Some docs describe features that don't exist. Don't assume they're present: `--config` flags, `test/fuzz`, `test/integration`, `test/platform`, `-tags=e2e` suites, `munna connect`, and the relay local-testing commands in `docs/development.md`. There are no `go:build` tags in the repo yet.
- Engine binaries must build with `CGO_ENABLED=0`. Android via gomobile is the only exception, and it needs cgo and the NDK.
- Windows CI runs `go test` without `-race`.
- golangci-lint (`.golangci.yml`) enables gosec, gocritic, errcheck, noctx, unparam and revive, and lints tests too. Patterns that have failed before:
  - Unchecked `fmt.Fprintf` or `Fprintln`. Build a `strings.Builder` and use `io.WriteString`, or assign the error to `_`.
  - `byte(len(x))` triggers G115. Use the `lengthByte` helper in `core/socks5/client.go`.
  - `net.Listen` in tests triggers noctx. Use `net.ListenConfig` with a context.
- `Controller.StartHost` accepts port `0` (an OS-chosen port). Tests rely on it, and the invite uses the real bound port.
- Security invariants in `core/api` and `core/socks5`:
  - `NoAuth` is allowed only on loopback listeners.
  - The client forwarder must bind a loopback address.
  - Passwords and invite tokens must never reach log output. `controller_test.go` checks this.
  - Use `log/slog`, and log destinations at debug level only.
- Desktop frontend is plain HTML/JS with no npm or build step. `desktop/frontend/index.html` has a CSP. Keep new code compatible with it. The GUI does not persist passwords (only host and username go to localStorage).
- `desktop/build/darwin/Info.plist` and `Info.dev.plist` are Go text/templates (`{{.Info.ProductName}}`), not plain plists. Both must keep `NSLocalNetworkUsageDescription` and `NSBonjourServices` (`_munna-proxy._tcp`), or mDNS fails silently on macOS.
- Both Windows targets default to `munna-proxy.exe`, so always pass `-o` with an arch suffix. `build-local.sh` already does.
- After changing `desktop/go.mod` or the engine's API surface, run `cd desktop && go mod tidy`. The CI tidy check does not cover `desktop/`.
- `desktop/build/bin` and `dist/` are ignored. Never commit binaries, `.app` bundles, keys or signing material.

## Process

- Commit directly to `main`. The owner's instruction overrides the branch workflow in `CONTRIBUTING.md`.
- Use Conventional Commits, for example `feat(api): ...` or `test: ...`.
- Commit tests with the code they cover. Publishing and tagging need explicit approval.
- Note user-visible changes in `CHANGELOG.md`. It has no `Unreleased` section yet, so add one.
- `origin` is `git@github.com:arafsk/munna-proxy.git` (SSH). Push with `git push origin main`.
- Platform work must say which OS was tested. Desktop GUI and Windows/Linux runtime are currently untested, and so is the macOS window.
