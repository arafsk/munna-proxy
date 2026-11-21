# Contributing

Thanks for your interest. The project is at an early scaffold stage, so the
most useful contributions right now are design review, protocol feedback and
platform research.

## Before you start

- Read [docs/architecture.md](docs/architecture.md),
  [docs/protocol.md](docs/protocol.md) and
  [docs/threat-model.md](docs/threat-model.md).
- For anything larger than a bug fix, open an issue first so the design can be
  agreed on before code is written.
- Security issues go through [SECURITY.md](SECURITY.md), never a public issue.

## Development setup

See [docs/development.md](docs/development.md). The short version:

```bash
make build
make ci
```

## Branch and commit conventions

- Branch from `main`; use a descriptive prefix such as `feat/`, `fix/`,
  `docs/` or `chore/`.
- Use [Conventional Commits](https://www.conventionalcommits.org/):
  `feat(socks5): add UDP associate`, `fix(tun): restore routes on exit`.
- Keep commits focused and the tree buildable at every commit.
- Do not commit generated binaries, keys, certificates, or local configuration.

## Pull requests

A pull request should:

- explain the problem and the chosen approach,
- include tests for new behaviour and regressions,
- pass `make ci` locally,
- update the relevant docs and, for user-visible changes, `CHANGELOG.md`,
- avoid unrelated refactors.

Platform-specific changes should state which operating systems were tested.
If you cannot test a platform, say so explicitly.

## Code guidelines

- Go 1.27+, `gofmt` and `golangci-lint` clean.
- Pass `context.Context` through all network operations; never dial or listen
  without a context or timeout.
- Wrap errors with `%w` and add context; do not log and return the same error.
- Use `log/slog` for structured logging. Never log credentials, keys, invite
  tokens, or full destination addresses at info level.
- Keep packages small and dependency-injected; no mutable global state.
- Table-driven tests; fuzz tests for every wire parser.
- Prefer the standard library. New dependencies need a justification in the
  pull request, a compatible license, and no cgo unless unavoidable.

## License

By contributing you agree that your contribution may be distributed under the
project's current license terms (see [LICENSE](LICENSE)). Because the license
is still to be determined, significant contributions should be discussed with
the maintainer first.
