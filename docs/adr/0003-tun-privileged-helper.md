# ADR 0003: TUN is managed by a narrow privileged helper

- Status: Accepted
- Date: 2026-11-21
- Related: [threat model](../threat-model.md), [development](../development.md)

## Context

Whole-device mirroring requires a TUN device, routes and DNS changes. Those
operations need root on macOS and Linux, and LocalSystem on Windows. The GUI
and the engine should run unprivileged so that a bug in the UI cannot become a
local privilege escalation.

## Decision

Split the privileged work into a separate `munna-helper` binary:

- The helper creates and configures the TUN device, installs routes and sets
  DNS, then hands the file descriptor or device name to the unprivileged
  client.
- The helper exposes a fixed-schema API over a unix socket (macOS/Linux) or a
  named pipe (Windows).
- Requests are authenticated by peer credentials and validated field by field;
  the helper never executes shell commands.
- Installation differs per platform: `SMAppService` on macOS, a Windows
  service, a systemd unit plus polkit on Linux.
- On Android there is no helper; `VpnService` provides the file descriptor to
  the app directly.

## Consequences

### Positive

- The UI and engine stay unprivileged.
- The attack surface is a small, auditable API rather than a root GUI.
- Platform-native installation and consent flows.
- The same helper interface works for development and signed releases.

### Negative

- Distribution requires signing and notarization to avoid friction.
- Each platform needs its own installer and lifecycle code.
- A helper bug is still a local privilege escalation, so it needs fuzzing and
  review.
- Development requires an admin prompt on every machine.

## Alternatives considered

| Alternative | Why not |
| --- | --- |
| Run the whole app as root | Unacceptable attack surface |
| Setuid binary | Fragile, platform-specific, easy to get wrong |
| macOS Network Extension | Requires a paid Apple account and entitlements; still needs a helper-like design |
| Userspace networking only | Cannot capture arbitrary applications |
| Kernel driver | Far more work and signing burden; wintun and utun are sufficient |
