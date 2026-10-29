# Contributing

## Set up

You need Go 1.26 (the version in `go.mod` and `.tool-versions`), Git and Make.

```bash
git clone https://github.com/techishthoughts-org/gitshift.git
cd gitshift
make test    # go test -v ./...
make build   # writes ./gitshift
./gitshift --version
```

`make install` copies the binary to `${GOPATH:-$HOME/go}/bin`. `go install` from the module path does not work, because the module path (`github.com/techishthoughts/gitshift`) differs from the repository (`techishthoughts-org/gitshift`).

Other targets: `make fmt`, `make vet`, `make lint` (needs golangci-lint), `make help`.

## Never touch your real identity in tests

Switching changes your global Git config, `~/.ssh/config` and the GitHub CLI login. Tests must use a temporary `HOME`, a temporary config file (`--config`) and fake executables. Do not run `switch`, `auto-switch` or the shell hook against your own configuration while developing; use `--config` with a scratch file and a scratch `HOME`.

## Before opening a pull request

```bash
make fmt
make vet
go test -race ./...
```

- **Changed a command, flag or help text?** Run `make docs` and commit the regenerated `docs/reference/commands/`. CI fails if it is stale. Never edit those files by hand.
- **User-visible change?** Add an entry under `## [Unreleased]` in [CHANGELOG.md](CHANGELOG.md), and update the relevant page under [docs/](docs/README.md).
- Add tests for the success path and the failure paths. Switch changes need tests for preflight, mutation order, rollback order, and persisting `current_account` only after success.

## Commits and branches

- Work on a feature branch, never directly on `main`.
- Commit messages use `type(scope): description` with an emoji, for example `🐛 fix(ssh): keep global options above the managed block` or `✨ feat(auto): add default account fallback`.

## Releases

Maintainers release through a single workflow. See [Releasing](docs/maintainers/releasing.md).

## Reporting security issues

Report suspected vulnerabilities privately to the maintainers instead of opening a public issue. Never include private keys or tokens in a report.
