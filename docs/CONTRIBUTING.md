# Contributing

## Set up

The repository requires the Go version declared in `go.mod` (currently Go 1.26), Git, and Make for convenience targets.

```bash
git clone https://github.com/techishthoughts-org/gitshift.git
cd gitshift
go mod download
make test
make build
```

Run the development binary as `./gitshift`. `make install` copies it to `${GOPATH:-$HOME/go}/bin`.

## Before opening a pull request

```bash
make fmt
make vet
go test -race ./...
go build ./...
```

If you change user-visible behavior:

- update root help and subcommand help;
- update the README and the relevant guide;
- keep examples limited to commands that actually exist;
- add or update tests for successful and failing paths;
- check links and release asset names.

The CLI's implementation and tests are authoritative. Generated help must also be reviewed because a help string can be stale.

## Change safety

Account switching mutates Git, SSH, GPG, and possibly GitHub CLI state. Changes to it need tests for preflight validation, mutation order, rollback order, persistence after success only, force behavior, and interruption limitations. Never use real credentials or a developer's actual home configuration in tests.

Configuration writes must preserve owner-only permissions, atomic replacement, concurrent-writer detection, and platform-specific replacement semantics.

Auto-switch changes require canonical-path, symlink, component-boundary, deepest-match, missing-account, no-match, idempotence, transaction-lock, and generated-hook tests. Tests must use temporary configuration, home, Git config, and fake executables; never enable a real shell hook or switch the developer's actual account.

## Documentation checks

The documentation workflow validates Markdown links under `docs/` and checks required files. Also inspect root README links locally. When modifying GitHub Actions, update [the workflow inventory](../.github/workflows/README.md) and [release process](../.github/RELEASE_PROCESS.md).

## Releases

Release assets and checksums are produced by the manually dispatched release workflow, which validates an explicit version and then creates the tag. Do not claim support for an operating system or architecture until that workflow actually builds the asset. See [Release process](../.github/RELEASE_PROCESS.md).
