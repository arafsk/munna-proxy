# Security policy

Munna Proxy exposes network services and installs a privileged helper on client
devices. Security reports are taken seriously.

## Reporting a vulnerability

**Do not open a public issue.**

Use GitHub's private vulnerability reporting:

<https://github.com/arafsk/munna-proxy/security/advisories/new>

Include:

- affected component and version or commit,
- a minimal reproduction or proof of concept,
- the impact you believe it has,
- any suggested mitigation.

You will receive an acknowledgement as soon as possible. This is a
pre-release project maintained on a best-effort basis, so no fixed response
deadline is offered yet.

## Supported versions

No version is supported for production use yet. Security fixes target `main`
and the most recent pre-release tag, if any.

## Scope

In scope:

- the SOCKS5 server and client (parsing, authentication, policy),
- the tunnel and relay protocols,
- device identity, pairing and key handling,
- the privileged TUN helper and its local API,
- TUN configuration, routing, DNS handling and kill-switch behaviour,
- installers and update paths.

Out of scope:

- vulnerabilities in the host operating system or third-party dependencies
  (report those upstream),
- attacks that require an already-compromised host or client machine,
- denial of service through sheer bandwidth exhaustion,
- anonymity guarantees (this is not a Tor-like system).

## Hardening expectations

The project aims to ship with:

- authentication enabled by default, with no anonymous mode,
- constant-time credential comparison and auth rate limiting,
- encrypted transports (TLS for direct connections, end-to-end Noise through
  the relay),
- a minimal privileged helper that never executes shell commands,
- no destination logging on the relay,
- fuzzed parsers and a documented threat model.

See [docs/threat-model.md](docs/threat-model.md) for the current analysis.
