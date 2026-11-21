# Changelog

All notable changes to this project are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and the project intends to use [Semantic Versioning](https://semver.org/)
once it reaches a release.

## [0.1.0] - 2026-11-21

First release: the host side of the SOCKS5 proxy.

### Added

- Repository scaffold: module layout, package documentation, build tooling.
- Architecture, protocol, threat model, roadmap and development docs.
- Architecture decision records for the SOCKS5 interface, QUIC relay,
  privileged helper, identity model and UI stack.
- GitHub Actions CI (build, vet, test, lint, cross-compile) and a GoReleaser
  release workflow.
- Issue and pull request templates, security policy, contributing guide.
- `core/socks5`: RFC 1928 CONNECT for domain, IPv4 and IPv6 targets, RFC 1929
  authentication with constant-time comparison, per-remote lockout after
  repeated failures, reply-code mapping, idle timeouts and connection caps.
- `core/dialer`: host-side DNS, happy-eyeballs racing, metadata-endpoint denial
  and pluggable policy.
- `core/discovery`: `munna://` invite encoding, mDNS advertisement and a
  terminal QR code.
- `munna serve`: host mode with generated credentials, invite output and
  configurable listen, advertise, mDNS and logging flags.
- Unit, integration and fuzz tests for the SOCKS5 parser, dialer and invites.

### Changed

- Replaced the placeholder README with project documentation.

### Security

- Documented the initial threat model and security reporting process.

[0.1.0]: https://github.com/arafsk/munna-proxy/releases/tag/v0.1.0
